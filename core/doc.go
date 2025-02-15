// Package core provides the foundational types and functions for building stream processing pipelines.
//
// The core package defines the main components that make up a processing pipeline:
//   - Source: Produces a stream of items
//   - Flow: Transforms items in the stream
//   - Sink: Consumes items and produces a final result
//   - Stream: Coordinates the execution of a complete pipeline
//
// These components can be composed using functions like:
//   - SourceThroughFlow: Attaches a Flow to a Source
//   - SourceToSink: Connects a Source directly to a Sink
//   - SourceThroughFlowToSink: Creates a complete pipeline with transformation
//
// The package also provides configuration options through the FlowOption and SourceOption types.
//
// Example:
//
//	stream := core.SourceThroughFlowToSink(
//	    mySource,
//	    myFlow,
//	    mySink,
//	)
//	result := <-stream.Run(ctx)
package core
