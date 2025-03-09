package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStream(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool)
		action      func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int])
		expectation func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool)
	}{
		{
			name: "happy path - processes single value",
			setup: func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool) {
				isSourceDone := &atomic.Bool{}
				sourceChan := make(chan Item[int], 1)

				stream := newStream(
					func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(sourceChan)
							defer isSourceDone.Store(true)

							select {
							case <-ctx.Done():
								return
							case <-complete:
								return
							case sourceChan <- Item[int]{Value: 42}:
								// Value sent successfully
							}
						}()
						return sourceChan
					},
				)

				return stream, sourceChan, isSourceDone
			},
			action: func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int]) {
				// No additional action needed, the setup will send the value
			},
			expectation: func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool) {
				result := <-res
				assert.Equal(t, 42, result.Value)
				assert.NoError(t, result.Err)

				// Channel should be closed after the value
				_, ok := <-res
				assert.False(t, ok, "output channel should be closed after result is emitted")

				// Source should complete
				assert.Eventually(t, func() bool {
					return isSourceDone.Load()
				}, 100*time.Millisecond, 10*time.Millisecond)
			},
		},
		{
			name: "cancel stream stops execution",
			setup: func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool) {
				isSourceDone := &atomic.Bool{}
				sourceChan := make(chan Item[int])

				sourceStarted := make(chan struct{})

				stream := newStream(
					func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(sourceChan)
							defer isSourceDone.Store(true)

							// Signal that the source has started
							close(sourceStarted)

							// Wait for cancellation
							<-ctx.Done()
						}()
						return sourceChan
					},
				)

				return stream, sourceChan, isSourceDone
			},
			action: func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int]) {
				s.Cancel()
			},
			expectation: func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool) {
				result := <-res
				assert.ErrorIs(t, result.Err, context.Canceled)

				// Channel should be closed after error
				_, ok := <-res
				assert.False(t, ok, "output channel should be closed after error")

				// Source should complete
				assert.Eventually(t, func() bool {
					return isSourceDone.Load()
				}, 100*time.Millisecond, 10*time.Millisecond)
			},
		},
		{
			name: "drain completes gracefully",
			setup: func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool) {
				isSourceDone := &atomic.Bool{}
				sourceChan := make(chan Item[int], 1)

				sourceStarted := make(chan struct{})

				stream := newStream(
					func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(sourceChan)
							defer isSourceDone.Store(true)

							// Signal that the source has started
							close(sourceStarted)

							// Wait for complete signal
							<-complete

							// Send a final value before closing
							sourceChan <- Item[int]{Value: 42}
						}()
						return sourceChan
					},
				)

				return stream, sourceChan, isSourceDone
			},
			action: func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int]) {
				s.Drain()
			},
			expectation: func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool) {
				result := <-res
				assert.Equal(t, 42, result.Value)
				assert.NoError(t, result.Err)

				// Channel should be closed after the value
				_, ok := <-res
				assert.False(t, ok, "output channel should be closed after result is emitted")

				// Source should complete
				assert.Eventually(t, func() bool {
					return isSourceDone.Load()
				}, 100*time.Millisecond, 10*time.Millisecond)
			},
		},
		{
			name: "source error is propagated",
			setup: func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool) {
				isSourceDone := &atomic.Bool{}
				sourceChan := make(chan Item[int], 1)

				stream := newStream(
					func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(sourceChan)
							defer isSourceDone.Store(true)

							sourceChan <- Item[int]{Err: errors.New("source error")}
						}()
						return sourceChan
					},
				)

				return stream, sourceChan, isSourceDone
			},
			action: func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int]) {
				// No additional action needed, the setup will send the error
			},
			expectation: func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool) {
				result := <-res
				assert.Error(t, result.Err)
				assert.Equal(t, "source error", result.Err.Error())

				// Channel should be closed after error
				_, ok := <-res
				assert.False(t, ok, "output channel should be closed after error")

				// Source should complete
				assert.Eventually(t, func() bool {
					return isSourceDone.Load()
				}, 100*time.Millisecond, 10*time.Millisecond)
			},
		},
		{
			name: "running already running stream returns same channel",
			setup: func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool) {
				isSourceDone := &atomic.Bool{}
				sourceChan := make(chan Item[int], 1)

				stream := newStream(
					func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(sourceChan)
							defer isSourceDone.Store(true)

							// Stay alive for a bit
							time.Sleep(50 * time.Millisecond)
							sourceChan <- Item[int]{Value: 42}
						}()
						return sourceChan
					},
				)

				return stream, sourceChan, isSourceDone
			},
			action: func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int]) {
				// Run the stream
				res1 := s.Run(context.Background())

				// Try to run again and verify we get the same channel
				res2 := s.Run(context.Background())

				assert.Equal(t, res1, res2)
			},
			expectation: func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool) {
				result := <-res
				assert.Equal(t, 42, result.Value)
				assert.NoError(t, result.Err)

				// Channel should be closed after the value
				_, ok := <-res
				assert.False(t, ok, "output channel should be closed after result is emitted")
			},
		},
		{
			name: "await done blocks until completion",
			setup: func(t *testing.T) (*Stream[int], <-chan Item[int], *atomic.Bool) {
				isSourceDone := &atomic.Bool{}
				sourceChan := make(chan Item[int], 1)

				stream := newStream(
					func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(sourceChan)
							defer isSourceDone.Store(true)

							time.Sleep(50 * time.Millisecond)
							sourceChan <- Item[int]{Value: 42}
						}()
						return sourceChan
					},
				)

				return stream, sourceChan, isSourceDone
			},
			action: func(t *testing.T, s *Stream[int], sourceChan <-chan Item[int]) {
				// Start a timer to measure time spent in AwaitDone
				start := time.Now()

				// Run the stream and consume the result
				res := s.Run(context.Background())
				<-res

				// AwaitDone should block until all goroutines finish
				s.AwaitDone()

				// Verify that we waited at least 40ms (the sleep duration)
				elapsed := time.Since(start)
				assert.GreaterOrEqual(t, elapsed, 40*time.Millisecond)
			},
			expectation: func(t *testing.T, res <-chan Item[int], isSourceDone *atomic.Bool) {
				assert.True(t, isSourceDone.Load(), "source should be done after AwaitDone returns")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			stream, sourceChan, isSourceDone := tt.setup(t)

			// Start the stream
			res := stream.Run(context.Background())

			// Perform the test action
			tt.action(t, stream, sourceChan)

			// Verify expectations
			tt.expectation(t, res, isSourceDone)
		})
	}
}

func TestStreamIsNotRunning(t *testing.T) {
	// Table-driven tests for methods that should do nothing when stream is not running
	tests := []struct {
		name   string
		method func(*Stream[int])
	}{
		{
			name:   "cancel on non-running stream",
			method: func(s *Stream[int]) { s.Cancel() },
		},
		{
			name:   "drain on non-running stream",
			method: func(s *Stream[int]) { s.Drain() },
		},
		{
			name:   "await done on non-running stream",
			method: func(s *Stream[int]) { s.AwaitDone() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a stream but don't run it
			stream := newStream(
				func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
					return make(chan Item[int])
				},
			)

			// Each method should return immediately without error when stream is not running
			tt.method(stream)
			assert.False(t, stream.isRunning.Load())
		})
	}
}
