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
//
// Parameters:
//   - opts: Optional SourceOption functions to configure the source
//   - generate: A function that takes a context and returns a channel of type O.
//     It receives:
//   - ctx: A context for cancellation
//
// Type Parameters:
//   - O: The type of items produced by this source
//
// Returns a configured Source ready to be started
func NewSource[O any](
	generate func(ctx context.Context) <-chan O,
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

			in := generate(ctx)
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
					case out <- elem:
					case <-drain:
						return
					case <-ctx.Done():
						return
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
