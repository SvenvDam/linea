package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// ForEach creates a Flow that applies a side-effect function to each item and
// passes it through unchanged. This is useful for operations like logging or
// debugging that don't modify the stream contents.
//
// Type Parameters:
//   - I: The type of items in the stream
//
// Parameters:
//   - fn: Function to execute for each item
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that applies the side-effect to each item
func ForEach[I any](
	fn func(context.Context, I),
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[I]) core.StreamAction {
			fn(ctx, elem)
			util.Send(ctx, core.Item[I]{Value: elem}, out)
			return core.ActionProceed
		},
		nil,
		nil,
		nil,
		opts...)
}
