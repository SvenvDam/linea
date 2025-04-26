package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func main() {
	ctx := context.Background()
	stream := compose.SourceThroughFlowToSink2(
		sources.Slice([]int{1, 2, 3, 4, 5}),                                         // source from slice
		flows.Filter(func(_ context.Context, i int) bool { return i%2 == 0 }),       // filter even numbers
		flows.Map(func(_ context.Context, i int) string { return strconv.Itoa(i) }), // map to string
		sinks.Slice[string](), // sink to slice
	)

	result := <-stream.Run(ctx) // run the stream, blocks until the stream is done

	fmt.Println(result.Value) // prints [2 4]
}
