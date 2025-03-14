package sources

import (
	"context"
	"sync"

	"github.com/svenvdam/linea/core"
)

// Chan creates a Source that emits items from a channel. The source will continue
// emitting items until the input channel is closed or the context is cancelled.
//
// Type Parameters:
//   - O: The type of items produced by this source
//
// Parameters:
//   - ch: The input channel from which items will be read
//   - opts: Optional configuration options for the source
//
// Returns a Source that produces items from the input channel
func Chan[O any](
	ch <-chan O,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(
		func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[O] {
			out := make(chan core.Item[O])
			wg.Add(1)
			go func() {
				defer close(out)
				defer wg.Done()
				for elem := range ch {
					select {
					case <-ctx.Done():
						return
					case <-complete:
						return
					case out <- core.Item[O]{Value: elem}:
					}
				}
			}()
			return out
		},
		opts...)
}
