package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
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
	return core.NewSink(
		struct{}{},
		func(ctx context.Context, in I, acc struct{}, cancel context.CancelFunc, complete core.CompleteFunc) struct{} {
			return acc
		},
	)
}
