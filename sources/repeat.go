package sources

import (
	"context"
	"sync"

	"github.com/svenvdam/linea/core"
)

// Repeat creates a Source that continuously emits the same item. The source will
// keep emitting the item until the context is cancelled or the stream is drained.
//
// Type Parameters:
//   - O: The type of item to repeat
//
// Parameters:
//   - elem: The item to emit repeatedly
//   - opts: Optional configuration options for the source
//
// Returns a Source that produces the same item indefinitely
func Repeat[O any](
	elem O,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan core.Item[O] {
		out := make(chan core.Item[O])
		wg.Add(1)
		go func() {
			defer close(out)
			defer wg.Done()
			for {
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
	})
}
