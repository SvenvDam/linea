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
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- I, cancel context.CancelFunc) bool {
		check(t, elem)
		util.Send(ctx, elem, out)
		return true
	}, func(ctx context.Context, out chan<- I) {}, opts...)
}

func CaptureItems[I any](
	elems *[]I,
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	return core.NewFlow(func(ctx context.Context, elem I, out chan<- I, cancel context.CancelFunc) bool {
		*elems = append(*elems, elem)
		util.Send(ctx, elem, out)
		return true
	}, func(ctx context.Context, out chan<- I) {}, opts...)
}
