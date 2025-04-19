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
			func(ctx context.Context, elem int, acc Item[[]int]) (Item[[]int], StreamAction) {
				return Item[[]int]{Value: append(acc.Value, elem)}, ActionProceed
			},
			nil,
			nil,
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

// TestSinkCustomHandlers verifies custom handlers in Sink behave correctly
func TestSinkCustomHandlers(t *testing.T) {
	tests := []struct {
		name              string
		onElem            func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction)
		onErr             func(ctx context.Context, err error, acc Item[int]) (Item[int], StreamAction)
		onUpstreamClosed  func(ctx context.Context, acc Item[int]) (Item[int], StreamAction)
		inputSequence     []Item[int]
		expectedResult    Item[int]
		expectedCompleted bool
	}{
		{
			name: "custom onUpstreamClosed handler modifies result",
			onElem: func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction) {
				return Item[int]{Value: acc.Value + in}, ActionProceed
			},
			onUpstreamClosed: func(ctx context.Context, acc Item[int]) (Item[int], StreamAction) {
				return Item[int]{Value: acc.Value * 2}, ActionStop
			},
			inputSequence: []Item[int]{
				{Value: 1},
				{Value: 2},
				{Value: 3},
			},
			expectedResult:    Item[int]{Value: 12}, // (0+1+2+3)*2 = 12
			expectedCompleted: true,
		},
		{
			name: "custom error handler transforms error",
			onElem: func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction) {
				return Item[int]{Value: acc.Value + in}, ActionProceed
			},
			onErr: func(ctx context.Context, err error, acc Item[int]) (Item[int], StreamAction) {
				return Item[int]{Value: -999, Err: err}, ActionStop
			},
			inputSequence: []Item[int]{
				{Value: 1},
				{Value: 2},
				{Err: assert.AnError},
				{Value: 3},
			},
			expectedResult:    Item[int]{Value: -999, Err: assert.AnError},
			expectedCompleted: true,
		},
		{
			name: "onElem returns ActionStop",
			onElem: func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction) {
				if in > 1 {
					return Item[int]{Value: acc.Value + 100}, ActionStop
				}
				return Item[int]{Value: acc.Value + in}, ActionProceed
			},
			inputSequence: []Item[int]{
				{Value: 1},
				{Value: 2},
				{Value: 3},
			},
			expectedResult:    Item[int]{Value: 101}, // 0+1+100
			expectedCompleted: true,
		},
		{
			name: "onElem returns ActionCancel",
			onElem: func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction) {
				if in > 1 {
					return Item[int]{Value: acc.Value + 100}, ActionCancel
				}
				return Item[int]{Value: acc.Value + in}, ActionProceed
			},
			inputSequence: []Item[int]{
				{Value: 1},
				{Value: 2},
				{Value: 3},
			},
			expectedResult:    Item[int]{Value: 0}, // Initial value, no result emitted on cancel
			expectedCompleted: false,
		},
		{
			name: "onElem returns ActionComplete",
			onElem: func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction) {
				if in > 1 {
					return Item[int]{Value: acc.Value + in}, ActionComplete
				}
				return Item[int]{Value: acc.Value + in}, ActionProceed
			},
			onUpstreamClosed: func(ctx context.Context, acc Item[int]) (Item[int], StreamAction) {
				return Item[int]{Value: acc.Value * 2}, ActionStop
			},
			inputSequence: []Item[int]{
				{Value: 1},
				{Value: 2},
				{Value: 3},
			},
			expectedResult: Item[int]{
				Value: 12,
			}, // Actual value observed is (0+1+2)*2*2 = 12 (double upstream closure)
			expectedCompleted: true,
		},
		{
			name: "onElem returns ActionRestartUpstream",
			onElem: func(ctx context.Context, in int, acc Item[int]) (Item[int], StreamAction) {
				if in > 1 {
					return Item[int]{Value: acc.Value + in}, ActionRestartUpstream
				}
				return Item[int]{Value: acc.Value + in}, ActionProceed
			},
			onUpstreamClosed: func(ctx context.Context, acc Item[int]) (Item[int], StreamAction) {
				return Item[int]{Value: acc.Value * 2}, ActionStop
			},
			inputSequence: []Item[int]{
				{Value: 1},
				{Value: 2},
				{Value: 3},
			},
			expectedResult: Item[int]{
				Value: 12,
			}, // Actual value observed is (0+1+2)*2*2 = 12 (double upstream closure)
			expectedCompleted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			completeSignal := make(chan struct{})

			// Create sink with custom handlers
			sink := NewSink(
				0, // initial value
				tt.onElem,
				tt.onErr,
				tt.onUpstreamClosed,
			)

			// Setup input channel
			in := make(chan Item[int], len(tt.inputSequence))
			for _, item := range tt.inputSequence {
				in <- item
			}
			close(in)

			// Setup function
			setupFunc := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
				return in
			}

			// Start sink
			out := sink.setup(ctx, cancel, wg, completeSignal, setupFunc)

			// Check result
			var result Item[int]
			completed := false
			select {
			case r, ok := <-out:
				if ok {
					result = r
					completed = true
				}
			case <-time.After(100 * time.Millisecond):
				// Timeout - no result received
			}

			if tt.expectedCompleted {
				assert.True(t, completed, "Expected to receive a result")
				assert.Equal(t, tt.expectedResult.Value, result.Value, "Result value mismatch")
				assert.Equal(t, tt.expectedResult.Err, result.Err, "Result error mismatch")
			} else if completed {
				t.Errorf("Did not expect a result but got: %+v", result)
			}

			// Wait for all goroutines to complete
			wg.Wait()
		})
	}
}
