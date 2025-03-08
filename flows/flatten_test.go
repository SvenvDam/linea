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

func TestFlatten(t *testing.T) {
	tests := []struct {
		name  string
		input [][]int
		want  []int
		check func(t *testing.T, item int)
	}{
		{
			name:  "flattens nested slices",
			input: [][]int{{1, 2}, {3, 4}, {5}},
			want:  []int{1, 2, 3, 4, 5},
			check: func(t *testing.T, item int) {
				assert.Contains(t, []int{1, 2, 3, 4, 5}, item)
			},
		},
		{
			name:  "handles empty input",
			input: [][]int{},
			want:  []int{},
			check: func(t *testing.T, item int) {
				t.Error("should not receive any items")
			},
		},
		{
			name:  "handles empty nested slices",
			input: [][]int{{}, {1, 2}, {}, {3}},
			want:  []int{1, 2, 3},
			check: func(t *testing.T, item int) {
				assert.Contains(t, []int{1, 2, 3}, item)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CheckItems(t, func(t *testing.T, elems [][]int) {
					assert.Equal(t, tt.input, elems)
				}),
				Flatten[int](),
				test.CheckItems(t, func(t *testing.T, elems []int) {
					assert.Equal(t, tt.want, elems)
				}),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
		})
	}
}
