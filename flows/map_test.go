package flows

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestMap(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		mapper func(int) string
		want   []string
	}{
		{
			name:   "maps integers to strings",
			input:  []int{1, 2, 3},
			mapper: strconv.Itoa,
			want:   []string{"1", "2", "3"},
		},
		{
			name:   "handles empty input",
			input:  []int{},
			mapper: strconv.Itoa,
			want:   []string{},
		},
		{
			name:   "propagates errors",
			input:  []int{1, 2, 3},
			mapper: strconv.Itoa,
			want:   []string{"1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			before := make([]int, 0)
			after := make([]string, 0)

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CaptureItems(&before),
				Map(tt.mapper),
				test.CaptureItems(&after),
				sinks.Noop[string](),
			)

			res := <-stream.Run(ctx)
			assert.Equal(t, tt.want, after)
			assert.Equal(t, tt.input, before)
			assert.True(t, res.Ok)
		})
	}
}
