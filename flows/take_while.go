package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// TakeWhile creates a Flow that emits items as long as the predicate returns true.
// Once the predicate returns false for an item, the flow stops emitting items.
//
// Type Parameters:
//   - I: The type of items to check
//
// Parameters:
//   - pred: Function that returns true for items that should be emitted
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that selectively emits items based on the predicate
func TakeWhile[I any](
	pred func(I) bool,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- core.Item[I], cancel context.CancelFunc, complete core.CompleteFunc) bool {
		if !pred(elem) {
			return false
		}
		util.Send(ctx, core.Item[I]{Value: elem}, out)
		return true
	}, nil, nil, opts...)
}
