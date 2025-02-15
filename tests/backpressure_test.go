package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestBackpressure(t *testing.T) {
	waitDuration := 20 * time.Millisecond
	sleepDuration := 500 * time.Millisecond
	tests := []struct {
		name    string
		bufSize int
	}{
		{
			name:    "no buffer",
			bufSize: 0,
		},
		{
			name:    "large buffer",
			bufSize: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			seenTimes := make([]time.Time, 0)

			timeLogFlow := flows.ForEach(func(elem int) {
				seenTimes = append(seenTimes, time.Now())
			}, core.WithFlowBufSize(tt.bufSize))

			stream := core.SourceThroughFlowToSink(
				sources.Repeat(1), // fast source
				timeLogFlow,
				sinks.ForEach(func(elem int) { // slow sink
					time.Sleep(waitDuration)
				}),
			)

			streamCapacity := 2 + tt.bufSize // stream of 3 components has 2 in-flight + buffer size

			resChan := stream.Run(ctx)

			time.Sleep(sleepDuration)

			stream.Drain()

			res := <-resChan
			assert.True(t, res.Ok)

			for i := 1; i < len(seenTimes); i++ {
				if i < streamCapacity {
					assert.InDelta(t, 0, seenTimes[i].Sub(seenTimes[i-1]).Milliseconds(), 5)
				} else {
					assert.InDelta(t, waitDuration.Milliseconds(), seenTimes[i].Sub(seenTimes[i-1]).Milliseconds(), 5)
				}
			}
		})
	}
}
