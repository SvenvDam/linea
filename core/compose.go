package core

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
func SourceThroughFlow[I, O any](source *Source[I], flow *Flow[I, O]) *Source[O] {
	return appendFlowToSource(source, flow)
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
func SourceToSink[I, R any](source *Source[I], sink *Sink[I, R]) *Stream[R] {
	return connectSourceToSink(source, sink)
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
	source *Source[I],
	flow1 *Flow[I, O1],
	flow2 *Flow[O1, O2],
) *Source[O2] {
	s1 := appendFlowToSource(source, flow1)
	return appendFlowToSource(s1, flow2)
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
	source *Source[I],
	flow1 *Flow[I, O1],
	flow2 *Flow[O1, O2],
	flow3 *Flow[O2, O3],
) *Source[O3] {
	s1 := appendFlowToSource(source, flow1)
	s2 := appendFlowToSource(s1, flow2)
	return appendFlowToSource(s2, flow3)
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
func SourceThroughFlowToSink[I, O, R any](source *Source[I], flow *Flow[I, O], sink *Sink[O, R]) *Stream[R] {
	s := appendFlowToSource(source, flow)
	return connectSourceToSink(s, sink)
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
	source *Source[I],
	flow1 *Flow[I, O1],
	flow2 *Flow[O1, O2],
	sink *Sink[O2, R],
) *Stream[R] {
	s1 := appendFlowToSource(source, flow1)
	s2 := appendFlowToSource(s1, flow2)
	return connectSourceToSink(s2, sink)
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
	source *Source[I],
	flow1 *Flow[I, O1],
	flow2 *Flow[O1, O2],
	flow3 *Flow[O2, O3],
	sink *Sink[O3, R],
) *Stream[R] {
	s1 := appendFlowToSource(source, flow1)
	s2 := appendFlowToSource(s1, flow2)
	s3 := appendFlowToSource(s2, flow3)
	return connectSourceToSink(s3, sink)
}
