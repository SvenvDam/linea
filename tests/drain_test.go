package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/flows"
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

			stream := compose.SourceThroughFlowToSink(
				sources.Repeat(1),
				flows.Map(func(i int) int { return i * 2 }),
				sinks.Slice[int](),
			)

			resChan := stream.Run(ctx)

			select {
			case <-resChan:
				assert.Fail(t, "stream should not have any results")
			case <-time.After(50 * time.Millisecond):
			}

			stream.Drain()

			res := <-resChan
			stream.AwaitDone()
			assert.NoError(t, res.Err)
			assert.Greater(t, len(res.Value), 5)

			_, ok := <-resChan
			assert.False(t, ok)

		})
	}
}
