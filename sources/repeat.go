package sources

import (
	"context"

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
	return core.NewSource(func(ctx context.Context, drain <-chan struct{}) <-chan O {
		out := make(chan O)
		go func() {
			defer close(out)
			for {
				select {
				case <-ctx.Done():
					return
				case <-drain:
					return
				case out <- elem:
				}
			}
		}()
		return out
	})
}
