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

func TestFlatMap(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		mapper func(int) []string
		want   []string
		check  func(t *testing.T, item string)
	}{
		{
			name:  "maps single item to multiple items",
			input: []int{1, 2},
			mapper: func(i int) []string {
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
			mapper: func(i int) []string {
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
			mapper: func(i int) []string {
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
			before := make([]int, 0)
			after := make([]string, 0)

			stream := core.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CaptureItems(&before),
				FlatMap(tt.mapper),
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
