package flows

import (
	"context"
	"time"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

// Throttle creates a Flow that limits the rate at which items pass through.
// It allows n items to pass through per interval, holding back additional items
// until the next interval begins.
//
// Type Parameters:
//   - I: The type of items to throttle
//
// Parameters:
//   - n: Maximum number of items allowed per interval
//   - interval: Duration of each interval
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that throttles the rate of items
func Throttle[I any](
	n int,
	interval time.Duration,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	remaining := n
	ticker := time.NewTicker(interval)
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[I]) core.StreamAction {
			for remaining <= 0 {
				select {
				case <-ctx.Done():
					return core.ActionStop
				case <-ticker.C:
					remaining = n
				}
			}
			util.Send(ctx, core.Item[I]{Value: elem}, out)
			remaining--
			return core.ActionProceed
		},
		nil,
		nil,
		func(ctx context.Context, out chan<- core.Item[I]) {
			ticker.Stop()
		},
		opts...)
}
