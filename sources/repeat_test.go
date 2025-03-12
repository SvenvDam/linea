package sources

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sinks"
)

func TestRepeat(t *testing.T) {
	tests := []struct {
		name    string
		element int
		action  func(stream *core.Stream[[]int])
		check   func(t *testing.T, res core.Item[[]int])
	}{
		{
			name:    "cancels when context is cancelled",
			element: 100,
			action: func(stream *core.Stream[[]int]) {
				go func() {
					stream.Cancel()
					time.Sleep(10 * time.Millisecond)
				}()
			},
			check: func(t *testing.T, res core.Item[[]int]) {
				fmt.Println(res)
				assert.Nil(t, res.Value)
				assert.Equal(t, res.Err, context.Canceled)
			},
		},
		{
			name:    "emits and gracefully stops when stream is drained",
			element: 42,
			action: func(stream *core.Stream[[]int]) {
				go func() {
					time.Sleep(10 * time.Millisecond)
					stream.Drain()
				}()
			},
			check: func(t *testing.T, res core.Item[[]int]) {
				// We should get some elements before draining
				assert.NotEmpty(t, res.Value)
				assert.NoError(t, res.Err)
				for _, val := range res.Value {
					assert.Equal(t, 42, val)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceToSink(
				Repeat(tt.element),
				sinks.Slice[int](),
			)

			resChan := stream.Run(ctx)

			tt.action(stream)

			res := <-resChan

			tt.check(t, res)
		})
	}
}
