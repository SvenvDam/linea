package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// ForEach creates a Sink that applies a side-effect function to each item without
// producing a result. This is useful for operations like logging or writing to
// external systems.
//
// Type Parameters:
//   - I: The type of items to process
//
// Parameters:
//   - fn: Function to execute for each item
//
// Returns a Sink that applies the side-effect to each item
func ForEach[I any](
	fn func(I),
) *core.Sink[I, struct{}] {
	return core.NewSink(func(ctx context.Context, in <-chan I, cancel context.CancelFunc) struct{} {
		return util.SinkLoop(ctx, in, struct{}{}, func(item I, acc struct{}) struct{} {
			fn(item)
			return acc
		})
	})
}
