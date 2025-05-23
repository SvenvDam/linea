package core

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/util"
)

func TestConnectFlows(t *testing.T) {
	// Create two simple flows: int -> string -> int
	flow1 := NewFlow(
		func(ctx context.Context, elem int, out chan<- Item[string]) StreamAction {
			out <- Item[string]{Value: strconv.Itoa(elem)}
			return ActionProceed
		},
		nil,
		nil,
		nil,
	)

	flow2 := NewFlow(
		func(ctx context.Context, elem string, out chan<- Item[int]) StreamAction {
			val, _ := strconv.Atoi(elem)
			out <- Item[int]{Value: val * 2}
			return ActionProceed
		},
		nil,
		nil,
		nil,
	)

	// Connect the flows
	combined := ConnectFlows(flow1, flow2)

	// Create channels for testing
	in := make(chan Item[int])
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	completeChan, _ := util.NewCompleteChannel()
	setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
		return in
	}

	// Set up the combined flow
	out := combined.setup(ctx, cancel, &wg, completeChan, setup)

	// Test the transformation
	go func() {
		in <- Item[int]{Value: 42}
		close(in)
	}()

	// Verify the result
	result := <-out
	assert.Equal(t, 84, result.Value) // 42 -> "42" -> 84
	assert.NoError(t, result.Err)

	// Verify channel closes
	_, ok := <-out
	assert.False(t, ok)
}

func TestAppendFlowToSource(t *testing.T) {
	// Create a simple source that emits one number
	source := NewSource(
		func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[int] {
			out := make(chan Item[int])
			go func() {
				defer close(out)
				out <- Item[int]{Value: 42}
			}()
			return out
		},
	)

	// Create a flow that doubles the number
	flow := NewFlow(
		func(ctx context.Context, elem int, out chan<- Item[int]) StreamAction {
			out <- Item[int]{Value: elem * 2}
			return ActionProceed
		},
		nil,
		nil,
		nil,
	)

	// Combine source and flow
	combined := AppendFlowToSource(source, flow)

	// Set up the combined source
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	drain := make(chan struct{})
	out := combined.setup(ctx, cancel, &wg, drain)

	// Verify the transformation
	result := <-out
	assert.Equal(t, 84, result.Value) // 42 -> 84
	assert.NoError(t, result.Err)

	// Verify channel closes
	_, ok := <-out
	assert.False(t, ok)
}

func TestPrependFlowToSink(t *testing.T) {
	// Create a flow that converts int to string
	flow := NewFlow(
		func(ctx context.Context, elem int, out chan<- Item[string]) StreamAction {
			out <- Item[string]{Value: strconv.Itoa(elem)}
			return ActionProceed
		},
		nil,
		nil,
		nil,
	)

	// Create a sink that concatenates strings
	sink := NewSink(
		"",
		func(ctx context.Context, elem string, acc Item[string]) (Item[string], StreamAction) {
			return Item[string]{Value: acc.Value + elem}, ActionProceed
		},
		nil,
		nil,
	)

	// Combine flow and sink
	combined := PrependFlowToSink(flow, sink)

	// Set up the combined sink
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	in := make(chan Item[int])
	completeChan, _ := util.NewCompleteChannel()
	setup := func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[int] {
		return in
	}
	out := combined.setup(ctx, cancel, &wg, completeChan, setup)

	// Test the transformation
	go func() {
		in <- Item[int]{Value: 42}
		close(in)
	}()

	// Verify the result
	result := <-out
	assert.Equal(t, "42", result.Value) // 42 -> "42"
	assert.NoError(t, result.Err)

	// Verify channel closes
	_, ok := <-out
	assert.False(t, ok)
}

func TestConnectSourceToSink(t *testing.T) {
	// Create a source that emits one number
	source := NewSource(
		func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[int] {
			out := make(chan Item[int])
			go func() {
				defer close(out)
				out <- Item[int]{Value: 42}
			}()
			return out
		},
	)

	// Create a sink that converts to string
	sink := NewSink(
		"",
		func(ctx context.Context, elem int, acc Item[string]) (Item[string], StreamAction) {
			return Item[string]{Value: acc.Value + strconv.Itoa(elem)}, ActionProceed
		},
		nil,
		nil,
	)

	// Create the stream
	stream := ConnectSourceToSink(source, sink)

	// Run the stream
	ctx := context.Background()
	result := <-stream.Run(ctx)

	// Verify the result
	assert.NoError(t, result.Err)
	assert.Equal(t, "42", result.Value)
}
