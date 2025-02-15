package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestDrain(t *testing.T) {
	tests := []struct {
		name  string
		setup func(context.Context) context.Context
	}{
		{
			name: "drain",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			stream := core.SourceToSink(
				sources.Repeat(1),
				sinks.Slice[int](),
			)

			resChan := stream.Run(ctx)

			select {
			case <-resChan:
				assert.Fail(t, "stream should not have any results")
			case <-time.After(20 * time.Millisecond):
			}

			stream.Drain()

			res := <-resChan
			assert.True(t, res.Ok)
			assert.Greater(t, len(res.Value), 0)

			_, ok := <-resChan
			assert.False(t, ok)
		})
	}
}
