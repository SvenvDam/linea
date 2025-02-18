package sinks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sources"
)

func TestReduce(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(context.Context) context.Context
		elements []int
		initial  int
		reducer  func(int, int) int
		want     int
	}{
		{
			name:     "sums all elements",
			setup:    func(ctx context.Context) context.Context { return ctx },
			elements: []int{1, 2, 3, 4, 5},
			initial:  0,
			reducer: func(acc, elem int) int {
				return acc + elem
			},
			want: 15,
		},
		{
			name:     "handles empty input",
			setup:    func(ctx context.Context) context.Context { return ctx },
			elements: []int{},
			initial:  42,
			reducer: func(acc, elem int) int {
				return acc + elem
			},
			want: 42,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			stream := compose.SourceToSink(
				sources.Slice(tt.elements),
				Reduce(tt.initial, tt.reducer),
			)

			res := <-stream.Run(ctx)
			assert.Equal(t, tt.want, res.Value)
			assert.True(t, res.Ok)
		})
	}
}
