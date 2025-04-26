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

func TestFlatMap(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		mapper func(context.Context, int) []string
		want   []string
		check  func(t *testing.T, item string)
	}{
		{
			name:  "maps single item to multiple items",
			input: []int{1, 2},
			mapper: func(ctx context.Context, i int) []string {
				return []string{
					string(rune('a' + i - 1)),
					string(rune('A' + i - 1)),
				}
			},
			want: []string{"a", "A", "b", "B"},
			check: func(t *testing.T, item string) {
				assert.Contains(t, []string{"a", "A", "b", "B"}, item)
			},
		},
		{
			name:  "handles empty input",
			input: []int{},
			mapper: func(ctx context.Context, i int) []string {
				return []string{string(rune('a' + i - 1))}
			},
			want: []string{},
			check: func(t *testing.T, item string) {
				t.Error("should not receive any items")
			},
		},
		{
			name:  "handles mapper returning empty slice",
			input: []int{1, 2},
			mapper: func(ctx context.Context, i int) []string {
				return []string{}
			},
			want: []string{},
			check: func(t *testing.T, item string) {
				t.Error("should not receive any items")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CheckItems(t, func(t *testing.T, elems []int) {
					assert.Equal(t, tt.input, elems)
				}),
				FlatMap(tt.mapper),
				test.CheckItems(t, func(t *testing.T, elems []string) {
					assert.Equal(t, tt.want, elems)
				}),
				sinks.Noop[string](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
		})
	}
}
