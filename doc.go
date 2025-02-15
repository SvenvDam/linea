// Package linea provides a composable streaming data processing library for Go.
//
// Linea allows you to build data processing pipelines by connecting three main components:
//   - Sources: Produce data items (e.g., from slices, channels)
//   - Flows: Transform data items (e.g., map, filter, batch)
//   - Sinks: Consume and reduce data items to final results
//
// Key features:
//   - Type-safe: Uses Go generics for compile-time type checking
//   - Composable: Components can be chained together flexibly
//   - Concurrent: Built on Go channels and goroutines
//   - Cancellable: Supports context-based cancellation
//   - Parallel: Offers parallel processing flows (MapPar, FlatMapPar)
//
// Example usage:
//
//	stream := core.SourceThroughFlowToSink2(
//	    sources.Slice([]int{1,2,3,4,5}),
//	    flows.Filter(func(i int) bool { return i%2 == 0 }),
//	    flows.Map(func(i int) string { return strconv.Itoa(i) }),
//	    sinks.Slice[string](),
//	)
//
//	result := <-stream.Run(context.Background())
//	// result.Value contains []string{"2", "4"}
//
// The library provides a rich set of built-in components while allowing custom
// implementations through the core interfaces.
package linea
