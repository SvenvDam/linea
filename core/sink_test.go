package core

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSink(t *testing.T) {
	createSink := func() *Sink[int, []int] {
		return NewSink(
			[]int{}, // initial empty slice
			func(ctx context.Context, elem int, acc []int, cancel context.CancelFunc, complete CompleteFunc) ([]int, bool) {
				return append(acc, elem), true
			},
			nil, // Using default error handler
		)
	}

	tests := []struct {
		name   string
		setup  func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool)
		action func(completeSignal chan<- struct{}, cancel context.CancelFunc)
		check  func(t *testing.T, out <-chan Item[[]int], upstreamCompleteCalled *atomic.Bool)
	}{
		{
			name: "happy path - processes all input values",
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				upstreamCompleteCalled := &atomic.Bool{}
				in := make(chan Item[int], 3)
				in <- Item[int]{Value: 1}
				in <- Item[int]{Value: 2}
				in <- Item[int]{Value: 3}
				close(in)

				return in, upstreamCompleteCalled
			},
			check: func(t *testing.T, out <-chan Item[[]int], upstreamCompleteCalled *atomic.Bool) {
				result, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, []int{1, 2, 3}, result.Value)

				_, ok = <-out
				assert.False(t, ok, "output channel should be closed after result is emitted")
			},
		},
		{
			name: "respects context cancellation",
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int])

				return in, nil
			},
			action: func(_ chan<- struct{}, cancel context.CancelFunc) {
				cancel()
			},
			check: func(t *testing.T, out <-chan Item[[]int], upstreamCompleteCalled *atomic.Bool) {
				result, ok := <-out
				assert.False(t, ok)
				assert.Nil(t, result.Value)
			},
		},
		{
			name: "signals upstream to complete when requested",
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int])
				upstreamCompleteCalled := &atomic.Bool{}

				wg.Add(1)
				go func() {
					defer wg.Done()
					defer close(in)

					// Send a test value
					select {
					case in <- Item[int]{Value: 1}:
					case <-ctx.Done():
						return
					}

					// Wait for complete signal
					select {
					case <-complete:
						upstreamCompleteCalled.Store(true)
						return
					case <-ctx.Done():
						return
					}
				}()

				return in, upstreamCompleteCalled
			},
			action: func(completeSignal chan<- struct{}, _ context.CancelFunc) {
				close(completeSignal)
			},
			check: func(t *testing.T, out <-chan Item[[]int], upstreamCompleteCalled *atomic.Bool) {
				<-out
				assert.True(t, upstreamCompleteCalled.Load(), "upstream complete function should have been called")
			},
		},
		{
			name: "handles errors with default error handler",
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int], 3)
				in <- Item[int]{Value: 1}
				in <- Item[int]{Err: assert.AnError}
				in <- Item[int]{Value: 2}
				close(in)

				return in, nil
			},
			check: func(t *testing.T, out <-chan Item[[]int], upstreamCompleteCalled *atomic.Bool) {
				// Get the result
				result, ok := <-out
				assert.True(t, ok, "should receive a result")
				assert.Equal(t, []int{1}, result.Value, "should only contain the value processed before the error")
				assert.Equal(t, assert.AnError, result.Err, "should contain the original error")

				// Channel should be closed after error
				_, ok = <-out
				assert.False(t, ok, "output channel should be closed after result is emitted")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			completeSignal := make(chan struct{})

			sink := createSink()

			var upstreamCompleteCalled *atomic.Bool

			in, upstreamCompleteCalled := tt.setup(ctx, cancel, wg, completeSignal)
			setupFunc := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
				return in
			}

			// Start sink
			out := sink.setup(ctx, cancel, wg, completeSignal, setupFunc)

			// Run test
			if tt.action != nil {
				tt.action(completeSignal, cancel)
			}

			// Give sink time to start
			time.Sleep(10 * time.Millisecond)

			tt.check(t, out, upstreamCompleteCalled)

			// Wait for all goroutines to complete
			wg.Wait()
		})
	}
}
