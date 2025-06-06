package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestCancel(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(context.Context) context.Context
		cancel      bool
		expectedErr error
	}{
		{
			name: "cancelled on start",
			setup: func(ctx context.Context) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				cancel()
				return ctx
			},
			cancel:      false,
			expectedErr: context.Canceled,
		},
		{
			name: "cancel after start",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			cancel:      true,
			expectedErr: context.Canceled,
		},
		{
			name: "not cancelled",
			setup: func(ctx context.Context) context.Context {
				return ctx
			},
			cancel:      false,
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			stream := compose.SourceToSink(
				sources.Repeat(1),
				sinks.Noop[int](),
			)

			resChan := stream.Run(ctx)

			time.Sleep(10 * time.Millisecond)

			if tt.cancel {
				stream.Cancel()
			} else {
				stream.Drain()
			}

			time.Sleep(10 * time.Millisecond)

			res := <-resChan
			assert.Equal(t, tt.expectedErr, res.Err)
		})
	}
}
