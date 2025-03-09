package sinks

import (
	"github.com/svenvdam/linea/core"
)

// Noop creates a Sink that consumes all items without performing any operation
// or producing any meaningful result. This is useful for cases where you want
// to drain a stream without processing its items.
//
// Type Parameters:
//   - I: The type of items to consume
//
// Returns a Sink that discards all items
func Noop[I any]() *core.Sink[I, struct{}] {
	return ForEach(func(_ I) {})
}
