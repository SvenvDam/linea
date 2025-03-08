package flows

import (
	"context"
	"sync"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// FlatMapPar creates a Flow that transforms items into zero or more items in parallel using the provided mapping
// function. Up to 'parallelism' items will be processed concurrently. The order of
// output items is not guaranteed to match the input order.
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Parameters:
//   - fn: Function that transforms an input item into a slice of output items
//   - parallelism: Maximum number of items to process concurrently
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that transforms items in parallel
func FlatMapPar[I, O any](
	fn func(I) []O,
	parallelism int,
	opts ...core.FlowOption,
) *core.Flow[I, O] {
	sem := make(chan struct{}, parallelism)
	wg := sync.WaitGroup{}
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- core.Item[O], cancel context.CancelFunc, complete core.CompleteFunc) bool {
		sem <- struct{}{} // wait for a slot
		wg.Add(1)
		go func() {
			defer func() {
				wg.Done()
				<-sem // release the slot
			}()
			res := fn(elem)
			items := make([]core.Item[O], len(res))
			for i, item := range res {
				items[i] = core.Item[O]{Value: item}
			}
			util.SendMany(ctx, items, out)
		}()
		return true
	}, nil, func(ctx context.Context, out chan<- core.Item[O]) {
		wg.Wait() // wait for all goroutines to finish
	}, opts...)
}
