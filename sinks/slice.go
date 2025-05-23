package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
)

// Slice creates a Sink that collects all items into a slice in the order they
// are received. The resulting slice contains all successfully processed items.
//
// Type Parameters:
//   - I: The type of items to collect
//
// Returns a Sink that accumulates items into a slice
func Slice[I any]() *core.Sink[I, []I] {
	return core.NewSink(
		make([]I, 0),
		func(ctx context.Context, in I, acc core.Item[[]I]) (core.Item[[]I], core.StreamAction) {
			return core.Item[[]I]{Value: append(acc.Value, in)}, core.ActionProceed
		},
		nil,
		nil,
	)
}
