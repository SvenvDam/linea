package sources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/test"
)

func TestSlice(t *testing.T) {
	tests := []struct {
		name     string
		elements []int
		want     []int
	}{
		{
			name:     "emits all elements from slice",
			elements: []int{1, 2, 3},
			want:     []int{1, 2, 3},
		},
		{
			name:     "handles empty slice",
			elements: []int{},
			want:     []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink(
				Slice(tt.elements),
				test.CheckItems(t, func(t *testing.T, seen []int) {
					assert.Equal(t, tt.want, seen)
				}),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
		})
	}
}
