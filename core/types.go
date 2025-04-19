package core

import (
	"context"
	"sync"
)

// setupFunc is a function type used to initialize and coordinate stream components.
// It's used throughout the core package to connect sources, flows, and sinks together.
//
// Parameters:
//   - ctx: Context used to control cancellation
//   - cancel: Function to cancel execution
//   - wg: WaitGroup to coordinate goroutine completion
//   - complete: Channel used to signal graceful shutdown
//
// Type Parameters:
//   - T: The type of data flowing through the stream component
//
// Returns a channel that receives stream data of type T
type setupFunc[T any] func(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, complete <-chan struct{}) <-chan Item[T]

// CompleteFunc is a function type used to signal graceful shutdown.
// It can be called multiple times, but will only close the channel once.
type CompleteFunc func()

// StreamAction represents instructions returned by stream handlers like onElem and onErr
// to control the flow of data processing in a stream. It indicates what action the stream
// should take after processing an element or encountering an error.
type StreamAction int

const (
	// ActionProceed indicates that stream processing should continue to the next element.
	ActionProceed StreamAction = iota

	// ActionStop signals that the stream should stop processing.
	// Any in-flight message upstream will be discarded but downstream messages will still be processed.
	ActionStop

	// ActionCancel signals that the stream should be cancelled immediately.
	// This will cause the stream to stop processing immediately and discard any in-flight messages.
	ActionCancel

	// ActionComplete signals that the stream should perform a graceful shutdown.
	// All in-flight messages in the stream will still be processed but no new messages will be accepted.
	ActionComplete

	// ActionRestartUpstream signals that the stream should restart its upstream component.
	ActionRestartUpstream
)
