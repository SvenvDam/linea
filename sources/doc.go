// Package sources provides implementations of Source components for stream processing.
//
// Sources are the entry points of a processing pipeline, producing items that flow
// through the system. This package includes common source implementations.
//
// Each source can be configured with options and connected to flows or sinks
// using the core package's composition functions.
//
// Example:
//
//	source := sources.Slice([]int{1, 2, 3})
//	// or
//	source := sources.Repeat(42)
package sources
