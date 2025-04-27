package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Filter creates a Flow that only allows items satisfying a predicate to pass through.
// Items that don't match the predicate are discarded.
//
// Type Parameters:
//   - I: The type of items to filter
//
// Parameters:
//   - pred: Function that returns true for items that should be emitted
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that selectively emits items based on the predicate
func Filter[I any](
	pred func(context.Context, I) bool,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[I]) core.StreamAction {
			if pred(ctx, elem) {
				util.Send(ctx, core.Item[I]{Value: elem}, out)
			}
			return core.ActionProceed
		},
		nil,
		nil,
		nil,
		opts...)
}
