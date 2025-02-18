package sources

import (
	"context"

	"github.com/svenvdam/linea/core"
)

// Slice creates a Source that emits all items from a slice in order. The source
// will emit each item in the slice exactly once and then complete.
//
// Type Parameters:
//   - O: The type of items in the slice
//
// Parameters:
//   - slice: The slice containing items to emit
//   - opts: Optional configuration options for the source
//
// Returns a Source that produces items from the slice
func Slice[O any](
	slice []O,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context, drain <-chan struct{}) <-chan O {
		out := make(chan O)
		go func() {
			defer close(out)
			for _, elem := range slice {
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
