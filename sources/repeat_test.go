package sources

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/test"
)

func TestRepeat(t *testing.T) {
	tests := []struct {
		name     string
		element  int
		check    func(t *testing.T, res []int)
		duration time.Duration
	}{
		{
			name:    "emits same element multiple times",
			element: 42,
			check: func(t *testing.T, res []int) {
				assert.Greater(t, len(res), 1)
			},
			duration: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink2(
				Repeat(tt.element),
				test.AssertEachItem(t, func(t *testing.T, in int) {
					assert.Equal(t, tt.element, in)
				}),
				test.CheckItems(t, func(t *testing.T, seen []int) {
					assert.Greater(t, len(seen), 1)
					tt.check(t, seen)
				}),
				sinks.Noop[int](),
			)

			resChan := stream.Run(ctx)
			time.Sleep(tt.duration)
			stream.Drain()
			res := <-resChan

			assert.True(t, res.Ok)
		})
	}
}
