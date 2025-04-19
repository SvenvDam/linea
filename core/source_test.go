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

// TestSourceAdditionalScenarios tests edge cases for Source to improve coverage
func TestSourceAdditionalScenarios(t *testing.T) {
	tests := []struct {
		name        string
		bufSize     int
		setupSource func() (*Source[int], chan Item[int])
		action      func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, out <-chan Item[int], inChan chan Item[int])
		verify      func(t *testing.T, results []Item[int])
	}{
		{
			name:    "forwards errors from generator",
			bufSize: 0,
			setupSource: func() (*Source[int], chan Item[int]) {
				in := make(chan Item[int], 3)

				source := NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[int] {
						return in
					},
				)

				return source, in
			},
			action: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, out <-chan Item[int], inChan chan Item[int]) {
				inChan <- Item[int]{Value: 1}
				inChan <- Item[int]{Err: assert.AnError}
				inChan <- Item[int]{Value: 2}
				close(inChan)
			},
			verify: func(t *testing.T, results []Item[int]) {
				assert.Len(t, results, 3)
				assert.Equal(t, 1, results[0].Value)
				assert.Equal(t, assert.AnError, results[1].Err)
				assert.Equal(t, 2, results[2].Value)
			},
		},
		{
			name:    "large source buffer with slow consumer",
			bufSize: 10, // Large buffer
			setupSource: func() (*Source[int], chan Item[int]) {
				in := make(chan Item[int], 20)

				source := NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[int] {
						return in
					},
					WithSourceBufSize(10),
				)

				return source, in
			},
			action: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, out <-chan Item[int], inChan chan Item[int]) {
				// Fill input channel with sequential values
				for i := 1; i <= 15; i++ {
					inChan <- Item[int]{Value: i}
				}
				close(inChan)

				// Just read the first few items to ensure slow consumer behavior
				count := 0
				for range out {
					count++
					if count >= 5 {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
			},
			verify: func(t *testing.T, results []Item[int]) {
				// Just check we got the expected number of items
				assert.GreaterOrEqual(t, len(results), 5)

				// Check that all values are between 1 and 15
				for _, item := range results {
					assert.GreaterOrEqual(t, item.Value, 1)
					assert.LessOrEqual(t, item.Value, 15)
				}
			},
		},
		{
			name:    "context cancellation during processing",
			bufSize: 0,
			setupSource: func() (*Source[int], chan Item[int]) {
				in := make(chan Item[int])

				source := NewSource(
					func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[int] {
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(in)

							// Send several values with delay to ensure context cancellation happens mid-processing
							for i := 0; i < 5; i++ {
								select {
								case <-ctx.Done():
									return
								case <-complete:
									return
								case in <- Item[int]{Value: i}:
									time.Sleep(10 * time.Millisecond)
								}
							}
						}()

						return in
					},
				)

				return source, nil
			},
			action: func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, out <-chan Item[int], _ chan Item[int]) {
				// Read a few values
				count := 0
				for range out {
					count++
					if count == 2 {
						// Cancel after reading 2 items
						cancel()
						break
					}
				}
			},
			verify: func(t *testing.T, results []Item[int]) {
				// Should have no more than 3 items (2 + maybe 1 extra in flight)
				assert.LessOrEqual(t, len(results), 3)
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

			source, inChan := tt.setupSource()

			// Start source
			out := source.setup(ctx, cancel, wg, complete)

			// Collect results in a goroutine
			results := make([]Item[int], 0)
			resultsDone := make(chan struct{})

			wg.Add(1)
			go func() {
				defer wg.Done()
				defer close(resultsDone)

				for item := range out {
					results = append(results, item)
				}
			}()

			// Execute test actions
			tt.action(ctx, cancel, wg, out, inChan)

			// Wait for all results to be collected
			<-resultsDone

			// Verify results
			tt.verify(t, results)

			// Wait for all goroutines to complete
			wg.Wait()
		})
	}
}
