package core

import (
	"context"
	"sync"

	"github.com/svenvdam/linea/util"
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
//   - setup: A function that initializes the Flow's goroutine and connects it to the input channel.
//
// It receives:
//   - ctx: Context used to control cancellation
//   - cancel: Function to cancel execution
//   - wg: WaitGroup to coordinate goroutine completion
//   - complete: Channel used to signal graceful shutdown, when closed the flow
//     will stop accepting new items but continue processing remaining ones
//   - setupUpstream: The setup function of the upstream component, allowing composition
//     of pipeline components through function composition
//     The setup function returns a channel that provides the flow's output items
type Flow[I, O any] struct {
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
		setupUpstream setupFunc[I],
	) <-chan Item[O]
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

// DefaultFlowErrorHandler is the default implementation for handling errors in a Flow.
// It sends the error downstream and stops the flow by returning ActionStop.
func DefaultFlowErrorHandler[O any](ctx context.Context, err error, out chan<- Item[O]) StreamAction {
	util.Send(ctx, Item[O]{Err: err}, out)
	return ActionStop
}

// DefaultFlowDoneHandler is the default implementation for cleanup when a Flow completes.
// It performs no operations and is used when no custom cleanup is needed.
func DefaultFlowDoneHandler[O any](ctx context.Context, out chan<- Item[O]) {
	// No operation by default
}

// DefaultFlowUpstreamClosedHandler is the default implementation for closing a Flow.
// It returns ActionStop to stop the flow.
func DefaultFlowUpstreamClosedHandler[O any](ctx context.Context, out chan<- Item[O]) StreamAction {
	return ActionStop
}

// NewFlow creates a new Flow that transforms input items to output items using
// the provided process function.
//
// The Flow automatically handles:
//   - Goroutine lifecycle management
//   - Context cancellation propagation
//   - Channel cleanup on completion
//
// Parameters:
//   - onElem: A function called for each input element.
//   - onDone: A cleanup function called when the flow stops.
//   - opts: Optional FlowOption functions to configure the flow
//
// onElem is responsible for:
//   - Transforming the input item
//   - Sending the transformed item to the output channel
//   - Returning true to continue processing, false to stop the flow
//
// # It receives the current context, input element, output channel, cancel function, and complete function
//
// onErr is responsible for:
//   - Handling upstream errors that occur during processing
//   - Sending error information to the output channel if needed
//   - Returning true to continue processing, false to stop the flow
//
// # It receives the current context, the error, output channel, cancel function, and complete function
// # If nil is provided, a default handler will be used that sends the error and stops the flow
//
// onDone is responsible for:
//   - Performing final operations and cleanup
//   - Sending remaining items to the output channel
//   - Handling cleanup of any resources created during processing
//
// # It receives the current context and output channel
// # If nil is provided, an empty default implementation will be used
//
// Type Parameters:
//   - I: The type of input items
//   - O: The type of output items
//
// Returns:
//   - A new Flow instance that will perform the specified transformation
func NewFlow[I, O any](
	onElem func(ctx context.Context, elem I, out chan<- Item[O]) StreamAction,
	onErr func(ctx context.Context, err error, out chan<- Item[O]) StreamAction,
	onUpstreamClosed func(ctx context.Context, out chan<- Item[O]) StreamAction,
	onDone func(ctx context.Context, out chan<- Item[O]),
	opts ...FlowOption,
) *Flow[I, O] {
	cfg := &flowConfig{}

	if onErr == nil {
		onErr = DefaultFlowErrorHandler[O]
	}

	if onUpstreamClosed == nil {
		onUpstreamClosed = DefaultFlowUpstreamClosedHandler[O]
	}

	if onDone == nil {
		onDone = DefaultFlowDoneHandler[O]
	}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
		setupUpstream setupFunc[I],
	) <-chan Item[O] {
		out := make(chan Item[O], cfg.bufSize)
		completeUpstreamChan, completeUpstream := util.NewCompleteChannel()
		in := setupUpstream(ctx, cancel, wg, completeUpstreamChan)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(out)
			defer onDone(ctx, out)
			defer completeUpstream()

			for {
				select {
				case <-ctx.Done():
					return
				case <-complete:
					completeUpstream()
				case elem, ok := <-in:
					var action StreamAction
					if !ok {
						action = onUpstreamClosed(ctx, out)
					} else if elem.Err != nil {
						action = onErr(ctx, elem.Err, out)
					} else {
						action = onElem(ctx, elem.Value, out)
					}

					switch action {
					case ActionProceed:
						continue
					case ActionStop:
						return
					case ActionCancel:
						cancel()
						return
					case ActionComplete:
						completeUpstream()
						continue
					case ActionRestartUpstream:
						completeUpstream()
						completeUpstreamChan, completeUpstream = util.NewCompleteChannel()
						in = setupUpstream(ctx, cancel, wg, completeUpstreamChan)
						continue
					}
				}
			}
		}()

		return out
	}

	f := &Flow[I, O]{
		setup: setup,
	}

	return f
}
