package sources

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
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

			stream := compose.SourceToSink(
				Repeat(tt.element),
				sinks.Slice[int](),
			)

			resChan := stream.Run(ctx)
			time.Sleep(tt.duration)
			stream.Drain()
			res := <-resChan

			assert.Greater(t, len(res.Value), 1)
			for _, val := range res.Value {
				assert.Equal(t, tt.element, val, "every item should equal the repeated element")
			}
			assert.NoError(t, res.Err)
		})
	}
}
