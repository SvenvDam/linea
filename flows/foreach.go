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
	fn func(I),
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- I, cancel context.CancelFunc) bool {
		fn(elem)
		util.Send(ctx, elem, out)
		return true
	}, func(ctx context.Context, out chan<- I) {}, opts...)
}
