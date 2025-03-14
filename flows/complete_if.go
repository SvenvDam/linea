package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// CompleteIf creates a Flow that gracefully completes stream processing when the predicate returns
// true for any item. Items are passed through unchanged until completion is triggered.
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
// Returns a Flow that may gracefully complete processing based on item values
func CompleteIf[I any](
	pred func(I) bool,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[I]) core.StreamAction {
			util.Send(ctx, core.Item[I]{Value: elem}, out)
			if pred(elem) {
				return core.ActionComplete
			} else {
				return core.ActionProceed
			}
		},
		nil,
		nil,
		nil,
		opts...)
}
