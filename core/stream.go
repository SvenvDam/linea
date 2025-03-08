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
//   - complete: Channel used to signal graceful shutdown to all components in the pipeline
//   - wg: WaitGroup to coordinate goroutine completion
//   - res: Channel that receives the stream results
//   - run: Function called to initialize and start the stream
type Stream[R any] struct {
	isRunning atomic.Bool
	cancel    context.CancelFunc
	complete  context.CancelFunc
	wg        *sync.WaitGroup
	res       <-chan Item[R]
	run       func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
	)
}

// newStream creates a new Stream with the provided setup function.
//
// Parameters:
//   - setup: Function that sets up and coordinates the stream execution.
//
// The setup function receives:
//   - ctx: Context used to control cancellation
//   - cancel: Function to cancel execution
//   - wg: WaitGroup to coordinate goroutine completion
//   - complete: Channel used to signal graceful shutdown
//     It returns a channel that receives the stream's results
//
// Type Parameters:
//   - R: The type of the final result produced by the stream
//
// Returns a configured Stream ready to be run.
func newStream[R any](
	setup setupFunc[R],
) *Stream[R] {
	stream := &Stream[R]{
		isRunning: atomic.Bool{},
		cancel:    nil,
		complete:  nil,
		wg:        &sync.WaitGroup{},
		res:       nil,
	}

	out := make(chan Item[R], 1)
	stream.res = out

	stream.run = func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		complete <-chan struct{},
	) {
		res := setup(ctx, cancel, wg, complete)
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
				out <- Item[R]{Err: ctx.Err()}
				return
			case r, ok := <-res:
				if !ok {
					return
				}
				util.Send(ctx, r, out)
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
// Returns a channel that will receive Item[R] values containing the stream's output
func (s *Stream[R]) Run(ctx context.Context) <-chan Item[R] {
	if !s.isRunning.Load() {
		ctx, cancel := context.WithCancel(ctx)
		s.cancel = cancel

		complete, completeFn := util.NewCompleteChannel()
		s.complete = completeFn
		s.run(ctx, cancel, s.wg, complete)
	}

	return s.res
}

// Cancel cancels the stream's context and waits for all goroutines to complete.
// This performs an immediate shutdown of the stream.
func (s *Stream[R]) Cancel() {
	if s.isRunning.Load() {
		s.cancel()
	}
}

// Drain signals the stream to stop accepting new items and process only the
// remaining items in the pipeline. This performs a graceful shutdown of the stream.
// This method returns immediately and does not block - to wait for all items to be
// processed, continue reading from the stream's result channel until it closes.
func (s *Stream[R]) Drain() {
	if s.isRunning.Load() {
		s.complete()
	}
}

// AwaitDone blocks until all goroutines in the stream have completed.
// Use this method to wait for all processing to finish after calling Cancel or Drain.
// It waits on the internal sync.WaitGroup that is passed to all setup functions
// in the pipeline to coordinate goroutine completion.
func (s *Stream[R]) AwaitDone() {
	if s.isRunning.Load() {
		s.wg.Wait()
	}
}
