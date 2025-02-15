package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Slice creates a Sink that collects all items into a slice in the order they
// are received. The resulting slice contains all successfully processed items.
//
// Type Parameters:
//   - I: The type of items to collect
//
// Returns a Sink that accumulates items into a slice
func Slice[I any]() *core.Sink[I, []I] {
	return core.NewSink(func(ctx context.Context, in <-chan I, cancel context.CancelFunc) []I {
		slice := make([]I, 0)
		return util.SinkLoop(ctx, in, slice, func(item I, acc []I) []I {
			return append(acc, item)
		})
	})
}
