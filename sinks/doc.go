// Package sinks provides terminal components for stream processing pipelines.
//
// Sinks are the endpoints of a pipeline, consuming items and producing final results.
// This package includes common sink implementations.
//
// Sinks can be connected to sources and flows using the core package's
// composition functions.
//
// Example:
//
//	sink := sinks.Slice[int]()
//	// or
//	sink := sinks.Reduce(0, func(acc, i int) int { return acc + i })
package sinks
