package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// TryMap creates a Flow that transforms each input item into an output item
// using the provided mapping function that can return errors.
// If the mapping function returns an error for any item, the stream is cancelled.
// Each item is processed independently.
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Parameters:
//   - fn: Function that transforms an input item into an output item or returns an error
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that transforms items using the mapping function and handles errors
func TryMap[I, O any](
	fn func(I) (O, error),
	opts ...core.FlowOption,
) *core.Flow[I, O] {
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- O, cancel context.CancelFunc) bool {
		result, err := fn(elem)
		if err != nil {
			cancel()
			return false
		}
		util.Send(ctx, result, out)
		return true
	}, func(ctx context.Context, out chan<- O) {}, opts...)
}
