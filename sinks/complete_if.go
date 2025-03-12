package sinks

import (
	"context"

	"github.com/svenvdam/linea/core"
)

// CompleteIf creates a Sink that gracefully completes stream processing when the predicate returns
// true for any item. Processing continues until all upstream components finish their work.
//
// Unlike CancelIf which abruptly cancels the stream, CompleteIf signals a graceful shutdown
// that allows upstream components to finish processing their current work.
//
// Type Parameters:
//   - I: The type of items to check
//
// Parameters:
//   - pred: Function that returns true if processing should be completed
//
// Returns a Sink that may gracefully complete processing based on item values
func CompleteIf[I any](
	pred func(I) bool,
) *core.Sink[I, struct{}] {
	return core.NewSink(
		struct{}{},
		func(ctx context.Context, in I, acc core.Item[struct{}]) (core.Item[struct{}], core.StreamAction) {
			if pred(in) {
				return acc, core.ActionComplete
			}
			return acc, core.ActionProceed
		},
		nil,
		nil,
	)
}
