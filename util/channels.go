package util

import (
	"context"
)

// SourceLoop processes elements from an input channel and handles completion.
// Type parameters:
//   - O: The type of elements received from the input channel
//
// Parameters:
//   - ctx: Context for cancellation
//   - out: Output channel for downstream processing
//   - drain: Channel to signal when the source should drain
//   - generate: Function that takes a context and returns a channel of type O
func SourceLoop[O any](
	ctx context.Context,
	out chan<- O,
	drain chan struct{},
	generate func(ctx context.Context) <-chan O,
) {
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
			case <-ctx.Done():
				return
			case <-drain:
				return
			case out <- elem:
			}
		}
	}
}

// ProcessLoop processes elements from an input channel and handles completion.
// Type parameters:
//   - I: The type of elements received from the input channel
//   - O: The type of elements that can be sent to the output channel
//
// Parameters:
//   - ctx: Context for cancellation
//   - in: Input channel to receive elements from
//   - out: Output channel for downstream processing
//   - onElem: Callback function executed for each received element
//   - onDone: Callback function executed when processing is complete
//     (either due to channel close or context cancellation)
func ProcessLoop[I, O any](
	ctx context.Context,
	in <-chan I,
	out chan<- O,
	onElem func(I),
	onDone func(),
) {
	for {
		select {
		case <-ctx.Done():
			onDone()
			return
		case elem, ok := <-in:
			if !ok {
				onDone()
				return
			}
			onElem(elem)
		}
	}
}

// SinkLoop accumulates a result by processing elements from an input channel.
// Type parameters:
//   - I: The type of elements received from the input channel
//   - R: The type of the accumulator/result
//
// Parameters:
//   - ctx: Context for cancellation
//   - in: Input channel to receive elements from
//   - initial: Initial value for the accumulator
//   - onElem: Function that processes each element and returns updated accumulator value
//
// Returns:
//   - The final accumulated result
func SinkLoop[I, R any](
	ctx context.Context,
	in <-chan I,
	initial R,
	onElem func(I, R) R,
) R {
	acc := initial
	for {
		select {
		case <-ctx.Done():
			return acc
		case elem, ok := <-in:
			if !ok {
				return acc
			}
			acc = onElem(elem, acc)
		}
	}
}

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
	case out <- elem:
	case <-ctx.Done():
		return
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
		case out <- elem:
		case <-ctx.Done():
			return
		}
	}
}
