package core

import (
	"context"
	"sync"
)

// connectFlows combines two Flow components into a single Flow, where the output of flow1
// becomes the input to flow2. This allows for chaining data transformations.
//
// Type Parameters:
//   - I: Type of input data for the first flow
//   - O1: Type of output data from first flow (and input to second flow)
//   - O2: Type of output data from second flow
//
// Parameters:
//   - flow1: First Flow component that processes input type I to output type O1
//   - flow2: Second Flow component that processes O1 to produce O2
//
// Returns a new Flow that processes data from type I to type O2
func connectFlows[I, O1, O2 any](
	flow1 *Flow[I, O1],
	flow2 *Flow[O1, O2],
) *Flow[I, O2] {
	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	) {
		flow1.setup(ctx, cancel, wg, in)
		flow2.setup(ctx, cancel, wg, flow1.out)
	}

	return &Flow[I, O2]{
		out:   flow2.out,
		setup: setup,
	}
}

// appendFlowToSource attaches a Flow to a Source component, creating a new Source that
// outputs the processed data. This allows for transforming the data as it leaves the source.
//
// Type Parameters:
//   - I: Type of data produced by the original source
//   - O: Type of data after processing through the flow
//
// Parameters:
//   - source: Original Source component producing type I
//   - flow: Flow component that transforms I to O
//
// Returns a new Source that produces data of type O
func appendFlowToSource[I, O any](source *Source[I], flow *Flow[I, O]) *Source[O] {
	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		drain chan struct{},
	) {
		source.setup(ctx, cancel, wg, drain)
		flow.setup(ctx, cancel, wg, source.out)
	}

	return &Source[O]{
		out:   flow.out,
		setup: setup,
	}
}

// prependFlowToSink attaches a Flow to a Sink component, creating a new Sink that
// accepts the input type of the Flow. This allows for transforming data before
// it reaches the sink.
//
// Type Parameters:
//   - I: Type of input data to the flow
//   - O: Type of data after flow processing (and input to sink)
//   - R: Type of final result produced by the sink
//
// Parameters:
//   - flow: Flow component that transforms I to O
//   - sink: Sink component that processes O and produces result R
//
// Returns a new Sink that accepts type I and produces result R
func prependFlowToSink[I, O, R any](flow *Flow[I, O], sink *Sink[O, R]) *Sink[I, R] {
	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		in <-chan I,
	) {
		flow.setup(ctx, cancel, wg, in)
		sink.setup(ctx, cancel, wg, flow.out)
	}

	return &Sink[I, R]{
		out:   sink.out,
		setup: setup,
	}
}

// connectSourceToSink connects a Source directly to a Sink, creating a complete Stream
// that can be executed to produce a result. This is the final step in building a
// processing pipeline.
//
// Type Parameters:
//   - I: Type of data produced by the source and consumed by the sink
//   - R: Type of final result produced by the sink
//
// Parameters:
//   - source: Source component producing data of type I
//   - sink: Sink component consuming type I and producing result R
//
// Returns a Stream that can be executed to produce a result of type R
func connectSourceToSink[I, R any](source *Source[I], sink *Sink[I, R]) *Stream[R] {
	setup := func(
		ctx context.Context,
		cancel context.CancelFunc,
		wg *sync.WaitGroup,
		drain chan struct{},
	) {
		source.setup(ctx, cancel, wg, drain)
		sink.setup(ctx, cancel, wg, source.out)
	}

	return newStream(setup, sink.out)
}
