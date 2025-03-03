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

func TestTakeWhile(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		pred  func(int) bool
		want  []int
	}{
		{
			name:  "takes items until predicate returns false",
			input: []int{1, 2, 3, 4, 5},
			pred:  func(i int) bool { return i < 3 },
			want:  []int{1, 2},
		},
		{
			name:  "takes all items when predicate always true",
			input: []int{1, 2, 3},
			pred:  func(i int) bool { return true },
			want:  []int{1, 2, 3},
		},
		{
			name:  "takes no items when predicate starts false",
			input: []int{1, 2, 3},
			pred:  func(i int) bool { return false },
			want:  []int{},
		},
		{
			name:  "handles empty input",
			input: []int{},
			pred:  func(i int) bool { return true },
			want:  []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink2(
				sources.Slice(tt.input),
				TakeWhile(tt.pred),
				test.CheckItems(t, func(t *testing.T, seen []int) {
					assert.Equal(t, tt.want, seen)
				}),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.True(t, res.Ok)
		})
	}
}
