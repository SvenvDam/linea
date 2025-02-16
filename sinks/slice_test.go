package sinks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sources"
)

func TestSlice(t *testing.T) {
	tests := []struct {
		name  string
		setup func(ctx context.Context) context.Context
		input []int
		want  []int
	}{
		{
			name:  "empty slice",
			setup: func(ctx context.Context) context.Context { return ctx },
			input: []int{},
			want:  []int{},
		},
		{
			name:  "non-empty slice",
			setup: func(ctx context.Context) context.Context { return ctx },
			input: []int{1, 2, 3},
			want:  []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			stream := compose.SourceToSink(
				sources.Slice(tt.input),
				Slice[int](),
			)

			res := <-stream.Run(ctx)
			assert.Equal(t, tt.want, res.Value)
			assert.True(t, res.Ok)
		})
	}
}
