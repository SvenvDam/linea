package sources

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/test"
)

func TestPoll(t *testing.T) {
	tests := []struct {
		name     string
		poll     func() func(context.Context) int
		interval time.Duration
		duration time.Duration
		check    func(t *testing.T, res []int)
	}{
		{
			name: "polls at regular intervals",
			poll: func() func(context.Context) int {
				counter := atomic.Int32{}
				return func(context.Context) int {
					return int(counter.Add(1))
				}
			},
			interval: 50 * time.Millisecond,
			duration: 200 * time.Millisecond,
			check: func(t *testing.T, res []int) {
				assert.Greater(t, len(res), 2, "should have polled multiple times")
				for i := 0; i < len(res)-1; i++ {
					assert.Equal(t, 1, res[i+1]-res[i], "values should increment by 1")
				}
			},
		},
		{
			name: "respects polling interval",
			poll: func() func(context.Context) int {
				var lastPoll time.Time
				counter := atomic.Int32{}
				return func(context.Context) int {
					now := time.Now()
					if !lastPoll.IsZero() {
						diff := now.Sub(lastPoll)
						assert.InDelta(t, 100, diff.Milliseconds(), 20, "polling interval should be respected")
					}
					lastPoll = now
					return int(counter.Add(1))
				}
			},
			interval: 100 * time.Millisecond,
			duration: 250 * time.Millisecond,
			check: func(t *testing.T, res []int) {
				assert.Greater(t, len(res), 1)
			},
		},
		{
			name: "handles slow polling function",
			poll: func() func(context.Context) int {
				counter := atomic.Int32{}
				return func(ctx context.Context) int {
					time.Sleep(150 * time.Millisecond) // Simulate slow operation
					return int(counter.Add(1))
				}
			},
			interval: 50 * time.Millisecond, // Shorter than poll duration
			duration: 400 * time.Millisecond,
			check: func(t *testing.T, res []int) {
				// Should have fewer items than theoretical maximum due to skipped polls
				theoreticalMax := int(400 / 50) // duration / interval
				assert.Less(t, len(res), theoreticalMax,
					"should have fewer items due to slow polling")

				// Values should still increment by 1
				for i := 0; i < len(res)-1; i++ {
					assert.Equal(t, 1, res[i+1]-res[i],
						"values should increment by 1")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			seen := make([]int, 0)

			stream := compose.SourceThroughFlowToSink2(
				Poll(tt.poll(), tt.interval),
				test.AssertEachItem(t, func(t *testing.T, in int) {
					assert.Positive(t, in)
				}),
				test.CaptureItems(&seen),
				sinks.Noop[int](),
			)

			resChan := stream.Run(ctx)
			time.Sleep(tt.duration)
			stream.Drain()
			res := <-resChan

			assert.True(t, res.Ok)
			tt.check(t, seen)
		})
	}
}
