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
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[I], cancel context.CancelFunc, complete core.CompleteFunc) bool {
			check(t, elem)
			util.Send(ctx, core.Item[I]{Value: elem}, out)
			return true
		},
		nil,
		nil,
		opts...,
	)
}

func CheckItems[I any](
	t *testing.T,
	check func(t *testing.T, seen []I),
	opts ...core.FlowOption,
) *core.Flow[I, I] {
	seen := make([]I, 0)
	return core.NewFlow(
		func(ctx context.Context, elem I, out chan<- core.Item[I], cancel context.CancelFunc, complete core.CompleteFunc) bool {
			seen = append(seen, elem)
			util.Send(ctx, core.Item[I]{Value: elem}, out)
			return true
		},
		nil,
		func(ctx context.Context, out chan<- core.Item[I]) {
			check(t, seen)
		},
		opts...,
	)
}
