Linea is a composable stream processing library for Go.

It allows you to build data processing pipelines by connecting three main components:
- Sources: Produce data items (e.g., from slices, channels)
- Flows: Transform data items (e.g., map, filter, batch)
- Sinks: Consume and reduce data items to final results

Key features:
- Type-safe: Uses Go generics for compile-time type checking
- Composable: Components can be chained together flexibly
- Concurrent: Built on Go channels and goroutines
- Cancellable: Supports context-based cancellation
- Parallel: Offers parallel processing flows (MapPar, FlatMapPar)

# Quickstart
```go
package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func main() {
	ctx := context.Background()
	stream := core.SourceThroughFlowToSink2(
		sources.Slice([]int{1, 2, 3, 4, 5}), // source from slice
		flows.Filter(func(i int) bool { return i%2 == 0 }), // filter even numbers
		flows.Map(func(i int) string { return strconv.Itoa(i) }), // map to string
		sinks.Slice[string](), // sink to slice
	)

	result := <-stream.Run(ctx) // run the stream, blocks until the stream is done

	fmt.Println(result.Value) // prints [2 4]
}
```

# Installation
```shell
go get github.com/svenvdam/linea
```

# Creating Streams

Linea provides several ways to create and compose streams. The recommended approach is to use the pre-built components in the `sources`, `flows`, and `sinks` packages:

```go
// Create a stream using SourceThroughFlowToSink helpers
stream1 := core.SourceThroughFlowToSink(
    sources.Slice([]int{1, 2, 3}),    // Source
    flows.Map(func(i int) int { return i * 2 }), // Flow
    sinks.Slice[int](),               // Sink
)

// Chain multiple flows using SourceThroughFlowToSink2, 3, etc.
stream2 := core.SourceThroughFlowToSink2(
    sources.Slice([]string{"a", "b"}),
    flows.Map(strings.ToUpper),
    flows.Filter(func(s string) bool { return s != "B" }),
    sinks.Slice[string](),
)
```

The `core` package provides more advanced functionality for creating custom components and composing streams manually. However, for most use cases, the pre-built components should be sufficient and are the recommended approach.

# Stream Lifecycle Management

Streams in Linea follow a simple lifecycle model that helps manage resources and control execution:

1. **Creation**: When you create a stream using any of the composition helpers (like `SourceThroughFlowToSink`), the stream is initialized but not yet running.

2. **Execution**: Streams start processing data when you call `Run(ctx)`. This method:
   - Starts all internal goroutines
   - Begins data flow from source through flows to sink
   - Returns a channel that will receive the final result

3. **Termination**: Streams can be terminated in several ways:
   - Natural completion: when the source is exhausted
   - Context cancellation: `cancel()`
   - Immediate shutdown: `stream.Cancel()`.
   - Graceful shutdown: `stream.Drain()`

4. **Cleanup**: When a stream terminates:
   - All internal goroutines are properly terminated
   - Channels are closed in the correct order
   - Resources are released

## Shutdown Options

Linea provides multiple ways to stop stream processing, and you can determine how the stream terminated by checking the Result:

```go
func processWithShutdown(data []int) {
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    stream := core.SourceThroughFlowToSink2(
        sources.Slice(data),
        flows.Map(slowOperation),
        sinks.Slice[int](),
    )

    // Option 1: Immediate shutdown - stops processing immediately
    go func() {
        time.Sleep(1 * time.Second)
        stream.Cancel() // Cancels immediately, may leave items unprocessed. Blocks until all resources have been cleaned up.
    }()

    // Option 2: Graceful shutdown - processes remaining items
    go func() {
        time.Sleep(1 * time.Second)
        stream.Drain() // Stops accepting new items but processes existing ones
    }()

    // Option 3: Context cancellation - similar to Cancel()
    go func() {
        time.Sleep(1 * time.Second)
        cancel() // Cancels via context
    }()

    result := <-stream.Run(ctx)

    if !result.Ok {
        // Stream was cancelled or encountered an error
        // Items may not have been fully processed
    } else {
        // Stream completed successfully
        // All items were processed
        processedItems := result.Value
    }
}
```

**Shutdown Methods:**
- `stream.Cancel()`: Immediate shutdown that stops all processing. Results in `Ok: false`. This method blocks until all internal goroutines have been terminated, ensuring proper resource cleanup.
- `stream.Drain()`: Graceful shutdown that processes remaining items. Results in `Ok: true` if all items were processed.
- Context cancellation: Cancels via context, similar to `Cancel()`. Results in `Ok: false`.

**Understanding Results:**
- `result.Ok == true`: Indicates the stream completed successfully, processing all items
- `result.Ok == false`: Indicates the stream was cancelled before completion
- `result.Value`: Contains the processed data when `Ok` is true

**Best Practices:**
- Use `defer cancel()` when creating cancellable contexts
- Check `result.Ok` to determine if all items were processed
- Use `Drain()` when you want to ensure all items in the pipeline are processed
- Use `Cancel()` when immediate shutdown is required
