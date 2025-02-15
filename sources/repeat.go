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
//   - opts: Optional SourceOption functions to configure the source
//
// Returns a Source that produces the same item indefinitely
func Repeat[O any](
	elem O,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context) <-chan O {
		out := make(chan O)
		go func() {
			defer close(out)
			for {
				select {
				case out <- elem:
				case <-ctx.Done():
					return
				}
			}
		}()
		return out
	}, opts...)
}
