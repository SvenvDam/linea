package sources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/test"
)

func TestChan(t *testing.T) {
	tests := []struct {
		name     string
		elements []int
		want     []int
	}{
		{
			name:     "emits all elements from channel",
			elements: []int{1, 2, 3},
			want:     []int{1, 2, 3},
		},
		{
			name:     "handles empty channel",
			elements: []int{},
			want:     []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			ch := make(chan int)
			go func() {
				for _, elem := range tt.elements {
					ch <- elem
				}
				close(ch)
			}()

			seen := make([]int, 0)
			stream := core.SourceThroughFlowToSink(
				Chan(ch),
				test.CaptureItems(&seen),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.True(t, res.Ok)
			assert.Equal(t, tt.want, seen)
		})
	}
}
