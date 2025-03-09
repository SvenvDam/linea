package core

import (
	"context"
	"errors"
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
//   - complete: Function to signal graceful shutdown to all components in the pipeline
//   - wg: WaitGroup to coordinate goroutine completion
//   - res: Channel that receives the stream results
//   - run: Function called to initialize and start the stream
type Stream[R any] struct {
	isRunning atomic.Bool
	cancel    context.CancelFunc
	complete  CompleteFunc
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
// A Stream represents an executable data processing pipeline that can be started,
// cancelled, and awaited. It provides a uniform interface for executing and managing
// data processing operations.
//
// Parameters:
//   - setup: Function that sets up and coordinates the stream execution.
//     This setup function is responsible for connecting all components of the
//     processing pipeline and returning a channel with the final results.
//
// The setup function receives:
//   - ctx: Context used to control cancellation
//   - cancel: Function to cancel execution
//   - wg: WaitGroup to coordinate goroutine completion
//   - complete: Channel used to signal graceful shutdown
//
// The setup function returns:
//   - A channel that emits the stream's results as they are produced
//
// Type Parameters:
//   - R: The type of the final result produced by the stream
//
// Returns:
//   - A configured Stream object that can be controlled through its methods:
//   - Run: Starts the stream processing
//   - Cancel: Cancels the stream processing immediately
//   - Drain: Triggers graceful shutdown of the stream
//   - AwaitDone: Waits for all goroutines to complete
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
			defer stream.isRunning.Store(false)

			select {
			case <-ctx.Done():
				out <- Item[R]{Err: ctx.Err()}
				return
			case r, ok := <-res:
				if !ok {
					if ctx.Err() != nil {
						out <- Item[R]{Err: ctx.Err()}
					} else {
						out <- Item[R]{Err: errors.New("result channel closed unexpectedly")}
					}
					return
				}
				out <- r
				return
			}
		}()
	}

	return stream
}

// Run starts the stream execution with the provided context.
// It initializes all components and begins processing items through the pipeline.
// If the stream is already running, this method will not restart it and will
// simply return the existing result channel.
//
// Parameters:
//   - ctx: Context used to control the stream's lifecycle and cancellation
//
// Returns:
//   - A channel that will receive a single Item[R] value containing the stream's output result
//   - The channel will be closed when the stream completes or encounters an error
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

// Cancel cancels the stream's context and triggers immediate shutdown.
// This will stop all processing as soon as possible without waiting for
// in-flight items to complete. After cancellation, any items still in the
// pipeline may be lost.
//
// This method is non-blocking - to wait for all goroutines to complete
// after cancellation, call AwaitDone().
//
// If the stream is not running, this method has no effect.
func (s *Stream[R]) Cancel() {
	if s.isRunning.Load() {
		s.cancel()
	}
}

// Drain signals the stream to stop accepting new items and process only the
// remaining items in the pipeline. This performs a graceful shutdown of the stream.
//
// Unlike Cancel, Drain allows all components in the pipeline to finish processing
// any items they currently have. Sources will stop producing new items, but
// existing items will continue through the pipeline until completion.
//
// This method is non-blocking - to wait for all items to be processed, either:
//   - Continue reading from the stream's result channel until it closes, or
//   - Call AwaitDone() to block until all goroutines have completed
//
// If the stream is not running, this method has no effect.
func (s *Stream[R]) Drain() {
	if s.isRunning.Load() {
		s.complete()
	}
}

// AwaitDone blocks until all goroutines in the stream have completed.
// Use this method to wait for all processing to finish after calling Cancel or Drain.
//
// This method blocks on the internal sync.WaitGroup that coordinates the completion
// of all goroutines in the pipeline. It's useful when you need to ensure that all
// resources have been properly cleaned up before proceeding.
//
// If the stream is not running, this method returns immediately.
func (s *Stream[R]) AwaitDone() {
	s.wg.Wait()
}
