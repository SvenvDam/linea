package sources

import (
	"context"
	"errors"
	"sync"
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
		name      string
		poll      func() func(context.Context) (*int, bool, error)
		interval  time.Duration
		duration  time.Duration
		check     func(t *testing.T, res []int)
		expectErr bool
	}{
		{
			name: "polls at regular intervals",
			poll: func() func(context.Context) (*int, bool, error) {
				counter := atomic.Int32{}
				return func(context.Context) (*int, bool, error) {
					val := int(counter.Add(1))
					return &val, false, nil
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
			poll: func() func(context.Context) (*int, bool, error) {
				var lastPoll time.Time
				counter := atomic.Int32{}
				return func(context.Context) (*int, bool, error) {
					now := time.Now()
					if !lastPoll.IsZero() {
						diff := now.Sub(lastPoll)
						assert.InDelta(t, 100, diff.Milliseconds(), 20, "polling interval should be respected")
					}
					lastPoll = now
					val := int(counter.Add(1))
					return &val, false, nil
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
			poll: func() func(context.Context) (*int, bool, error) {
				counter := atomic.Int32{}
				return func(ctx context.Context) (*int, bool, error) {
					time.Sleep(150 * time.Millisecond) // Simulate slow operation
					val := int(counter.Add(1))
					return &val, false, nil
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
		{
			name: "skips nil values",
			poll: func() func(context.Context) (*int, bool, error) {
				counter := atomic.Int32{}
				return func(ctx context.Context) (*int, bool, error) {
					val := int(counter.Add(1))
					// Only return non-nil for even values
					if val%2 == 0 {
						return &val, false, nil
					}
					return nil, false, nil
				}
			},
			interval: 50 * time.Millisecond,
			duration: 250 * time.Millisecond,
			check: func(t *testing.T, res []int) {
				// All values should be even
				for _, val := range res {
					assert.Equal(t, 0, val%2, "all values should be even")
				}
			},
		},
		{
			name: "polls immediately when more flag is true",
			poll: func() func(context.Context) (*int, bool, error) {
				counter := atomic.Int32{}
				var timestamps []time.Time
				var mutex sync.Mutex

				return func(ctx context.Context) (*int, bool, error) {
					now := time.Now()

					mutex.Lock()
					timestamps = append(timestamps, now)

					// Check time intervals when we have enough data
					if len(timestamps) >= 7 {
						// First 5 polls should be very close together (immediate polling)
						for i := 1; i < 5; i++ {
							interval := timestamps[i].Sub(timestamps[i-1])
							// Should be much less than the configured polling interval
							assert.Less(t, interval, 10*time.Millisecond,
								"interval between polls with more=true should be very small")
						}

						// After value 5, intervals should be close to the configured interval
						for i := 6; i < len(timestamps); i++ {
							interval := timestamps[i].Sub(timestamps[i-1])
							// Should be close to the configured polling interval
							assert.InDelta(t, 100, interval.Milliseconds(), 10,
								"interval between polls with more=false should be close to configured interval")
						}
					}
					mutex.Unlock()

					val := int(counter.Add(1))
					// Signal more available for the first 5 values
					return &val, val < 5, nil
				}
			},
			interval: 100 * time.Millisecond,
			duration: 350 * time.Millisecond, // Longer duration to get enough regular polls too
			check: func(t *testing.T, res []int) {
				// Should have collected more than 5 values
				assert.Greater(t, len(res), 5,
					"should have collected more than 5 values")

				// Values should be sequential
				for i := 0; i < len(res)-1; i++ {
					assert.Equal(t, 1, res[i+1]-res[i], "values should increment by 1")
				}
			},
		},
		{
			name: "cancels on error",
			poll: func() func(context.Context) (*int, bool, error) {
				counter := atomic.Int32{}
				return func(ctx context.Context) (*int, bool, error) {
					val := int(counter.Add(1))
					// Return error after 3 successful polls
					if val > 3 {
						return nil, false, errors.New("simulated error")
					}
					return &val, false, nil
				}
			},
			interval: 50 * time.Millisecond,
			duration: 250 * time.Millisecond,
			check: func(t *testing.T, res []int) {
				// Should have collected exactly 3 values before error
				assert.Len(t, res, 3, "should have collected exactly 3 values before error")

				// Values should be sequential 1, 2, 3
				for i, val := range res {
					assert.Equal(t, i+1, val, "values should be sequential")
				}
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink(
				Poll(tt.poll(), tt.interval),
				test.CheckItems(t, func(t *testing.T, seen []int) {
					tt.check(t, seen)
				}),
				sinks.Noop[int](),
			)

			resChan := stream.Run(ctx)
			time.Sleep(tt.duration)
			stream.Drain()
			res := <-resChan

			if tt.expectErr {
				assert.Error(t, res.Err)
			} else {
				assert.NoError(t, res.Err)
			}
		})
	}
}
