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

func TestCancel(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(context.Context) context.Context
		cancel bool
		ok     bool
	}{
		{
			name: "cancelled on start",
			setup: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			cancel: false,
			ok:     false,
		},
		{
			name: "cancel after start",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			cancel: true,
			ok:     false,
		},
		{
			name: "not cancelled",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			cancel: false,
			ok:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			stream := core.SourceToSink(
				sources.Repeat(1),
				sinks.Noop[int](),
			)

			resChan := stream.Run(ctx)
			if tt.cancel {
				stream.Cancel()
			}
			time.Sleep(20 * time.Millisecond)
			stream.Drain()
			res := <-resChan
			assert.Equal(t, tt.ok, res.Ok)
		})
	}
}
