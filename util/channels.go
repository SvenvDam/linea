package util

import (
	"context"
	"sync"
)

// Send attempts to send an element to a channel with context cancellation support.
// If the context is cancelled before the send operation completes, the function returns
// without sending the element.
//
// Type parameters:
//   - T: The type of element to send
//
// Parameters:
//   - ctx: Context for cancellation
//   - elem: The element to send
//   - out: The output channel to send the element to
func Send[T any](ctx context.Context, elem T, out chan<- T) {
	select {
	case <-ctx.Done():
		return
	case out <- elem:
	}
}

// SendMany attempts to send multiple elements to a channel with context cancellation support.
// If the context is cancelled before all elements are sent, the function returns without
// sending the remaining elements.
//
// Type Parameters:
//   - T: The type of elements to send
//
// Parameters:
//   - ctx: Context for cancellation
//   - elems: Slice of elements to send
//   - out: The output channel to send the elements to
func SendMany[T any](ctx context.Context, elems []T, out chan<- T) {
	for _, elem := range elems {
		select {
		case <-ctx.Done():
			return
		case out <- elem:
		}
	}
}

// NewCompleteChannel creates a new complete channel and a cancel function.
// The cancel function can be used to close the channel.
//
// Returns:
//   - complete: The complete channel
//   - cancel: The cancel function
func NewCompleteChannel() (chan struct{}, func()) {
	complete := make(chan struct{})
	once := sync.Once{}
	completeFn := func() {
		once.Do(func() {
			close(complete)
		})
	}
	return complete, completeFn
}
