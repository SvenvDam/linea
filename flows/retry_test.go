package flows

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/retry"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

// Define a testData type for use in the tests
type testData struct {
	sourceInvocations []time.Time
}

// Define a type for context keys to avoid string collisions
type contextKey string

// Keys for context values
const (
	cancelFuncKey contextKey = "cancelFunc"
)

func TestRetry(t *testing.T) {
	// Define test cases
	tests := []struct {
		name         string
		setupRetry   func() *retry.Config
		createSource func() (*core.Source[int], *atomic.Int32, any)
		setupContext func() (context.Context, context.CancelFunc)
		runTest      func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any)
	}{
		{
			name: "successful processing without errors",
			setupRetry: func() *retry.Config {
				return retry.NewConfig(time.Millisecond, 10*time.Millisecond, 0, retry.WithMaxRetries(3))
			},
			createSource: func() (*core.Source[int], *atomic.Int32, any) {
				// Simple slice source with no errors
				input := []int{1, 2, 3}
				return sources.Slice(input), nil, input
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			runTest: func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any) {
				input := data.([]int)

				// Create the stream
				stream := compose.SourceThroughFlowToSink(
					source,
					Retry[int](retryConfig),
					sinks.Slice[int](),
				)

				// Run the stream
				result := <-stream.Run(ctx)

				// Verify the result
				assert.NoError(t, result.Err)
				assert.Equal(t, input, result.Value)
			},
		},
		{
			name: "retries until success",
			setupRetry: func() *retry.Config {
				return retry.NewConfig(time.Millisecond, 10*time.Millisecond, 0, retry.WithMaxRetries(3))
			},
			createSource: func() (*core.Source[int], *atomic.Int32, any) {
				// Source that fails on first attempt but succeeds on second
				attempts := &atomic.Int32{}

				source := core.NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[int] {
						out := make(chan core.Item[int], 1)

						go func() {
							defer close(out)

							attemptCount := attempts.Add(1)

							// Fail on first attempt
							if attemptCount == 1 {
								out <- core.Item[int]{Err: errors.New("temporary error")}
								return
							}

							// Succeed on subsequent attempts
							out <- core.Item[int]{Value: 42}
						}()

						return out
					},
				)

				return source, attempts, []int{42}
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			runTest: func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any) {
				expectedOutput := data.([]int)

				// Create the stream
				stream := compose.SourceThroughFlowToSink(
					source,
					Retry[int](retryConfig),
					sinks.Slice[int](),
				)

				// Run the stream
				result := <-stream.Run(ctx)

				// Verify the result
				assert.NoError(t, result.Err)
				assert.Equal(t, expectedOutput, result.Value)
				assert.Equal(t, int32(2), sourceAttempts.Load(), "Expected exactly 2 source attempts")
			},
		},
		{
			name: "exhausts retries and propagates error",
			setupRetry: func() *retry.Config {
				return retry.NewConfig(time.Millisecond, 5*time.Millisecond, 0, retry.WithMaxRetries(2))
			},
			createSource: func() (*core.Source[int], *atomic.Int32, any) {
				// Source that always fails
				attempts := &atomic.Int32{}
				expectedError := errors.New("persistent error")

				source := core.NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[int] {
						out := make(chan core.Item[int], 1)

						go func() {
							defer close(out)
							attempts.Add(1)
							out <- core.Item[int]{Err: expectedError}
						}()

						return out
					},
				)

				return source, attempts, expectedError
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			runTest: func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any) {
				expectedError := data.(error)

				// Create the stream
				stream := compose.SourceThroughFlowToSink(
					source,
					Retry[int](retryConfig),
					sinks.Slice[int](),
				)

				// Run the stream
				result := <-stream.Run(ctx)

				// Verify the result
				assert.Error(t, result.Err)
				assert.Equal(t, expectedError, result.Err)
				assert.Equal(t, int32(3), sourceAttempts.Load(), "Expected 3 source attempts (initial + 2 retries)")
			},
		},
		{
			name: "respects context cancellation during backoff",
			setupRetry: func() *retry.Config {
				return retry.NewConfig(100*time.Millisecond, 500*time.Millisecond, 0, retry.WithMaxRetries(3))
			},
			createSource: func() (*core.Source[int], *atomic.Int32, any) {
				// Source that always fails
				attempts := &atomic.Int32{}

				source := core.NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[int] {
						out := make(chan core.Item[int], 1)

						go func() {
							defer close(out)
							attempts.Add(1)
							out <- core.Item[int]{Err: errors.New("error that triggers retry")}
						}()

						return out
					},
				)

				return source, attempts, nil
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			runTest: func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any) {
				// Create the stream
				stream := compose.SourceThroughFlowToSink(
					source,
					Retry[int](retryConfig),
					sinks.Slice[int](),
				)

				// Start the stream in a separate goroutine
				resultCh := stream.Run(ctx)

				// Wait a moment to ensure the retry backoff has started
				time.Sleep(10 * time.Millisecond)

				// Cancel the context during backoff
				cancel, ok := ctx.Value(cancelFuncKey).(context.CancelFunc)
				if ok {
					cancel()
				} else {
					t.Fatal("Failed to get cancel function from context")
				}

				// Get the result
				result := <-resultCh

				// Verify the context cancellation error was propagated
				assert.Error(t, result.Err)
				assert.ErrorIs(t, result.Err, context.Canceled)
				assert.Equal(t, int32(1), sourceAttempts.Load(), "Expected only 1 source attempt before cancellation")
			},
		},
		{
			name: "retries with exponential backoff",
			setupRetry: func() *retry.Config {
				minBackoff := 10 * time.Millisecond
				maxBackoff := 1 * time.Second
				return retry.NewConfig(minBackoff, maxBackoff, 0, retry.WithMaxRetries(3))
			},
			createSource: func() (*core.Source[int], *atomic.Int32, any) {
				// Track retry timing and source attempts
				attempts := &atomic.Int32{}

				// Create a shared slice that will store timestamps
				retryTimesData := &struct {
					times []time.Time
				}{
					times: []time.Time{time.Now()}, // Initial timestamp
				}

				source := core.NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[int] {
						out := make(chan core.Item[int], 1)

						go func() {
							defer close(out)
							attempts.Add(1)

							// Add the current time to our timestamps
							retryTimesData.times = append(retryTimesData.times, time.Now())

							out <- core.Item[int]{Err: errors.New("error to trigger retry")}
						}()

						return out
					},
				)

				return source, attempts, retryTimesData
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			runTest: func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any) {
				retryTimesData := data.(*struct {
					times []time.Time
				})

				// Create the stream
				stream := compose.SourceThroughFlowToSink(
					source,
					Retry[int](retryConfig),
					sinks.Slice[int](),
				)

				// Run the stream
				<-stream.Run(ctx)

				// Verify attempts match expected value
				assert.Equal(t, int32(4), sourceAttempts.Load(), "Expected 4 source attempts (initial + 3 retries)")

				// Verify we have the expected number of timestamps (may include initial + actual data points)
				// The first timestamp is our baseline before the test starts
				retryTimes := retryTimesData.times
				assert.GreaterOrEqual(t, len(retryTimes), 5, "Expected at least 5 timestamps (initial + 4 from source)")

				// We need at least 3 timestamps to verify backoff intervals (initial + 2 retries)
				if len(retryTimes) >= 3 {
					// Test that intervals are generally increasing
					// Use the last intervals to verify the pattern (in case we have more timestamps than expected)
					last := len(retryTimes)
					if last >= 4 {
						// Get the last 3 intervals
						interval1 := retryTimes[last-3].Sub(retryTimes[last-4])
						interval2 := retryTimes[last-2].Sub(retryTimes[last-3])
						interval3 := retryTimes[last-1].Sub(retryTimes[last-2])

						// Ensure intervals are generally increasing (not exact due to scheduling variance)
						assert.Less(t, interval1.Milliseconds(), interval3.Milliseconds(),
							"Expected later intervals to be larger than earlier ones")

						t.Logf("Backoff intervals: %v, %v, %v", interval1, interval2, interval3)
					}
				}
			},
		},
		{
			name: "maintains attempt counter across multiple errors",
			setupRetry: func() *retry.Config {
				return retry.NewConfig(1*time.Millisecond, 100*time.Millisecond, 0, retry.WithMaxRetries(3))
			},
			createSource: func() (*core.Source[int], *atomic.Int32, any) {
				// Track invocations and create test data
				data := &testData{
					sourceInvocations: []time.Time{},
				}

				source := core.NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[int] {
						out := make(chan core.Item[int], 3) // Buffer to hold multiple items

						go func() {
							defer close(out)

							// Record when this source was invoked
							now := time.Now()
							data.sourceInvocations = append(data.sourceInvocations, now)

							// Only send errors on first run
							if len(data.sourceInvocations) == 1 {
								// Send a value first
								out <- core.Item[int]{Value: 42}
								// Then send an error
								out <- core.Item[int]{Err: errors.New("first error")}
							} else {
								// Just succeed on second run
								out <- core.Item[int]{Value: 100}
							}
						}()

						return out
					},
				)

				return source, nil, data
			},
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			runTest: func(t *testing.T, source *core.Source[int], retryConfig *retry.Config, ctx context.Context, sourceAttempts *atomic.Int32, data any) {
				testData := data.(*testData)

				errorCount := 0
				successCount := 0

				// Create a flow to count errors and successes
				errorCountFlow := core.NewFlow(
					func(ctx context.Context, elem int, out chan<- core.Item[int]) core.StreamAction {
						successCount++
						out <- core.Item[int]{Value: elem}
						return core.ActionProceed
					},
					func(ctx context.Context, err error, out chan<- core.Item[int]) core.StreamAction {
						errorCount++
						out <- core.Item[int]{Err: err}
						return core.ActionProceed
					},
					nil, nil,
				)

				// Create the stream with the error counter flow
				stream := compose.SourceThroughFlowToSink2(
					source,
					errorCountFlow,
					Retry[int](retryConfig),
					sinks.Slice[int](),
				)

				// Run the stream
				result := <-stream.Run(ctx)

				// Verify counts
				assert.NoError(t, result.Err)
				assert.Equal(t, 1, errorCount, "Expected exactly 1 error processed")
				assert.Equal(t, 2, successCount, "Expected 2 success values (one from each source)")
				assert.Len(t, testData.sourceInvocations, 2, "Expected 2 source invocations")

				// Check we got expected values (42 from first source run, 100 from second)
				assert.Equal(t, []int{42, 100}, result.Value, "Expected successful values from both source invocations")

				// Verify timing - the second invocation should come after the first
				if len(testData.sourceInvocations) >= 2 {
					interval := testData.sourceInvocations[1].Sub(testData.sourceInvocations[0])
					assert.Positive(t, interval, "Expected positive interval between source invocations")
					t.Logf("Interval between source invocations: %v", interval)
				}
			},
		},
	}

	// Run all test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup retry config
			retryConfig := tt.setupRetry()

			// Create source
			source, sourceAttempts, testData := tt.createSource()

			// Setup context
			ctx, cancel := tt.setupContext()
			if cancel != nil {
				// Store cancel function in context to access it in runTest
				ctx = context.WithValue(ctx, cancelFuncKey, cancel)
				defer cancel()
			}

			// Run the test
			tt.runTest(t, source, retryConfig, ctx, sourceAttempts, testData)
		})
	}
}
