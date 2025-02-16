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
			before := make([][]int, 0)
			after := make([]int, 0)

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CaptureItems(&before),
				Flatten[int](),
				test.CaptureItems(&after),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.Equal(t, tt.want, after)
			assert.Equal(t, tt.input, before)
			assert.True(t, res.Ok)
		})
	}
}
