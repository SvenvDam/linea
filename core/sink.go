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

// NewSink creates a new Sink that processes items using the provided functions.
// The sink accumulates results by applying onElem to each input item, starting with
// the initial value.
//
// The Sink automatically handles:
//   - Goroutine lifecycle management
//   - Context cancellation propagation
//   - Channel cleanup on completion
//
// Parameters:
//   - initial: The initial value of the accumulator
//   - onElem: A function called for each input element to update the accumulator.
//
// onElem receives:
//   - ctx: A context for cancellation
//   - in: The current input element
//   - acc: The current accumulator value
//   - cancel: Function to cancel the sink's context
//
// onElem returns the new accumulator value
//
// Type Parameters:
//   - I: The type of items consumed by this sink
//   - R: The type of the final result
//
// Returns:
//   - A configured Sink ready to be connected to a stream
func NewSink[I, R any](
	initial R,
	onElem func(ctx context.Context, in I, acc R, cancel context.CancelFunc) R,
) *Sink[I, R] {
	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	) <-chan R {
		out := make(chan R)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(out)
			acc := initial
			for {
				select {
				case <-ctx.Done():
					util.Send(ctx, acc, out)
					return
				case elem, ok := <-in:
					if !ok {
						util.Send(ctx, acc, out)
						return
					}
					acc = onElem(ctx, elem, acc, cancel)
				}
			}
		}()

		return out
	}

	return &Sink[I, R]{
		setup: setup,
	}
}
