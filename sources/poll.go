package sources

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Poll creates a Source that emits values received from a polling function at a specified interval.
// The source will continue polling and emitting values until the context is cancelled, the stream is drained,
// or the poll function returns an error.
//
// The poll function returns three values:
//   - val: Pointer to the value to emit (or nil if no value should be emitted)
//   - more: Whether there are more items available to poll immediately
//   - err: Error that occurred during polling (if non-nil, the stream will be cancelled)
//
// Type Parameters:
//   - O: The type of items produced by this source
//
// Parameters:
//   - poll: Function that takes a context and returns a pointer to a value (or nil), a flag indicating whether
//     there are more items to poll immediately, and an error
//   - interval: Duration between polling attempts
//   - opts: Optional configuration options for the source
//
// Returns a Source that produces items from the polling function
func Poll[O any](
	poll func(context.Context) (val *O, more bool, err error),
	interval time.Duration,
	opts ...core.SourceOption,
) *core.Source[O] {
	return core.NewSource(func(ctx context.Context, drain <-chan struct{}, cancel context.CancelFunc) <-chan O {
		out := make(chan O)
		go func() {
			defer close(out)

			// Create a ticker to control polling frequency
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			// Set to true to poll immediately on start, using atomic operations for thread safety
			shouldPoll := atomic.Bool{}

			for {
				if shouldPoll.Load() {
					// Get value from polling function
					val, more, err := poll(ctx)

					// If there's an error, cancel the stream and return
					if err != nil {
						cancel()
						return
					}

					// Send the value if it's not nil
					if val != nil {
						util.Send(ctx, *val, out)
					}

					// Reset the ticker based on whether there are more items to poll immediately
					if more {
						ticker.Reset(time.Nanosecond)
					} else {
						ticker.Reset(interval)
					}

					// Wait for next tick before polling again
					shouldPoll.Store(false)
				}

				// Wait for next tick or context cancellation
				select {
				case <-ctx.Done():
					return
				case <-drain:
					return
				case <-ticker.C:
					shouldPoll.Store(true)
				}
			}
		}()
		return out
	}, opts...)
}
