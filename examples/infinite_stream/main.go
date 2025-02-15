package main

import (
	"context"
	"fmt"
	"time"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func main() {
	ctx := context.Background()
	stream := core.SourceThroughFlowToSink(
		sources.Repeat(1),  // infinite stream of 1s
		flows.Map(func(i int) int { return i * 2 }), // multiply by 2
		sinks.ForEach(func(i int) { fmt.Println(i) }), // print the result
	)

	resChan := stream.Run(ctx)  // stream now runs, numbers will be printed

	time.Sleep(time.Second) // sleep for 1 second

	stream.Drain() // gracefully stop the stream

	<-resChan // wait for the stream to finish
}
