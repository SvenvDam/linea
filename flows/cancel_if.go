package flows

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// CancelIf creates a Flow that cancels stream processing when the predicate returns
// true for any item. Items are passed through unchanged until cancellation occurs.
//
// Type Parameters:
//   - I: The type of items to check
//
// Parameters:
//   - pred: Function that returns true if processing should be cancelled
//
// Returns a Flow that may cancel processing based on item values
func CancelIf[I any](
	pred func(I) bool,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, in <-chan I, out chan<- I, cancel context.CancelFunc) {
		util.ProcessLoop(ctx, in, out, func(item I) {
			if pred(item) {
				cancel()
				return
			}
			util.Send(ctx, item, out)
		}, func() {})
	})
}
