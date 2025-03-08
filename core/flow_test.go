package core

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/util"
)

func TestFlow(t *testing.T) {
	tests := []struct {
		name    string
		bufSize int
		test    func(t *testing.T, in chan<- Item[int], out <-chan Item[string], cancel context.CancelFunc)
	}{
		{
			name:    "happy path - transforms all input values",
			bufSize: 0,
			test: func(t *testing.T, in chan<- Item[int], out <-chan Item[string], cancel context.CancelFunc) {
				in <- Item[int]{Value: 42}
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:42", res.Value)

				close(in)
				_, ok = <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "respects context cancellation",
			bufSize: 0,
			test: func(t *testing.T, in chan<- Item[int], out <-chan Item[string], cancel context.CancelFunc) {
				cancel()

				go func() {
					select {
					case in <- Item[int]{Value: 42}:
						assert.Fail(t, "items should not be accepted after stop!")
					case <-time.After(20 * time.Millisecond):
					}
				}()
				_, ok := <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "handles buffered channel",
			bufSize: 2,
			test: func(t *testing.T, in chan<- Item[int], out <-chan Item[string], cancel context.CancelFunc) {
				in <- Item[int]{Value: 1}
				in <- Item[int]{Value: 2}
				in <- Item[int]{Value: 3}
				close(in)

				res := make([]string, 0, 3)
				for v := range out {
					res = append(res, v.Value)
				}
				assert.Equal(t, []string{"value:1", "value:2", "value:3"}, res)
			},
		},
		{
			name:    "onElem can stop processing",
			bufSize: 0,
			test: func(t *testing.T, in chan<- Item[int], out <-chan Item[string], cancel context.CancelFunc) {
				in <- Item[int]{Value: 42} // This will be processed
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:42", res.Value)

				in <- Item[int]{Value: -1} // This will trigger early stop
				_, ok = <-out
				assert.False(t, ok)

				go func() {
					select {
					case in <- Item[int]{Value: 43}:
						assert.Fail(t, "items should not be accepted after stop!")
					case <-time.After(20 * time.Millisecond):
					}
				}()

				_, ok = <-out
				assert.False(t, ok)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			in := make(chan Item[int])
			complete := make(chan struct{})
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

			// Give flow time to start
			time.Sleep(20 * time.Millisecond)

			// Run test
			tt.test(t, in, out, cancel)

			// Wait for all goroutines to complete
			wg.Wait()
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
