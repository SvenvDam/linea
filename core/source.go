package core

import (
	"context"
	"sync"
)

// SourceOption is a function that configures a Source.
// It takes a sourceConfig pointer and modifies it to customize Source behavior.
type SourceOption func(*sourceConfig)

// sourceConfig holds configuration options for a Source.
type sourceConfig struct {
	// bufSize determines the buffer size of the output channel
	bufSize int
}

// WithSourceBufSize returns a SourceOption that sets the buffer size for the source's output channel.
// The buffer size controls how many items can be buffered in the source's output channel before blocking.
//
// Parameters:
//   - size: The desired buffer size for the output channel
func WithSourceBufSize(size int) SourceOption {
	return func(c *sourceConfig) {
		c.bufSize = size
	}
}

// Source is a source of items in a stream. It produces items of type O and sends them
// downstream through its output channel. Sources are lazy and do not start generating
// items until explicitly started.
//
// Type Parameters:
//   - O: The type of items produced by this source
//
// Fields:
//   - setup: Function called to initialize and start the source.
//
// It receives:
//   - ctx: Context used to control cancellation
//   - cancel: Function to cancel execution
//   - wg: WaitGroup to coordinate goroutine completion
//   - complete: Channel used to signal graceful shutdown, when closed the source
//     will stop generating new items but continue sending any remaining items
//
// The setup function returns a channel that provides the source's output items
type Source[O any] struct {
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
	) <-chan Item[O]
}

// NewSource creates a new data source that can be connected to other components in a data processing pipeline.
// It serves as the entry point for data flowing through the system.
//
// Parameters:
//   - generate: A function that implements the source's item generation logic.
//     It should return a channel from which items will be read and forwarded downstream.
//   - opts: Optional SourceOption functions to configure the source behavior
//     (e.g., WithSourceBufSize to set the output channel buffer size)
//
// The generate function receives the following arguments:
//   - ctx: A context for cancellation and coordination
//   - complete: A channel that closes when the source should stop producing new items
//   - cancel: A function to cancel execution if needed
//   - wg: A WaitGroup for synchronization with other components
//
// Type Parameters:
//   - O: The type of items that will be produced by this source
//
// Returns:
//   - A configured Source ready to be connected to a flow or sink
func NewSource[O any](
	generate func(ctx context.Context, complete <-chan struct{}, cancel context.CancelFunc, wg *sync.WaitGroup) <-chan Item[O],
	opts ...SourceOption,
) *Source[O] {
	cfg := &sourceConfig{}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
	) <-chan Item[O] {
		out := make(chan Item[O], cfg.bufSize)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(out)
			in := generate(ctx, complete, cancel, wg)

			for {
				select {
				case <-ctx.Done():
					return
				case <-complete:
					return
				case elem, ok := <-in:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case <-complete:
						return
					case out <- elem:
					}
				}
			}
		}()

		return out
	}

	source := &Source[O]{
		setup: setup,
	}

	return source
}
