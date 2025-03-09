package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Flatten creates a Flow that takes a stream of slices and emits each item in those
// slices individually. This effectively converts a stream of slices into a stream
// of individual items.
//
// Type Parameters:
//   - I: The type of items contained in the input slices
//
// Parameters:
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that flattens slices into individual items
func Flatten[I any](
	opts ...core.FlowOption,
) *core.Flow[[]I, I] {
	return core.NewFlow(
		func(ctx context.Context, elem []I, out chan<- core.Item[I], cancel context.CancelFunc, complete core.CompleteFunc) bool {
			items := make([]core.Item[I], len(elem))
			for i, item := range elem {
				items[i] = core.Item[I]{Value: item}
			}
			util.SendMany(ctx, items, out)

			return true
		},
		nil,
		nil,
		opts...)
}
