package sources

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Repeat creates a Source that continuously emits the same item. The source will
// keep emitting the item until the context is cancelled or the stream is drained.
//
// Type Parameters:
//   - O: The type of item to repeat
//
// Parameters:
//   - elem: The item to emit repeatedly
//   - opts: Optional SourceOption functions to configure the source
//
// Returns a Source that produces the same item indefinitely
func Repeat[O any](
	elem O,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context, out chan<- O, drain chan struct{}, cancel context.CancelFunc) {
		util.SourceLoop(ctx, out, drain, func(ctx context.Context) <-chan O {
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
	}, opts...)
}
