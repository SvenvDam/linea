// Package flows provides transformation components for stream processing pipelines.
//
// Flows are the transformation stages in a pipeline, processing items as they pass
// through. This package includes common flow implementations.
//
// Flows can be composed using the core package's functions and configured with options.
//
// Example:
//
//	flow := flows.Filter(func(i int) bool { return i%2 == 0 })
//	// or
//	flow := flows.Map(func(i int) string { return strconv.Itoa(i) })
package flows
