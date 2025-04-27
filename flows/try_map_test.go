package flows

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestTryMap(t *testing.T) {
	tests := []struct {
		name        string
		input       []int
		mapFn       func(context.Context, int) (string, error)
		expected    []string
		expectedErr error
	}{
		{
			name:  "transforms all items successfully",
			input: []int{1, 2, 3},
			mapFn: func(ctx context.Context, i int) (string, error) {
				return strconv.Itoa(i), nil
			},
			expected:    []string{"1", "2", "3"},
			expectedErr: nil,
		},
		{
			name:  "cancels on error",
			input: []int{1, 2, 3, 4, 5},
			mapFn: func(ctx context.Context, i int) (string, error) {
				if i == 3 {
					return "", errors.New("error on 3")
				}
				return strconv.Itoa(i), nil
			},
			expected:    []string{"1", "2"},
			expectedErr: errors.New("error on 3"),
		},
		{
			name:  "handles empty input",
			input: []int{},
			mapFn: func(ctx context.Context, i int) (string, error) {
				return strconv.Itoa(i), nil
			},
			expected:    []string{},
			expectedErr: nil,
		},
		{
			name:  "immediate error cancels",
			input: []int{1, 2, 3},
			mapFn: func(ctx context.Context, i int) (string, error) {
				if i == 1 {
					return "", errors.New("error on first item")
				}
				return strconv.Itoa(i), nil
			},
			expected:    []string{},
			expectedErr: errors.New("error on first item"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mapFunc := tt.mapFn // local copy to avoid data race

			mapErrFlow := TryMap(func(ctx context.Context, i int) (string, error) {
				return mapFunc(ctx, i)
			})

			stream := compose.SourceThroughFlowToSink(
				sources.Slice(tt.input),
				mapErrFlow,
				sinks.Slice[string](),
			)

			res := <-stream.Run(ctx)

			assert.Equal(t, tt.expectedErr, res.Err)
			assert.Equal(t, tt.expected, res.Value)
		})
	}
}
