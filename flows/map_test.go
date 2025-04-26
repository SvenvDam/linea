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
		mapper func(context.Context, int) string
		want   []string
	}{
		{
			name:  "maps integers to strings",
			input: []int{1, 2, 3},
			mapper: func(ctx context.Context, i int) string {
				return strconv.Itoa(i)
			},
			want: []string{"1", "2", "3"},
		},
		{
			name:  "handles empty input",
			input: []int{},
			mapper: func(ctx context.Context, i int) string {
				return strconv.Itoa(i)
			},
			want: []string{},
		},
		{
			name:  "propagates errors",
			input: []int{1, 2, 3},
			mapper: func(ctx context.Context, i int) string {
				return strconv.Itoa(i)
			},
			want: []string{"1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CheckItems(t, func(t *testing.T, seen []int) {
					assert.Equal(t, tt.input, seen)
				}),
				Map(tt.mapper),
				test.CheckItems(t, func(t *testing.T, seen []string) {
					assert.Equal(t, tt.want, seen)
				}),
				sinks.Noop[string](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
		})
	}
}
