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
//   - setup: Function called to initialize and start the source
type Source[O any] struct {
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		drain chan struct{},
	) <-chan O
}

// NewSource creates a new Source that produces items using the provided generate function.
// The source is lazy and won't start producing items until it is connected to a flow or sink
// and explicitly started.
//
// Parameters:
//   - generate: A function that implements the source's item generation logic.
//     It should return a channel from which items will be read and forwarded downstream.
//   - opts: Optional SourceOption functions to configure the source behavior
//     (e.g., WithSourceBufSize to set the output channel buffer size)
//
// The generate function receives the following arguments:
//   - ctx: A context for cancellation and coordination
//   - drain: A channel that closes when the source should stop producing new items
//
// Type Parameters:
//   - O: The type of items that will be produced by this source
//
// Returns:
//   - A configured Source instance that is ready to be connected to a flow or sink
func NewSource[O any](
	generate func(ctx context.Context, drain <-chan struct{}, cancel context.CancelFunc) <-chan O,
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
		drain chan struct{},
	) <-chan O {
		out := make(chan O, cfg.bufSize)

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer close(out)
			in := generate(ctx, drain, cancel)
			for {
				select {
				case <-ctx.Done():
					return
				case <-drain:
					return
				case elem, ok := <-in:
					if !ok {
						return
					}
					select {
					case <-ctx.Done():
						return
					case <-drain:
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
