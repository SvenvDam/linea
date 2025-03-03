package flows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestBatch(t *testing.T) {
	tests := []struct {
		name  string
		n     int
		items []int
		want  [][]int
	}{
		{
			name:  "batches items into groups of 2",
			n:     2,
			items: []int{1, 2, 3, 4, 5},
			want:  [][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			name:  "batches items evenly",
			n:     3,
			items: []int{1, 2, 3, 4, 5, 6},
			want:  [][]int{{1, 2, 3}, {4, 5, 6}},
		},
		{
			name:  "handles empty input",
			n:     2,
			items: []int{},
			want:  [][]int{},
		},
		{
			name:  "handles single item",
			n:     2,
			items: []int{1},
			want:  [][]int{{1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.items),
				test.CheckItems(t, func(t *testing.T, elems []int) {
					assert.Equal(t, tt.items, elems)
				}),
				Batch[int](tt.n),
				test.CheckItems(t, func(t *testing.T, elems [][]int) {
					assert.Equal(t, tt.want, elems)
				}),
				sinks.Noop[[]int](),
			)

			res := <-stream.Run(ctx)
			assert.True(t, res.Ok)
		})
	}
}
