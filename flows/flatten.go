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
	return core.NewFlow(func(ctx context.Context, elem []I, out chan<- I, cancel context.CancelFunc, complete core.CompleteFunc) bool {
		util.SendMany(ctx, elem, out)
		return true
	}, func(ctx context.Context, out chan<- I) {}, opts...)
}
