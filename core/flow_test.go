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
			flow := NewFlow(
				// onElem function
				func(ctx context.Context, elem int, out chan<- Item[string]) StreamAction {
					if elem == -1 {
						return ActionStop
					}
					out <- Item[string]{Value: "value:" + strconv.Itoa(elem)}
					return ActionProceed
				},
				nil,
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

			flow := NewFlow(
				func(ctx context.Context, elem int, out chan<- Item[string]) StreamAction {
					if elem == -1 {
						return ActionStop
					}
					return ActionProceed
				},
				nil,
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

// TestDefaultFlowDoneHandler verifies that the default flow done handler does nothing and returns
func TestDefaultFlowDoneHandler(t *testing.T) {
	tests := []struct {
		name     string
		ctx      context.Context
		bufSize  int
		sendData bool
	}{
		{
			name:     "empty channel",
			ctx:      context.Background(),
			bufSize:  1,
			sendData: false,
		},
		{
			name:     "channel with data",
			ctx:      context.Background(),
			bufSize:  1,
			sendData: true, // Should not affect the outcome
		},
		{
			name:     "canceled context",
			ctx:      func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			bufSize:  1,
			sendData: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a channel with the specified buffer size
			out := make(chan Item[int], tt.bufSize)

			// Optionally send data to the channel
			if tt.sendData {
				out <- Item[int]{Value: 42}
			}

			// Call DefaultFlowDoneHandler
			DefaultFlowDoneHandler(tt.ctx, out)

			// Verify that the channel is still open and data was not modified
			select {
			case item, ok := <-out:
				if tt.sendData {
					assert.True(t, ok, "Channel should still be open")
					assert.Equal(t, 42, item.Value, "Value should be unchanged")
				} else if ok {
					t.Errorf("DefaultFlowDoneHandler wrote to the output channel: %+v", item)
				} else {
					t.Error("DefaultFlowDoneHandler closed the output channel")
				}
			default:
				if !tt.sendData {
					// This is expected - no items should be in the channel
				} else {
					t.Error("Expected item disappeared from channel")
				}
			}

			// Clean up - close the channel
			close(out)
		})
	}
}

// TestFlowStreamActions verifies that all stream actions in the Flow component work correctly
func TestFlowStreamActions(t *testing.T) {
	tests := []struct {
		name         string
		setupInput   func() chan Item[int]
		actionToTest StreamAction
		expectOutput func(t *testing.T, out <-chan Item[string])
	}{
		{
			name: "ActionCancel cancels context",
			setupInput: func() chan Item[int] {
				in := make(chan Item[int], 2)
				in <- Item[int]{Value: 100}
				in <- Item[int]{Value: 200}
				return in
			},
			actionToTest: ActionCancel,
			expectOutput: func(t *testing.T, out <-chan Item[string]) {
				// We should get one item and then the channel should close
				item, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:100", item.Value)

				// Channel should be closed after context cancellation
				_, ok = <-out
				assert.False(t, ok, "expected channel to be closed after ActionCancel")
			},
		},
		{
			name: "ActionComplete completes upstream",
			setupInput: func() chan Item[int] {
				in := make(chan Item[int], 2)
				in <- Item[int]{Value: 100}
				// No need to send second value as it won't be processed
				close(in)
				return in
			},
			actionToTest: ActionComplete,
			expectOutput: func(t *testing.T, out <-chan Item[string]) {
				// Should receive the first item
				item, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:100", item.Value)

				// The channel will close after completing upstream and we won't see more items
				_, ok = <-out
				assert.False(t, ok, "expected channel to be closed after ActionComplete")
			},
		},
		{
			name: "ActionRestartUpstream restarts upstream source",
			setupInput: func() chan Item[int] {
				in := make(chan Item[int], 1)
				in <- Item[int]{Value: 100}
				// No need to add second item as it won't be processed
				close(in)
				return in
			},
			actionToTest: ActionRestartUpstream,
			expectOutput: func(t *testing.T, out <-chan Item[string]) {
				// Should receive first item
				item, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:100", item.Value)

				// The channel will close after restarting upstream
				_, ok = <-out
				assert.False(t, ok, "expected channel to be closed after ActionRestartUpstream")
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
			in := tt.setupInput()

			callCount := 0

			// Create flow that converts ints to strings but returns the specific action after first item
			flow := NewFlow(
				// onElem function
				func(ctx context.Context, elem int, out chan<- Item[string]) StreamAction {
					out <- Item[string]{Value: "value:" + strconv.Itoa(elem)}
					callCount++
					if callCount == 1 {
						return tt.actionToTest
					}
					return ActionProceed
				},
				nil,
				nil,
				nil,
			)

			// Setup input source
			setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
				return in
			}

			// Start flow
			out := flow.setup(ctx, cancel, wg, complete, setup)

			// Check the output
			tt.expectOutput(t, out)

			// Cleanup
			wg.Wait()
		})
	}
}
