package core

import (
	"context"
	"sync"

	"github.com/svenvdam/linea/util"
)

// Sink represents the terminal stage of a stream that consumes items and produces a final result.
// It processes incoming items of type I and produces a result of type R.
//
// Type Parameters:
//   - I: The type of items consumed by this sink
//   - R: The type of the final result
//
// Fields:
//   - setup: Function called to initialize and start the sink
type Sink[I, R any] struct {
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	) <-chan R
}

// NewSink creates a new Sink that processes items using the provided process function.
//
// Parameters:
//   - process: A function that takes a context, input channel, and cancel function,
//     processes the incoming items, and returns a result of type R. It receives:
//   - ctx: A context for cancellation
//   - in: Input channel receiving items of type I
//   - cancel: Function to cancel the sink's context
//
// Type Parameters:
//   - I: The type of items consumed by this sink
//   - R: The type of the final result
//
// Returns a configured Sink ready to be connected to a stream
func NewSink[I, R any](
	process func(ctx context.Context, in <-chan I, cancel context.CancelFunc) R,
) *Sink[I, R] {
	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	) <-chan R {
		res := make(chan R)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(res)

			util.Send(ctx, process(ctx, in, cancel), res)
		}()

		return res
	}

	return &Sink[I, R]{
		setup: setup,
	}
}
