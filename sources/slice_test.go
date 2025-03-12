package sources

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
)

func TestSlice(t *testing.T) {
	tests := []struct {
		name     string
		elements []int
		interval time.Duration
		action   func(stream *core.Stream[[]int])
		check    func(t *testing.T, res core.Item[[]int])
	}{
		{
			name:     "emits all elements from slice",
			elements: []int{1, 2, 3},
			interval: time.Nanosecond,
			action:   func(stream *core.Stream[[]int]) {},
			check: func(t *testing.T, res core.Item[[]int]) {
				assert.Equal(t, core.Item[[]int]{Value: []int{1, 2, 3}}, res)
			},
		},
		{
			name:     "handles empty slice",
			elements: []int{},
			interval: time.Nanosecond,
			action:   func(stream *core.Stream[[]int]) {},
			check:    func(t *testing.T, res core.Item[[]int]) { assert.Equal(t, core.Item[[]int]{Value: []int{}}, res) },
		},
		{
			name:     "cancels when context is cancelled",
			elements: []int{1, 2, 3, 4, 5},
			interval: time.Second,
			action: func(stream *core.Stream[[]int]) {
				go func() {
					stream.Cancel()
					time.Sleep(10 * time.Millisecond)
				}()
			},
			check: func(t *testing.T, res core.Item[[]int]) {
				assert.Equal(t, core.Item[[]int]{Err: context.Canceled}, res)
			},
		},
		{
			name:     "gracefully stops when stream completes",
			elements: make([]int, 256),
			interval: 100 * time.Millisecond,
			action: func(stream *core.Stream[[]int]) {
				time.Sleep(10 * time.Millisecond)
				stream.Drain()
			},
			check: func(t *testing.T, res core.Item[[]int]) {
				assert.NoError(t, res.Err)
				assert.Less(t, len(res.Value), 256)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink(
				Slice(tt.elements),
				flows.Throttle[int](1, tt.interval),
				sinks.Slice[int](),
			)

			resChan := stream.Run(ctx)

			tt.action(stream)

			res := <-resChan

			tt.check(t, res)
		})
	}
}
