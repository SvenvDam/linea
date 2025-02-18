package flows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestThrottle(t *testing.T) {
	const allowedDeltaNanos = 10_000_000 // 10ms tolerance
	var last *time.Time
	tests := []struct {
		name     string
		n        int
		interval time.Duration
		items    []int
		check    func(t *testing.T, elem int)
	}{
		{
			name:     "throttles single item per interval",
			n:        1,
			interval: 100 * time.Millisecond,
			items:    []int{1, 2, 3, 4, 5},
			check: func(t *testing.T, elem int) {
				now := time.Now()
				if last == nil {
					last = &now
					return
				}

				expectedTime := last.Add(100 * time.Millisecond)

				// Verify throttle interval is respected
				assert.InDelta(t,
					expectedTime.UnixNano(),
					now.UnixNano(),
					allowedDeltaNanos,
				)

				last = &now
			},
		},
		{
			name:     "allows multiple items per interval",
			n:        3,
			interval: 200 * time.Millisecond,
			items:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9},
			check: func(t *testing.T, elem int) {
				now := time.Now()

				if last != nil {
					if (elem-1)%3 == 0 { // first item of each batch
						expectedTime := last.Add(200 * time.Millisecond)

						assert.InDelta(t,
							expectedTime.UnixNano(),
							now.UnixNano(),
							allowedDeltaNanos,
						)
					} else {
						assert.InDelta(t,
							last.UnixNano(),
							now.UnixNano(),
							10_000_000,
						)
					}
				}

				last = &now
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			last = nil
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink2(
				sources.Slice(tt.items),
				Throttle[int](tt.n, tt.interval),
				test.AssertEachItem(t, tt.check),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.True(t, res.Ok)
		})
	}
}
