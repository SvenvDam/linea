package util

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSend(t *testing.T) {
	tests := []struct {
		name              string
		setupContext      func() context.Context
		outChannel        chan int
		expectElementSent bool
	}{
		{
			name: "successfully sends element",
			setupContext: func() context.Context {
				return context.Background()
			},
			outChannel:        make(chan int, 1),
			expectElementSent: true,
		},
		{
			name: "does not send when context is cancelled",
			setupContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx
			},
			outChannel:        make(chan int), // Unbuffered channel, so it blocks
			expectElementSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupContext()

			msg := 42

			Send(ctx, msg, tt.outChannel)

			if tt.expectElementSent {
				select {
				case res := <-tt.outChannel:
					assert.Equal(t, msg, res)
				case <-time.After(50 * time.Millisecond):
					assert.Fail(t, "expected element to be sent")
				}
			} else {
				select {
				case <-tt.outChannel:
					assert.Fail(t, "expected element to not be sent")
				case <-time.After(50 * time.Millisecond):
					// Function has exited, break the loop
				}
			}
		})
	}
}

func TestSendMany(t *testing.T) {
	tests := []struct {
		name         string
		setupContext func() (context.Context, context.CancelFunc)
		elements     []int
		outChannel   chan int
		action       func(cancel context.CancelFunc)
		expectedSent []int
	}{
		{
			name: "successfully sends all elements",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			elements:     []int{1, 2, 3},
			outChannel:   make(chan int, 3),
			expectedSent: []int{1, 2, 3},
		},
		{
			name: "sends no elements when context is cancelled before start",
			setupContext: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancel immediately
				return ctx, cancel
			},
			elements:     []int{1, 2, 3},
			outChannel:   make(chan int), // Unbuffered channel, so it blocks
			expectedSent: []int{},
		},
		{
			name: "sends partial elements when context is cancelled during execution",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			elements:   []int{1, 2, 3, 4, 5},
			outChannel: make(chan int, 3), // blocks before all elements are sent.
			action: func(cancel context.CancelFunc) {
				time.Sleep(20 * time.Millisecond)
				cancel()
			},
			expectedSent: []int{1, 2, 3},
		},
		{
			name: "handles empty slice",
			setupContext: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			elements:     []int{},
			outChannel:   make(chan int),
			expectedSent: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.setupContext()
			defer cancel()

			done := make(chan struct{})

			go func() {
				SendMany(ctx, tt.elements, tt.outChannel)
				close(done)
				close(tt.outChannel)
			}()

			if tt.action != nil {
				tt.action(cancel)
			}

			<-done

			received := []int{}
			for res := range tt.outChannel {
				received = append(received, res)
			}

			assert.Equal(t, tt.expectedSent, received)
		})
	}
}
