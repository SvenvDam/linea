package core

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/util"
)

func TestFlow(t *testing.T) {
	tests := []struct {
		name    string
		bufSize int
		setup   func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool)
		action  func(completeSignal chan<- struct{}, cancel context.CancelFunc)
		check   func(t *testing.T, out <-chan Item[string], upstreamCompleteCalled *atomic.Bool)
		test    func(t *testing.T, in chan<- Item[int], out <-chan Item[string], cancel context.CancelFunc)
	}{
		{
			name:    "happy path - transforms all input values",
			bufSize: 0,
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int], 1)
				in <- Item[int]{Value: 42}
				close(in)

				return in, nil
			},
			check: func(t *testing.T, out <-chan Item[string], upstreamCompleteCalled *atomic.Bool) {
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:42", res.Value)

				_, ok = <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "respects context cancellation",
			bufSize: 0,
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int])

				return in, nil

			},
			action: func(_ chan<- struct{}, cancel context.CancelFunc) {
				cancel()
			},
			check: func(t *testing.T, out <-chan Item[string], upstreamCompleteCalled *atomic.Bool) {
				_, ok := <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "onElem can stop processing",
			bufSize: 0,
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int], 3)
				in <- Item[int]{Value: 42}
				in <- Item[int]{Value: -1}
				in <- Item[int]{Value: 43}
				close(in)

				return in, nil
			},
			check: func(t *testing.T, out <-chan Item[string], upstreamCompleteCalled *atomic.Bool) {
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:42", res.Value)

				_, ok = <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "signals upstream to complete when requested",
			bufSize: 0,
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int])
				upstreamCompleteCalled := &atomic.Bool{}

				go func() {
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
			check: func(t *testing.T, out <-chan Item[string], upstreamCompleteCalled *atomic.Bool) {
				<-out
				assert.True(t, upstreamCompleteCalled.Load(), "upstream complete function should have been called")
			},
		},
		{
			name:    "handles errors with default error handler",
			bufSize: 0,
			setup: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) (chan Item[int], *atomic.Bool) {
				in := make(chan Item[int], 3)
				in <- Item[int]{Value: 1}
				in <- Item[int]{Err: assert.AnError}
				in <- Item[int]{Value: 2}
				close(in)

				return in, nil
			},
			check: func(t *testing.T, out <-chan Item[string], upstreamCompleteCalled *atomic.Bool) {
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:1", res.Value)

				res, ok = <-out
				assert.True(t, ok)
				assert.Equal(t, assert.AnError, res.Err)

				_, ok = <-out
				assert.False(t, ok) // closed after error
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			complete := make(chan struct{})
			in, upstreamCompleteCalled := tt.setup(ctx, cancel, wg, complete)
			setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
				return in
			}

			// Create flow that converts ints to strings
			flow := NewFlow[int, string](
				// onElem function
				func(ctx context.Context, elem int, out chan<- Item[string], cancel context.CancelFunc, complete CompleteFunc) bool {
					if elem == -1 {
						return false
					}
					out <- Item[string]{Value: "value:" + strconv.Itoa(elem)}
					return true
				},
				nil,
				nil,
				WithFlowBufSize(tt.bufSize),
			)

			// Start flow
			out := flow.setup(ctx, cancel, wg, complete, setup)

			if tt.action != nil {
				tt.action(complete, cancel)
			}

			// Give flow time to start
			time.Sleep(10 * time.Millisecond)

			tt.check(t, out, upstreamCompleteCalled)
		})
	}
}

// TestFlowOnDone verifies that the onDone function is called in all termination scenarios
func TestFlowOnDone(t *testing.T) {
	tests := []struct {
		name          string
		triggerMethod func(in chan<- Item[int], cancel context.CancelFunc)
	}{
		{
			name: "called on input channel close",
			triggerMethod: func(in chan<- Item[int], cancel context.CancelFunc) {
				close(in)
			},
		},
		{
			name: "called on context cancellation",
			triggerMethod: func(in chan<- Item[int], cancel context.CancelFunc) {
				cancel()
			},
		},
		{
			name: "called when onElem returns false",
			triggerMethod: func(in chan<- Item[int], cancel context.CancelFunc) {
				in <- Item[int]{Value: -1} // Trigger value that causes onElem to return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			in := make(chan Item[int])
			completeChan, _ := util.NewCompleteChannel()

			setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
				return in
			}
			onDoneCalled := false

			flow := NewFlow[int, string](
				func(ctx context.Context, elem int, out chan<- Item[string], cancel context.CancelFunc, complete CompleteFunc) bool {
					return elem != -1
				},
				nil,
				func(ctx context.Context, out chan<- Item[string]) {
					onDoneCalled = true
				},
			)

			out := flow.setup(ctx, cancel, wg, completeChan, setup)

			time.Sleep(20 * time.Millisecond)
			tt.triggerMethod(in, cancel)

			// Wait for goroutine to complete
			wg.Wait()

			// Drain output channel
			for range out {
			}

			assert.True(t, onDoneCalled, "onDone should have been called")
		})
	}
}
