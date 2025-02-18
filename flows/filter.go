package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Filter creates a Flow that selectively passes through items based on a predicate.
// Only items for which the predicate returns true are emitted downstream.
//
// Type Parameters:
//   - I: The type of items to filter
//
// Parameters:
//   - pred: Function that returns true for items that should be kept
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that filters items based on the predicate
func Filter[I any](
	pred func(I) bool,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- I, cancel context.CancelFunc) bool {
		if pred(elem) {
			util.Send(ctx, elem, out)
		}
		return true
	}, func(ctx context.Context, out chan<- I) {}, opts...)
}
