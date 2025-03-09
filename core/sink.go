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

// DefaultSinkErrorHandler is the default implementation for handling errors in a Sink.
// It returns the error as-is and stops further processing by returning false.
//
// Parameters:
//   - ctx: Context used for cancellation
//   - err: The error that occurred
//   - acc: The current accumulator value
//   - cancel: Function to cancel execution
//   - complete: Function to signal graceful shutdown
//
// Returns:
//   - The original error
//   - false to stop processing
func DefaultSinkErrorHandler[R any](
	ctx context.Context,
	err error,
	acc R,
	cancel context.CancelFunc,
	complete CompleteFunc,
) (error, bool) {
	return err, false
}

// NewSink creates a terminal component in a data processing pipeline that consumes incoming data
// and produces a final result. It acts as an accumulator, processing each incoming item
// and updating a result value.
//
// The Sink automatically handles:
//   - Goroutine lifecycle management
//   - Context cancellation propagation
//   - Channel cleanup on completion
//
// Parameters:
//   - initial: The initial value of the accumulator that will be used as the starting point
//   - onElem: A function called for each input element to update the accumulator
//   - onErr: A function called when an error is encountered in the input stream
//
// onElem receives:
//   - ctx: A context for cancellation
//   - in: The current input element
//   - acc: The current accumulator value
//   - cancel: Function to cancel the sink's context
//   - complete: Function to signal upstream that processing is complete
//
// onElem returns:
//   - The new accumulator value
//   - A boolean indicating whether to continue processing (true) or stop (false)
//
// onErr receives:
//   - ctx: A context for cancellation
//   - err: The error that was encountered
//   - acc: The current accumulator value
//   - cancel: Function to cancel the sink's context
//   - complete: Function to signal upstream that processing is complete
//
// onErr returns:
//   - The error to be included in the final result (can be modified)
//   - A boolean indicating whether to continue processing (true) or stop (false)
//   - If nil is provided, a default handler will be used that returns the error and stops processing
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
		onErr = DefaultSinkErrorHandler[R]
	}

	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
		setupUpstream setupFunc[I],
	) <-chan Item[R] {
		out := make(chan Item[R], 1)

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
					return
				case <-complete:
					completeUpstream()
				case elem, ok := <-in:
					if !ok {
						out <- acc
						return
					}

					var proceed bool
					if elem.Err != nil {
						acc.Err, proceed = onErr(ctx, elem.Err, acc.Value, cancel, completeUpstream)
					} else {
						acc.Value, proceed = onElem(ctx, elem.Value, acc.Value, cancel, completeUpstream)
					}
					if !proceed {
						out <- acc
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
