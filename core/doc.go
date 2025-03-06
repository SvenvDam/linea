// Package core provides the foundational types and functions for building stream processing pipelines.
//
// Note: Direct usage of this package is considered advanced usage. For most use cases:
//   - Use the flows, sinks, and sources packages to create pipeline components
//   - Use the compose package to connect components together
//
// The core package defines the main components that make up a processing pipeline:
//   - Source: Produces a stream of items
//   - Flow: Transforms items in the stream
//   - Sink: Consumes items and produces a final result
//   - Stream: Coordinates the execution of a complete pipeline
//
// Core Concepts:
//   - Setup Functions: Each component provides a setup function that initializes its
//     execution, receives context, cancellation, and completion signals, and returns
//     a channel for its output. These functions are composed when connecting components.
//   - Complete Signal Channels: Used to signal graceful shutdown through the pipeline.
//     When closed, components stop accepting new items but process remaining ones.
//
// While this package provides the building blocks for custom components, most users
// should prefer the pre-built components from the specialized packages:
//   - sources: Ready-to-use Source implementations (Slice, Chan, Repeat, etc.)
//   - flows: Common transformations (Map, Filter, Batch, etc.)
//   - sinks: Standard Sink implementations (Slice, Reduce, ForEach, etc.)
//
// These components can be composed using functions from the compose package:
//   - compose.SourceThroughFlow: Attaches a Flow to a Source
//   - compose.SourceToSink: Connects a Source directly to a Sink
//   - compose.SourceThroughFlowToSink: Creates a complete pipeline with transformation
//
// This package provides configuration options which can be used to
// customize the behavior of the components.
package core
