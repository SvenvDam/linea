package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
)

// CancelIf creates a Sink that cancels stream processing when the predicate returns
// true for any item. Processing continues until either cancellation occurs or all
// items are processed.
//
// Type Parameters:
//   - I: The type of items to check
//
// Parameters:
//   - pred: Function that returns true if processing should be cancelled
//
// Returns a Sink that may cancel processing based on item values
func CancelIf[I any](
	pred func(I) bool,
) *core.Sink[I, struct{}] {
	return core.NewSink(
		struct{}{},
		func(ctx context.Context, in I, acc struct{}, cancel context.CancelFunc) struct{} {
			if pred(in) {
				cancel()
			}
			return acc
		},
	)
}
