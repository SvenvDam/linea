package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Batch creates a Flow that groups incoming items into slices of the specified size.
// When n items have been received, they are emitted as a single slice. If the stream
// ends with fewer than n items remaining, those items are emitted as a final batch.
//
// Type Parameters:
//   - I: The type of items to batch
//
// Parameters:
//   - n: The size of each batch
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that transforms individual items into slices of items
func Batch[I any](
	n int,
	opts ...core.FlowOption,
) *core.Flow[I, []I] {
	return core.NewFlow(func(ctx context.Context, in <-chan I, out chan<- []I, cancel context.CancelFunc) {
		batch := make([]I, 0, n)

		util.ProcessLoop(ctx, in, out, func(item I) {
			batch = append(batch, item)
			if len(batch) == n {
				util.Send(ctx, batch, out)
				batch = make([]I, 0, n)
			}
		}, func() {
			if len(batch) > 0 {
				util.Send(ctx, batch, out)
				batch = make([]I, 0, n)
			}
		})
	}, opts...)
}
