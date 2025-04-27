package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// FlatMap creates a Flow that transforms each input item into zero or more output items.
// The mapping function returns a slice of items, and each item in that slice is emitted
// individually downstream.
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Parameters:
//   - fn: Function that maps an input item to a slice of output items
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that transforms items using the mapping function
func FlatMap[I, O any](
	fn func(context.Context, I) []O,
	opts ...core.FlowOption,
) *core.Flow[I, O] {
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[O]) core.StreamAction {
			res := fn(ctx, elem)
			items := make([]core.Item[O], len(res))
			for i, item := range res {
				items[i] = core.Item[O]{Value: item}
			}
			util.SendMany(ctx, items, out)
			return core.ActionProceed
		},
		nil,
		nil,
		nil,
		opts...)
}
