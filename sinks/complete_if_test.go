package sinks

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestCompleteIf(t *testing.T) {
	tests := []struct {
		name     string
		elements []int
		pred     func(int) bool
		ok       bool
	}{
		{
			name:     "completes gracefully on matching item",
			elements: []int{1, 2, 3, 4, 5},
			pred:     func(i int) bool { return i == 3 },
			ok:       true,
		},
		{
			name:     "passes all items when predicate never matches",
			elements: []int{1, 2, 3},
			pred:     func(i int) bool { return false },
			ok:       true,
		},
		{
			name:     "completes gracefully on first item",
			elements: []int{1, 2, 3},
			pred:     func(i int) bool { return true },
			ok:       true,
		},
		{
			name:     "handles empty input",
			elements: []int{},
			pred:     func(i int) bool { return true },
			ok:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			stream := compose.SourceToSink(
				sources.Slice(tt.elements),
				CompleteIf(tt.pred),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
		})
	}
}

// TestCompleteIfWithInFlightMessages verifies that all in-flight messages
// are processed even when completion is triggered in the middle of processing.
// This demonstrates the graceful shutdown behavior of CompleteIf compared to CancelIf.
func TestCompleteIfWithInFlightMessages(t *testing.T) {
	input := []int{1, 2, 3, 4, 5}
	expected := []int{1, 2, 3, 4, 5} // All items should be processed

	// Create a predicate that triggers completion on seeing value 3
	// and adds a small delay to allow multiple values to be in-flight
	var slowPred = func(i int) bool {
		time.Sleep(50 * time.Millisecond)
		return i == 3
	}

	ctx := context.Background()

	// First, collect items through a flow to verify they are all processed
	stream := compose.SourceThroughFlowToSink(
		sources.Slice(input),
		test.CheckItems(t, func(t *testing.T, items []int) {
			assert.Equal(t, expected, items)
		}, core.WithFlowBufSize(5)),
		CompleteIf(slowPred),
	)

	res := <-stream.Run(ctx)
	assert.NoError(t, res.Err)
}
