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

func TestCancelIf(t *testing.T) {
	tests := []struct {
		name  string
		input []int
		pred  func(int) bool
		check func(t *testing.T, item int)
		ok    bool
	}{
		{
			name:  "cancels on first matching item",
			input: []int{1, 2, 3, 4, 5},
			pred:  func(i int) bool { return i == 3 },
			check: func(t *testing.T, item int) {
				assert.Less(t, item, 3, "should not receive items after cancellation")
			},
			ok: false,
		},
		{
			name:  "passes all items when predicate never matches",
			input: []int{1, 2, 3},
			pred:  func(i int) bool { return false },
			check: func(t *testing.T, item int) {
				assert.Contains(t, []int{1, 2, 3}, item)
			},
			ok: true,
		},
		{
			name:  "cancels on first item",
			input: []int{1, 2, 3},
			pred:  func(i int) bool { return true },
			check: func(t *testing.T, item int) {
				t.Error("should not receive any items")
			},
			ok: false,
		},
		{
			name:  "handles empty input",
			input: []int{},
			pred:  func(i int) bool { return true },
			check: func(t *testing.T, item int) {
				t.Error("should not receive any items")
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := core.SourceThroughFlowToSink2(
				sources.Slice(tt.input),
				CancelIf(tt.pred),
				test.AssertEachItem(t, tt.check),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.Equal(t, tt.ok, res.Ok)
		})
	}
}
