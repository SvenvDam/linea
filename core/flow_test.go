package core

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFlow(t *testing.T) {
	tests := []struct {
		name    string
		bufSize int
		test    func(t *testing.T, in chan<- int, out <-chan string, cancel context.CancelFunc)
	}{
		{
			name:    "happy path - transforms all input values",
			bufSize: 0,
			test: func(t *testing.T, in chan<- int, out <-chan string, cancel context.CancelFunc) {
				in <- 42
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:42", res)

				close(in)
				_, ok = <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "respects context cancellation",
			bufSize: 0,
			test: func(t *testing.T, in chan<- int, out <-chan string, cancel context.CancelFunc) {
				cancel()

				go func() {
					select {
					case in <- 42:
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
			test: func(t *testing.T, in chan<- int, out <-chan string, cancel context.CancelFunc) {
				in <- 1
				in <- 2
				in <- 3
				close(in)

				res := make([]string, 0, 3)
				for v := range out {
					res = append(res, v)
				}
				assert.Equal(t, []string{"value:1", "value:2", "value:3"}, res)
			},
		},
		{
			name:    "onElem can stop processing",
			bufSize: 0,
			test: func(t *testing.T, in chan<- int, out <-chan string, cancel context.CancelFunc) {
				in <- 42 // This will be processed
				res, ok := <-out
				assert.True(t, ok)
				assert.Equal(t, "value:42", res)

				in <- -1 // This will trigger early stop
				_, ok = <-out
				assert.False(t, ok)

				go func() {
					select {
					case in <- 43:
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
			in := make(chan int)

			// Create flow that converts ints to strings
			flow := NewFlow[int, string](
				// onElem function
				func(ctx context.Context, elem int, out chan<- string, cancel context.CancelFunc) bool {
					if elem == -1 {
						return false
					}
					out <- "value:" + strconv.Itoa(elem)
					return true
				},
				// onDone function
				func(ctx context.Context, out chan<- string) {
					// Nothing to clean up in this test
				},
				WithFlowBufSize(tt.bufSize),
			)

			// Start flow
			out := flow.setup(ctx, cancel, wg, in)

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
		triggerMethod func(in chan<- int, cancel context.CancelFunc)
	}{
		{
			name: "called on input channel close",
			triggerMethod: func(in chan<- int, cancel context.CancelFunc) {
				close(in)
			},
		},
		{
			name: "called on context cancellation",
			triggerMethod: func(in chan<- int, cancel context.CancelFunc) {
				cancel()
			},
		},
		{
			name: "called when onElem returns false",
			triggerMethod: func(in chan<- int, cancel context.CancelFunc) {
				in <- -1 // Trigger value that causes onElem to return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wg := &sync.WaitGroup{}
			in := make(chan int)
			onDoneCalled := false

			flow := NewFlow[int, string](
				func(ctx context.Context, elem int, out chan<- string, cancel context.CancelFunc) bool {
					return elem != -1
				},
				func(ctx context.Context, out chan<- string) {
					onDoneCalled = true
				},
			)

			out := flow.setup(ctx, cancel, wg, in)

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
