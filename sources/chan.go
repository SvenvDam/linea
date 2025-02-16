package sources

import (
	"context"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Chan creates a Source that emits items from a channel. The source will continue
// emitting items until the input channel is closed or the context is cancelled.
//
// Type Parameters:
//   - O: The type of items produced by this source
//
// Parameters:
//   - ch: The input channel from which items will be read
//   - opts: Optional SourceOption functions to configure the source
//
// Returns a Source that produces items from the input channel
func Chan[O any](
	ch <-chan O,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context, out chan<- O, drain chan struct{}, cancel context.CancelFunc) {
		util.SourceLoop(ctx, out, drain, func(ctx context.Context) <-chan O {
			return ch
		})
	}, opts...)
}
