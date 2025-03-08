package flows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestCompleteIf(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		pred   func(int) bool
		expect []int
		ok     bool
	}{
		{
			name:   "completes after matching item but continues current items",
			input:  []int{1, 2, 3, 4, 5},
			pred:   func(i int) bool { return i == 3 },
			expect: []int{1, 2, 3},
			ok:     true,
		},
		{
			name:   "passes all items when predicate never matches",
			input:  []int{1, 2, 3},
			pred:   func(i int) bool { return false },
			expect: []int{1, 2, 3},
			ok:     true,
		},
		{
			name:   "completes after first item",
			input:  []int{1, 2, 3},
			pred:   func(i int) bool { return true },
			expect: []int{1},
			ok:     true,
		},
		{
			name:   "handles empty input",
			input:  []int{},
			pred:   func(i int) bool { return true },
			expect: []int{},
			ok:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceThroughFlowToSink(
				sources.Slice(tt.input),
				CompleteIf(tt.pred),
				sinks.Slice[int](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
			assert.Equal(t, tt.expect, res.Value)
		})
	}
}

// TestCompleteIfWithInFlightMessages verifies that all in-flight messages
// are processed even when completion is triggered in the middle of processing.
// This demonstrates the graceful shutdown behavior of CompleteIf compared to CancelIf.
func TestCompleteIfWithInFlightMessages(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	inChan := make(chan int, 5)

	for _, i := range input {
		inChan <- i
	}
	close(inChan)

	// Create a predicate that triggers completion on seeing value 3
	// and adds a small delay to allow multiple values to be in-flight
	var slowPred = func(i int) bool {
		time.Sleep(50 * time.Millisecond)
		return i == 3
	}

	ctx := context.Background()

	// Use a source with buffer to allow multiple items to be "in-flight"
	stream := compose.SourceThroughFlowToSink(
		sources.Chan(inChan, core.WithSourceBufSize(5)),
		CompleteIf(slowPred),
		sinks.Slice[int](),
	)

	res := <-stream.Run(ctx)
	assert.NoError(t, res.Err)

	// Verify all items were processed, demonstrating that CompleteIf allows
	// in-flight items to be processed unlike CancelIf which would stop immediately
	assert.Equal(t, input, res.Value)
}
