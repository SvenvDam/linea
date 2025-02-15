package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Noop creates a Sink that consumes all items without performing any operation
// or producing any meaningful result. This is useful for cases where you want
// to drain a stream without processing its items.
//
// Type Parameters:
//   - I: The type of items to consume
//
// Returns a Sink that discards all items
func Noop[I any]() *core.Sink[I, struct{}] {
	return core.NewSink(func(ctx context.Context, in <-chan I, cancel context.CancelFunc) struct{} {
		return util.SinkLoop(ctx, in, struct{}{}, func(item I, acc struct{}) struct{} {
			return acc
		})
	})
}
