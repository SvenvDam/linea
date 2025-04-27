package flows

import (
	"context"
	"time"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/retry"
	"github.com/svenvdam/linea/util"
)

// Retry creates a flow that retries operations when errors are received from upstream.
// It uses the provided retry.Config to determine backoff timing and max retries.
// If the maximum number of retries is reached, the last error is propagated downstream.
//
// Type Parameters:
//   - I: The type of items flowing through the stream
//
// Parameters:
//   - config: The retry configuration controlling backoff and max retries
//   - opts: Optional flow configuration options
//
// Returns a Flow that adds retry capability to the stream
func Retry[I any](
	config *retry.Config,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	// Track attempts across error handler invocations
	var attempts uint

	return core.NewFlow(
		// Process normal elements
		func(ctx context.Context, elem I, out chan<- core.Item[I]) core.StreamAction {
			// Reset attempts counter on non-error items
			attempts = 0
			util.Send(ctx, core.Item[I]{Value: elem}, out)
			return core.ActionProceed
		},
		// Handle errors with retry logic
		func(ctx context.Context, err error, out chan<- core.Item[I]) core.StreamAction {
			// Check if retry is allowed based on the current attempt count
			backoff, canRetry := config.NextBackoff(attempts)
			if !canRetry {
				// Max retries reached, propagate the last error
				util.Send(ctx, core.Item[I]{Err: err}, out)
				return core.ActionStop
			}

			// Increment attempts before retry
			attempts++

			// Wait for the backoff duration before retrying
			select {
			case <-ctx.Done():
				// Context cancelled during backoff
				return core.ActionStop
			case <-time.After(backoff):
				// Backoff completed, retry by restarting upstream
				return core.ActionRestartUpstream
			}
		},
		// Handle upstream closure
		nil,
		// Cleanup on done (no action needed)
		nil,
		opts...,
	)
}
