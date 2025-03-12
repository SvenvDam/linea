package compose

import "github.com/svenvdam/linea/core"

// SourceThroughFlow creates a new source with the flow transformation applied.
// This is the basic building block for creating processing pipelines, allowing
// a single transformation to be applied to a source's output.
//
// Type Parameters:
//   - I: Type of items produced by the source
//   - O: Type of items after flow transformation
//
// Parameters:
//   - source: The original source producing items of type I
//   - flow: The flow that transforms items from type I to type O
//
// Returns a new Source that produces items of type O
func SourceThroughFlow[I, O any](source *core.Source[I], flow *core.Flow[I, O]) *core.Source[O] {
	return core.AppendFlowToSource(source, flow)
}

// SourceToSink creates a runnable stream from a source and a sink.
// This is the final step in building a processing pipeline, connecting
// a source directly to a sink to create an executable stream.
//
// Type Parameters:
//   - I: Type of items produced by the source and consumed by the sink
//   - R: Type of the final result produced by the sink
//
// Parameters:
//   - source: The source producing items of type I
//   - sink: The sink consuming items of type I and producing a result of type R
//
// Returns a Stream that can be executed to produce a result of type R
func SourceToSink[I, R any](source *core.Source[I], sink *core.Sink[I, R]) *core.Stream[R] {
	return core.ConnectSourceToSink(source, sink)
}

// Convenience functions for creating chains

// SourceThroughFlow2 creates a new source by applying two flows in sequence.
// This is a convenience function that chains two transformations together.
//
// Type Parameters:
//   - I: Type of items produced by the source
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow
//
// Parameters:
//   - source: The original source producing items of type I
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//
// Returns a new Source that produces items of type O2
func SourceThroughFlow2[I, O1, O2 any](
	source *core.Source[I],
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
) *core.Source[O2] {
	s1 := core.AppendFlowToSource(source, flow1)
	return core.AppendFlowToSource(s1, flow2)
}

// SourceThroughFlow3 creates a new source by applying three flows in sequence.
// This is a convenience function that chains three transformations together.
//
// Type Parameters:
//   - I: Type of items produced by the source
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow
//   - O3: Type of items after third flow
//
// Parameters:
//   - source: The original source producing items of type I
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//   - flow3: Third flow transforming O2 to O3
//
// Returns a new Source that produces items of type O3
func SourceThroughFlow3[I, O1, O2, O3 any](
	source *core.Source[I],
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
	flow3 *core.Flow[O2, O3],
) *core.Source[O3] {
	s1 := core.AppendFlowToSource(source, flow1)
	s2 := core.AppendFlowToSource(s1, flow2)
	return core.AppendFlowToSource(s2, flow3)
}

// SourceThroughFlowToSink connects a source to a sink through a flow and returns
// a stream. This is a convenience function that chains a single transformation
// between a source and sink.
//
// Type Parameters:
//   - I: Type of items produced by the source
//   - O: Type of items after the flow transformation
//   - R: Type of result produced by the sink
//
// Parameters:
//   - source: The original source producing items of type I
//   - flow: Flow transforming I to O
//   - sink: Sink consuming items of type O and producing result R
//
// Returns a Stream that will produce a result of type R when run
func SourceThroughFlowToSink[I, O, R any](
	source *core.Source[I],
	flow *core.Flow[I, O],
	sink *core.Sink[O, R],
) *core.Stream[R] {
	s := core.AppendFlowToSource(source, flow)
	return core.ConnectSourceToSink(s, sink)
}

// SourceThroughFlowToSink2 creates a runnable stream by connecting a source to a sink
// through two flows. This is a convenience function for creating a stream with two
// transformation stages.
//
// Type Parameters:
//   - I: Type of items produced by the source
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow (consumed by sink)
//   - R: Type of the final result produced by the sink
//
// Parameters:
//   - source: The original source producing items of type I
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//   - sink: The sink consuming O2 and producing result R
//
// Returns a Stream that can be executed to produce a result of type R
func SourceThroughFlowToSink2[I, O1, O2, R any](
	source *core.Source[I],
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
	sink *core.Sink[O2, R],
) *core.Stream[R] {
	s1 := core.AppendFlowToSource(source, flow1)
	s2 := core.AppendFlowToSource(s1, flow2)
	return core.ConnectSourceToSink(s2, sink)
}

