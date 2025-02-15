package core

import (
	"context"
	"sync"
)

// Flow represents a transformation stage in a data processing pipeline.
// It takes input items of type I, processes them, and produces output items of type O.
// Flow is designed to be composable, allowing multiple transformations to be chained together.
//
// Type Parameters:
//   - I: The type of input items the Flow receives
//   - O: The type of output items the Flow produces
//
// Fields:
//   - out: The output channel that delivers transformed items
//   - setup: A function that initializes the Flow's goroutine and connects it to the input channel
type Flow[I, O any] struct {
	out   <-chan O
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	)
}

// FlowOption is a function type for configuring Flow behavior.
// It follows the functional options pattern, allowing optional parameters
// to be passed when creating a new Flow.
type FlowOption func(*flowConfig)

// flowConfig holds configuration options for a Flow.
//
// Fields:
//   - bufSize: The size of the buffer for the Flow's output channel
type flowConfig struct {
	bufSize int
}

// WithFlowBufSize creates a FlowOption that configures the buffer size of a Flow's output channel.
// A larger buffer size can improve performance by reducing blocking, but uses more memory.
//
// Parameters:
//   - size: The desired size of the output channel buffer
//
// Returns:
//   - A FlowOption that can be passed to NewFlow
func WithFlowBufSize(size int) FlowOption {
	return func(c *flowConfig) {
		c.bufSize = size
	}
}

// NewFlow creates a new Flow that transforms input items to output items using the provided process function.
//
// Parameters:
//   - process: A function that defines how to transform input items to output items.
//     It receives:
//   - ctx: A context for cancellation
//   - in: Input channel receiving items of type I
//   - out: Output channel for sending items of type O
//   - cancel: Function to cancel the flow's context
//   - opts: Optional FlowOption functions to configure the Flow
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Returns:
//   - A new Flow instance that will perform the specified transformation
func NewFlow[I, O any](
	process func(ctx context.Context, in <-chan I, out chan<- O, cancel context.CancelFunc),
	opts ...FlowOption,
) *Flow[I, O] {
	cfg := &flowConfig{
		bufSize: defaultBufSize,
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	out := make(chan O, cfg.bufSize)

	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(out)
			process(ctx, in, out, cancel)
		}()
	}

	f := &Flow[I, O]{
		out:   out,
		setup: setup,
	}

	return f
}
