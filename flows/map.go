package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Map creates a Flow that transforms each input item into exactly one output item
// using the provided mapping function. Each item is processed independently.
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Parameters:
//   - fn: Function that transforms an input item into an output item
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that transforms items using the mapping function
func Map[I, O any](
	fn func(I) O,
	opts ...core.FlowOption,
) *core.Flow[I, O] {
	return core.NewFlow(func(ctx context.Context, in <-chan I, out chan<- O, cancel context.CancelFunc) {
		util.ProcessLoop(ctx, in, out, func(elem I) {
			util.Send(ctx, fn(elem), out)
		}, func() {})
	}, opts...)
}
