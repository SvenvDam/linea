package core

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/svenvdam/linea/util"
)

// Stream represents a complete stream processing pipeline consisting of a source,
// optional flows, and a sink. It manages the lifecycle of the stream and coordinates
// the execution of all components.
//
// Type Parameters:
//   - R: The type of the final result produced by the stream
//
// Fields:
//   - isRunning: Indicates whether the stream is currently executing
//   - cancel: Function to cancel stream execution
//   - drain: Channel used to signal graceful shutdown
//   - wg: WaitGroup to coordinate goroutine completion
//   - res: Channel that receives the stream results
//   - run: Function called to initialize and start the stream
type Stream[R any] struct {
	isRunning atomic.Bool
	cancel    context.CancelFunc
	drain     chan struct{}
	wg        *sync.WaitGroup
	res       <-chan Result[R]
	run       func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		drain chan struct{},
	)
}

// newStream creates a new Stream with the provided run function and result channel.
//
// Parameters:
//   - setup: Function that sets up and coordinates the stream execution
//
// Type Parameters:
//   - R: The type of the final result produced by the stream
//
// Returns a configured Stream ready to be run
func newStream[R any](
	setup func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		drain chan struct{},
	) <-chan R,
) *Stream[R] {
	stream := &Stream[R]{
		isRunning: atomic.Bool{},
		cancel:    nil,
		drain:     make(chan struct{}),
		wg:        &sync.WaitGroup{},
		res:       nil,
	}

	out := make(chan Result[R])
	stream.res = out

	stream.run = func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		drain chan struct{},
	) {
		res := setup(ctx, cancel, wg, drain)
		stream.isRunning.Store(true)

		wg.Add(1)
		go func() {
			defer close(out)
			defer cancel()
			defer wg.Done()
			defer func() {
				stream.isRunning.Store(false)
			}()

			select {
			case <-ctx.Done():
				return
			case r, ok := <-res:
				if !ok {
					return
				}
				util.Send(ctx, Result[R]{Value: r, Ok: true}, out)
			}
		}()
	}

	return stream
}

// Run starts the stream execution with the provided context.
// It initializes all components and begins processing items through the pipeline.
//
// Parameters:
//   - ctx: Context used to control the stream's lifecycle
//
// Returns a channel that will receive Result[R] values containing the stream's output
func (s *Stream[R]) Run(ctx context.Context) <-chan Result[R] {
	if !s.isRunning.Load() {
		ctx, cancel := context.WithCancel(ctx)
		s.cancel = cancel
		s.run(ctx, cancel, s.wg, s.drain)
	}

	return s.res
}

// Cancel cancels the stream's context and waits for all goroutines to complete.
// This performs an immediate shutdown of the stream.
func (s *Stream[R]) Cancel() {
	if s.isRunning.Load() {
		s.cancel()
		s.wg.Wait()
	}
}

// Drain signals the stream to stop accepting new items and process only the
// remaining items in the pipeline. This performs a graceful shutdown of the stream.
func (s *Stream[R]) Drain() {
	if s.isRunning.Load() {
		close(s.drain)
	}
}
