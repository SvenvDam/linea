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
//   - setup: Function called to initialize and start the sink. It receives:
//   - ctx: Context used to control cancellation
//   - cancel: Function to cancel execution
//   - wg: WaitGroup to coordinate goroutine completion
//   - complete: Channel used to signal graceful shutdown, when closed the sink
//     will stop accepting new items but continue processing remaining ones
//   - setupUpstream: The setup function of the upstream component, allowing composition
//     of pipeline components through function composition
//     The setup function returns a channel that provides the sink's final result
type Sink[I, R any] struct {
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
		setupUpstream setupFunc[I],
	) <-chan Item[R]
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
//   - complete: Function to signal upstream that processing is complete
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
	onElem func(ctx context.Context, in I, acc R, cancel context.CancelFunc, complete CompleteFunc) (R, bool),
	onErr func(ctx context.Context, err error, acc R, cancel context.CancelFunc, complete CompleteFunc) (error, bool),
) *Sink[I, R] {
	if onErr == nil {
		onErr = func(ctx context.Context, err error, acc R, cancel context.CancelFunc, complete CompleteFunc) (error, bool) {
			return err, false
		}
	}

	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
		setupUpstream setupFunc[I],
	) <-chan Item[R] {
		out := make(chan Item[R])

		completeUpstreamChan, completeUpstream := util.NewCompleteChannel()

		in := setupUpstream(ctx, cancel, wg, completeUpstreamChan)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(out)
			defer completeUpstream()
			acc := Item[R]{Value: initial}
			for {
				select {
				case <-ctx.Done():
					util.Send(ctx, acc, out)
					return
				case <-complete:
					completeUpstream()
				case elem, ok := <-in:
					if !ok {
						util.Send(ctx, acc, out)
						return
					}

					var proceed bool
					if elem.Err != nil {
						acc.Err, proceed = onErr(ctx, elem.Err, acc.Value, cancel, completeUpstream)
					} else {
						acc.Value, proceed = onElem(ctx, elem.Value, acc.Value, cancel, completeUpstream)
					}
					if !proceed {
						util.Send(ctx, acc, out)
						return
					}
				}
			}
		}()

		return out
	}

	return &Sink[I, R]{
		setup: setup,
	}
}
