// Package compose provides high-level functions for composing stream processing pipelines.
//
// The compose package offers convenience functions that make it easy to connect
// sources, flows, and sinks together in various combinations. These functions handle
// the details of pipeline construction while maintaining type safety through Go generics.
//
// Example usage:
//
//	// Chain multiple transformations
//	source := compose.SourceThroughFlow2(
//	    sourceOfInts,
//	    flows.Filter(func(i int) bool { return i%2 == 0 }),
//	    flows.Map(strconv.Itoa),
//	)
//
//	// Build a complete pipeline
//	stream := compose.SourceThroughFlowToSink(
//	    sourceOfInts,
//	    flows.Map(strconv.Itoa),
//	    sinks.Slice[string](),
//	)
//
//	// Combine flows
//	flow := compose.MergeFlows(
//	    flows.Filter(func(i int) bool { return i > 0 }),
//	    flows.Map(strconv.Itoa),
//	)
//
// The functions in this package are designed to be composable, allowing for
// flexible construction of processing pipelines while maintaining type safety
// and readability.
package compose
