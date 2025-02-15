package flows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/core"
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
			before := make([]int, 0)
			after := make([]int, 0)

			stream := core.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CaptureItems(&before),
				TakeWhile(tt.pred),
				test.CaptureItems(&after),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)

			assert.Equal(t, tt.input, before)
			assert.Equal(t, tt.want, after)
			assert.True(t, res.Ok)
		})
	}
}
