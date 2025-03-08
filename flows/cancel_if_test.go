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

func TestCancelIf(t *testing.T) {
	tests := []struct {
		name        string
		input       []int
		pred        func(int) bool
		check       func(t *testing.T, seen []int)
		expectedErr error
	}{
		{
			name:  "cancels on first matching item",
			input: []int{1, 2, 3, 4, 5},
			pred:  func(i int) bool { return i == 3 },
			check: func(t *testing.T, seen []int) {
				assert.Equal(t, []int{1, 2}, seen)
			},
			expectedErr: context.Canceled,
		},
		{
			name:  "passes all items when predicate never matches",
			input: []int{1, 2, 3},
			pred:  func(i int) bool { return false },
			check: func(t *testing.T, seen []int) {
				assert.Equal(t, []int{1, 2, 3}, seen)
			},
			expectedErr: nil,
		},
		{
			name:  "cancels on first item",
			input: []int{1, 2, 3},
			pred:  func(i int) bool { return true },
			check: func(t *testing.T, seen []int) {
				assert.Equal(t, []int{}, seen)
			},
			expectedErr: context.Canceled,
		},
		{
			name:  "handles empty input",
			input: []int{},
			pred:  func(i int) bool { return true },
			check: func(t *testing.T, seen []int) {
				assert.Equal(t, []int{}, seen)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink2(
				sources.Slice(tt.input),
				CancelIf(tt.pred),
				test.CheckItems(t, tt.check),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			if tt.expectedErr != nil {
				assert.ErrorIs(t, res.Err, tt.expectedErr)
			} else {
				assert.NoError(t, res.Err)
			}
		})
	}
}
