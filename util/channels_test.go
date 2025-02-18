package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSend verifies that Send correctly handles both successful sends and context cancellation
func TestSend(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(context.Context) context.Context
		element  int
		wantSent bool
	}{
		{
			name:     "successfully sends element to channel",
			setup:    func(ctx context.Context) context.Context { return ctx },
			element:  42,
			wantSent: true,
		},
		{
			name: "respects context cancellation",
			setup: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			element:  42,
			wantSent: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			ch := make(chan int)

			go func() {
				Send(ctx, tt.element, ch)
				close(ch)
			}()

			res, ok := <-ch
			assert.Equal(t, tt.wantSent, ok)
			if tt.wantSent {
				assert.Equal(t, tt.element, res)
			}
		})
	}
}

// TestSendMany verifies that SendMany correctly handles both successful sends and context cancellation
func TestSendMany(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(context.Context) context.Context
		elements  []int
		wantCount int
	}{
		{
			name:      "sends all elements",
			setup:     func(ctx context.Context) context.Context { return ctx },
			elements:  []int{1, 2, 3},
			wantCount: 3,
		},
		{
			name: "respects context cancellation",
			setup: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			elements:  []int{1, 2, 3},
			wantCount: 0,
		},
		{
			name:      "handles empty slice",
			setup:     func(ctx context.Context) context.Context { return ctx },
			elements:  []int{},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			ch := make(chan int)

			// Start sender
			go func() {
				SendMany(ctx, tt.elements, ch)
				close(ch)
			}()

			received := make([]int, 0)
			// Collect received elements
			for elem := range ch {
				received = append(received, elem)
			}

			assert.Len(t, received, tt.wantCount)
			if tt.wantCount > 0 {
				assert.Equal(t, tt.elements, received)
			}
		})
	}
}
