package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
)

// Reduce creates a Sink that combines all items into a single result using the given
// reduction function. The function is called for each item with the current
// accumulated result and the new item.
//
// Type Parameters:
//   - I: The type of input items
//   - R: The type of the reduced result
//
// Parameters:
//   - initial: The initial value for the reduction
//   - fn: Function that combines the current result with a new item
//
// Returns a Sink that reduces items to a single result
func Reduce[I, R any](
	initial R,
	fn func(R, I) R,
) *core.Sink[I, R] {
	return core.NewSink(
		initial,
		func(ctx context.Context, in I, acc R, cancel context.CancelFunc, complete core.CompleteFunc) (R, bool) {
			return fn(acc, in), true
		},
		nil,
	)
}
