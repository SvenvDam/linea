package flows

import (
	"context"
	"sync"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// MapPar creates a Flow that transforms items in parallel using the provided mapping
// function. Up to 'parallelism' items will be processed concurrently. The order of
// output items is not guaranteed to match the input order.
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Parameters:
//   - fn: Function that transforms an input item into an output item
//   - parallelism: Maximum number of items to process concurrently
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that transforms items in parallel
func MapPar[I, O any](
	fn func(I) O,
	parallelism int,
	opts ...core.FlowOption,
) *core.Flow[I, O] {
	return core.NewFlow(func(ctx context.Context, in <-chan I, out chan<- O, cancel context.CancelFunc) {
		sem := make(chan struct{}, parallelism)
		wg := sync.WaitGroup{}

		util.ProcessLoop(ctx, in, out,
			func(elem I) {
				sem <- struct{}{} // wait for a slot
				wg.Add(1)
				go func() {
					defer func() {
						wg.Done()
						<-sem // release the slot
					}()
					util.Send(ctx, fn(elem), out)
				}()
			},
			func() {
				wg.Wait() // wait for all goroutines to finish
			},
		)
	}, opts...)
}
