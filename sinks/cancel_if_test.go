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
		name        string
		elements    []int
		pred        func(int) bool
		expectedErr error
	}{
		{
			name:        "cancels on first matching item",
			elements:    []int{1, 2, 3, 4, 5},
			pred:        func(i int) bool { return i == 3 },
			expectedErr: context.Canceled,
		},
		{
			name:        "passes all items when predicate never matches",
			elements:    []int{1, 2, 3},
			pred:        func(i int) bool { return false },
			expectedErr: nil,
		},
		{
			name:        "cancels on first item",
			elements:    []int{1, 2, 3},
			pred:        func(i int) bool { return true },
			expectedErr: context.Canceled,
		},
		{
			name:        "handles empty input",
			elements:    []int{},
			pred:        func(i int) bool { return true },
			expectedErr: nil,
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
			if tt.expectedErr != nil {
				assert.ErrorIs(t, res.Err, tt.expectedErr)
			} else {
				assert.NoError(t, res.Err)
			}
		})
	}
}
