package test

import (
	"context"
	"testing"

	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/util"
)

func AssertEachItem[I any](
	t *testing.T,
	check func(t *testing.T, elem I),
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, in <-chan I, out chan<- I, cancel context.CancelFunc) {
		util.ProcessLoop(ctx, in, out, func(elem I) {
			check(t, elem)
			util.Send(ctx, elem, out)
		}, func() {})
	})
}

func CaptureItems[I any](
	elems *[]I,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, in <-chan I, out chan<- I, cancel context.CancelFunc) {
		util.ProcessLoop(ctx, in, out, func(elem I) {
			*elems = append(*elems, elem)
			util.Send(ctx, elem, out)
		}, func() {})
	})
}
