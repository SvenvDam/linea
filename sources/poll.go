package sources

import (
	"context"
	"time"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Poll creates a Source that emits values received from a polling function at a specified interval.
// The source will continue polling and emitting values until the context is cancelled or the stream is drained.
//
// Type Parameters:
//   - O: The type of items produced by this source
//
// Parameters:
//   - poll: Function that takes a context and returns the next value to emit
//   - interval: Duration between polling attempts
//   - opts: Optional SourceOption functions to configure the source
//
// Returns a Source that produces items from the polling function
func Poll[O any](
	poll func(context.Context) O,
	interval time.Duration,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context) <-chan O {
		out := make(chan O)
		go func() {
			defer close(out)
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					util.Send(ctx, poll(ctx), out)
				}
			}
		}()
		return out
	}, opts...)
}
