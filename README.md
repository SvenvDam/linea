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