// SourceThroughFlowToSink3 creates a runnable stream by connecting a source to a sink
// through three flows. This is a convenience function for creating a stream with three
// transformation stages.
//
// Type Parameters:
//   - I: Type of items produced by the source
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow
//   - O3: Type of items after third flow (consumed by sink)
//   - R: Type of the final result produced by the sink
//
// Parameters:
//   - source: The original source producing items of type I
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//   - flow3: Third flow transforming O2 to O3
//   - sink: The sink consuming O3 and producing result R
//
// Returns a Stream that can be executed to produce a result of type R
func SourceThroughFlowToSink3[I, O1, O2, O3, R any](
	source *core.Source[I],
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
	flow3 *core.Flow[O2, O3],
	sink *core.Sink[O3, R],
) *core.Stream[R] {
	s1 := core.AppendFlowToSource(source, flow1)
	s2 := core.AppendFlowToSource(s1, flow2)
	s3 := core.AppendFlowToSource(s2, flow3)
	return core.ConnectSourceToSink(s3, sink)
}

// SinkThroughFlow creates a new sink by prepending a flow transformation.
// This allows for transforming data before it reaches the sink.
//
// Type Parameters:
//   - I: Type of input items to the flow
//   - O: Type of items after flow transformation (consumed by sink)
//   - R: Type of result produced by the sink
//
// Parameters:
//   - flow: The flow that transforms items from type I to O
//   - sink: The sink consuming items of type O and producing result R
//
// Returns a new Sink that accepts items of type I and produces result R
func SinkThroughFlow[I, O, R any](flow *core.Flow[I, O], sink *core.Sink[O, R]) *core.Sink[I, R] {
	return core.PrependFlowToSink(flow, sink)
}

// SinkThroughFlow2 creates a new sink by prepending two flows in sequence.
// This is a convenience function that chains two transformations before a sink.
//
// Type Parameters:
//   - I: Type of input items to first flow
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow (consumed by sink)
//   - R: Type of result produced by the sink
//
// Parameters:
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//   - sink: The sink consuming O2 and producing result R
//
// Returns a new Sink that accepts items of type I and produces result R
func SinkThroughFlow2[I, O1, O2, R any](
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
	sink *core.Sink[O2, R],
) *core.Sink[I, R] {
	merged := core.ConnectFlows(flow1, flow2)
	return core.PrependFlowToSink(merged, sink)
}

// SinkThroughFlow3 creates a new sink by prepending three flows in sequence.
// This is a convenience function that chains three transformations before a sink.
//
// Type Parameters:
//   - I: Type of input items to first flow
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow
//   - O3: Type of items after third flow (consumed by sink)
//   - R: Type of result produced by the sink
//
// Parameters:
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//   - flow3: Third flow transforming O2 to O3
//   - sink: The sink consuming O3 and producing result R
//
// Returns a new Sink that accepts items of type I and produces result R
func SinkThroughFlow3[I, O1, O2, O3, R any](
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
	flow3 *core.Flow[O2, O3],
	sink *core.Sink[O3, R],
) *core.Sink[I, R] {
	f1 := core.ConnectFlows(flow1, flow2)
	f2 := core.ConnectFlows(f1, flow3)
	return core.PrependFlowToSink(f2, sink)
}

// MergeFlows creates a new flow by combining two flows in sequence.
// This allows for creating more complex transformations by combining simpler ones.
//
// Type Parameters:
//   - I: Type of input items to first flow
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow
//
// Parameters:
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//
// Returns a new Flow that transforms items from type I to O2
func MergeFlows[I, O1, O2 any](flow1 *core.Flow[I, O1], flow2 *core.Flow[O1, O2]) *core.Flow[I, O2] {
	return core.ConnectFlows(flow1, flow2)
}

// MergeFlows3 creates a new flow by combining three flows in sequence.
// This is a convenience function that chains three transformations together.
//
// Type Parameters:
//   - I: Type of input items to first flow
//   - O1: Type of items after first flow
//   - O2: Type of items after second flow
//   - O3: Type of items after third flow
//
// Parameters:
//   - flow1: First flow transforming I to O1
//   - flow2: Second flow transforming O1 to O2
//   - flow3: Third flow transforming O2 to O3
//
// Returns a new Flow that transforms items from type I to O3
func MergeFlows3[I, O1, O2, O3 any](
	flow1 *core.Flow[I, O1],
	flow2 *core.Flow[O1, O2],
	flow3 *core.Flow[O2, O3],
) *core.Flow[I, O3] {
	f1 := core.ConnectFlows(flow1, flow2)
	return core.ConnectFlows(f1, flow3)
}
