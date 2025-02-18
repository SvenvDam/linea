package core

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectFlows(t *testing.T) {
	// Create two simple flows: int -> string -> int
	flow1 := NewFlow(func(ctx context.Context, elem int, out chan<- string, cancel context.CancelFunc) bool {
		out <- strconv.Itoa(elem)
		return true
	}, func(ctx context.Context, out chan<- string) {})

	flow2 := NewFlow(func(ctx context.Context, elem string, out chan<- int, cancel context.CancelFunc) bool {
		val, _ := strconv.Atoi(elem)
		out <- val * 2
		return true
	}, func(ctx context.Context, out chan<- int) {})

	// Connect the flows
	combined := ConnectFlows(flow1, flow2)

	// Create channels for testing
	in := make(chan int)
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up the combined flow
	out := combined.setup(ctx, cancel, &wg, in)

	// Test the transformation
	go func() {
		in <- 42
		close(in)
	}()

	// Verify the result
	result := <-out
	assert.Equal(t, 84, result) // 42 -> "42" -> 84

	// Verify channel closes
	_, ok := <-out
	assert.False(t, ok)
}

func TestAppendFlowToSource(t *testing.T) {
	// Create a simple source that emits one number
	source := NewSource(func(ctx context.Context, drain <-chan struct{}) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			out <- 42
		}()
		return out
	})

	// Create a flow that doubles the number
	flow := NewFlow(func(ctx context.Context, elem int, out chan<- int, cancel context.CancelFunc) bool {
		out <- elem * 2
		return true
	}, func(ctx context.Context, out chan<- int) {})

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
	assert.Equal(t, 84, result) // 42 -> 84

	// Verify channel closes
	_, ok := <-out
	assert.False(t, ok)
}

func TestPrependFlowToSink(t *testing.T) {
	// Create a flow that converts int to string
	flow := NewFlow(func(ctx context.Context, elem int, out chan<- string, cancel context.CancelFunc) bool {
		out <- strconv.Itoa(elem)
		return true
	}, func(ctx context.Context, out chan<- string) {})

	// Create a sink that concatenates strings
	sink := NewSink("",
		func(ctx context.Context, elem string, acc string, cancel context.CancelFunc) string {
			return acc + elem
		},
	)

	// Combine flow and sink
	combined := PrependFlowToSink(flow, sink)

	// Set up the combined sink
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var wg sync.WaitGroup
	in := make(chan int)
	out := combined.setup(ctx, cancel, &wg, in)

	// Test the transformation
	go func() {
		in <- 42
		close(in)
	}()

	// Verify the result
	result := <-out
	assert.Equal(t, "42", result) // 42 -> "42"

	// Verify channel closes
	_, ok := <-out
	assert.False(t, ok)
}

func TestConnectSourceToSink(t *testing.T) {
	// Create a source that emits one number
	source := NewSource(func(ctx context.Context, drain <-chan struct{}) <-chan int {
		out := make(chan int)
		go func() {
			defer close(out)
			out <- 42
		}()
		return out
	})

	// Create a sink that converts to string
	sink := NewSink("",
		func(ctx context.Context, elem int, acc string, cancel context.CancelFunc) string {
			return acc + strconv.Itoa(elem)
		},
	)

	// Create the stream
	stream := ConnectSourceToSink(source, sink)

	// Run the stream
	ctx := context.Background()
	result := <-stream.Run(ctx)

	// Verify the result
	assert.True(t, result.Ok)
	assert.Equal(t, "42", result.Value)
}
