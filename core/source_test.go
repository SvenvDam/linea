package core

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSource(t *testing.T) {
	tests := []struct {
		name    string
		bufSize int
		test    func(in chan<- Item[int], out <-chan Item[int], drain chan struct{}, cancel context.CancelFunc)
	}{
		{
			name:    "happy path - emits all generated values",
			bufSize: 0,
			test: func(in chan<- Item[int], out <-chan Item[int], drain chan struct{}, cancel context.CancelFunc) {
				in <- Item[int]{Value: 1}
				res, ok := <-out
				assert.Equal(t, 1, res.Value)
				assert.True(t, ok)

				close(in)
				_, ok = <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "respects drain signal",
			bufSize: 0,
			test: func(in chan<- Item[int], out <-chan Item[int], drain chan struct{}, cancel context.CancelFunc) {
				close(drain)
				_, ok := <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "respects context cancellation",
			bufSize: 0,
			test: func(in chan<- Item[int], out <-chan Item[int], drain chan struct{}, cancel context.CancelFunc) {
				cancel()
				_, ok := <-out
				assert.False(t, ok)
			},
		},
		{
			name:    "handles buffered channel",
			bufSize: 2,
			test: func(in chan<- Item[int], out <-chan Item[int], drain chan struct{}, cancel context.CancelFunc) {
				in <- Item[int]{Value: 1}
				in <- Item[int]{Value: 2}
				in <- Item[int]{Value: 3}
				close(in)
				res := make([]int, 0)
				for v := range out {
					res = append(res, v.Value)
				}
				assert.Equal(t, []int{1, 2, 3}, res)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			in := make(chan Item[int])

			wg := &sync.WaitGroup{}
			drain := make(chan struct{})

			// Create source with test generator
			source := NewSource(
				func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[int] {
					return in
				},
				WithSourceBufSize(tt.bufSize),
			)

			// Start source
			out := source.setup(ctx, cancel, wg, drain)

			// Give source time to start
			time.Sleep(20 * time.Millisecond)

			tt.test(in, out, drain, cancel)

			// Wait for all goroutines to complete
			wg.Wait()
		})
	}
}
