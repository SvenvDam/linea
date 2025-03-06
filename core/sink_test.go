package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSink(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T, in chan<- int, out <-chan []int, completeSignal chan<- struct{}, cancel context.CancelFunc)
	}{
		{
			name: "happy path - processes all input values",
			test: func(t *testing.T, in chan<- int, out <-chan []int, completeSignal chan<- struct{}, cancel context.CancelFunc) {
				in <- 1
				in <- 2
				in <- 3
				close(in)

				result, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, []int{1, 2, 3}, result)

				_, ok = <-out
				assert.False(t, ok, "output channel should be closed after result is emitted")
			},
		},
		{
			name: "respects context cancellation",
			test: func(t *testing.T, in chan<- int, out <-chan []int, completeSignal chan<- struct{}, cancel context.CancelFunc) {
				in <- 1
				in <- 2

				cancel()

				assert.Eventually(t, func() bool {
					result, ok := <-out
					return ok == false && result == nil
				}, time.Second, 10*time.Millisecond)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			in := make(chan int)
			completeSignal := make(chan struct{})

			setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan int {
				return in
			}

			// Create sink that accumulates ints into a slice
			sink := NewSink(
				[]int{}, // initial empty slice
				func(ctx context.Context, elem int, acc []int, cancel context.CancelFunc, complete CompleteFunc) []int {
					return append(acc, elem)
				},
			)

			// Start sink
			out := sink.setup(ctx, cancel, wg, completeSignal, setup)

			// Give sink time to start
			time.Sleep(20 * time.Millisecond)

			// Run test
			tt.test(t, in, out, completeSignal, cancel)

			// Wait for all goroutines to complete
			wg.Wait()
		})
	}
}

// TestSinkWithCompleteUpstream verifies that the sink signals the upstream to complete when requested
func TestSinkWithCompleteUpstream(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	completeSignal := make(chan struct{})

	upstreamCompleteCalled := false

	// Mock upstream setup function that captures when complete is called
	setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan int {
		in := make(chan int)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(in)

			// Send a test value
			select {
			case in <- 1:
			case <-ctx.Done():
				return
			}

			// Wait for complete signal
			select {
			case <-complete:
				upstreamCompleteCalled = true
				return
			case <-ctx.Done():
				return
			}
		}()

		return in
	}

	// Create sink with a function that always calls complete
	sink := NewSink(
		[]int{},
		func(ctx context.Context, elem int, acc []int, cancel context.CancelFunc, complete CompleteFunc) []int {
			complete() // Always signal upstream to complete
			return append(acc, elem)
		},
	)

	// Start sink
	out := sink.setup(ctx, cancel, wg, completeSignal, setup)

	// Drain the output
	for range out {
	}

	// Wait for goroutines to complete
	wg.Wait()

	// Verify the upstream was signaled to complete
	assert.True(t, upstreamCompleteCalled, "upstream complete function should have been called")
}
