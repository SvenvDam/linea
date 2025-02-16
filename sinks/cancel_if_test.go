package sinks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sources"
)

func TestCancelIf(t *testing.T) {
	tests := []struct {
		name     string
		elements []int
		pred     func(int) bool
		ok       bool
	}{
		{
			name:     "cancels on first matching item",
			elements: []int{1, 2, 3, 4, 5},
			pred:     func(i int) bool { return i == 3 },
			ok:       false,
		},
		{
			name:     "passes all items when predicate never matches",
			elements: []int{1, 2, 3},
			pred:     func(i int) bool { return false },
			ok:       true,
		},
		{
			name:     "cancels on first item",
			elements: []int{1, 2, 3},
			pred:     func(i int) bool { return true },
			ok:       false,
		},
		{
			name:     "handles empty input",
			elements: []int{},
			pred:     func(i int) bool { return true },
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceToSink(
				sources.Slice(tt.elements),
				CancelIf(tt.pred),
			)

			res := <-stream.Run(ctx)
			assert.Equal(t, tt.ok, res.Ok)
		})
	}
}
