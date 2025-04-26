package compose

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestComposeVariants(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *core.Stream[[]int]
		expected []int
	}{
		{
			name: "SourceThroughFlow",
			setup: func() *core.Stream[[]int] {
				source := SourceThroughFlow(
					sources.Slice([]int{1}),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
				)
				return SourceToSink(source, sinks.Slice[int]())
			},
			expected: []int{2},
		},
		{
			name: "SourceThroughFlow2",
			setup: func() *core.Stream[[]int] {
				source := SourceThroughFlow2(
					sources.Slice([]int{1}),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					flows.Map(func(_ context.Context, i int) int { return i + 1 }),
				)
				return SourceToSink(source, sinks.Slice[int]())
			},
			expected: []int{3},
		},
		{
			name: "SourceThroughFlow3",
			setup: func() *core.Stream[[]int] {
				source := SourceThroughFlow3(
					sources.Slice([]int{1}),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					flows.Map(func(_ context.Context, i int) int { return i + 1 }),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
				)
				return SourceToSink(source, sinks.Slice[int]())
			},
			expected: []int{6},
		},
		{
			name: "SourceThroughFlowToSink",
			setup: func() *core.Stream[[]int] {
				return SourceThroughFlowToSink(
					sources.Slice([]int{1}),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					sinks.Slice[int](),
				)
			},
			expected: []int{2},
		},
		{
			name: "SourceThroughFlowToSink2",
			setup: func() *core.Stream[[]int] {
				return SourceThroughFlowToSink2(
					sources.Slice([]int{1}),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					flows.Map(func(_ context.Context, i int) int { return i + 1 }),
					sinks.Slice[int](),
				)
			},
			expected: []int{3},
		},
		{
			name: "SourceThroughFlowToSink3",
			setup: func() *core.Stream[[]int] {
				return SourceThroughFlowToSink3(
					sources.Slice([]int{1}),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					flows.Map(func(_ context.Context, i int) int { return i + 1 }),
					flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					sinks.Slice[int](),
				)
			},
			expected: []int{6},
		},
		{
			name: "SinkThroughFlow",
			setup: func() *core.Stream[[]int] {
				return SourceToSink(
					sources.Slice([]int{1}),
					SinkThroughFlow(
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
						sinks.Slice[int](),
					),
				)
			},
			expected: []int{2},
		},
		{
			name: "SinkThroughFlow2",
			setup: func() *core.Stream[[]int] {
				return SourceToSink(
					sources.Slice([]int{1}),
					SinkThroughFlow2(
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
						flows.Map(func(_ context.Context, i int) int { return i + 1 }),
						sinks.Slice[int](),
					),
				)
			},
			expected: []int{3},
		},
		{
			name: "SinkThroughFlow3",
			setup: func() *core.Stream[[]int] {
				return SourceToSink(
					sources.Slice([]int{1}),
					SinkThroughFlow3(
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
						flows.Map(func(_ context.Context, i int) int { return i + 1 }),
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
						sinks.Slice[int](),
					),
				)
			},
			expected: []int{6},
		},
		{
			name: "MergeFlows",
			setup: func() *core.Stream[[]int] {
				return SourceThroughFlowToSink(
					sources.Slice([]int{1}),
					MergeFlows(
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
						flows.Map(func(_ context.Context, i int) int { return i + 1 }),
					),
					sinks.Slice[int](),
				)
			},
			expected: []int{3},
		},
		{
			name: "MergeFlows3",
			setup: func() *core.Stream[[]int] {
				return SourceThroughFlowToSink(
					sources.Slice([]int{1}),
					MergeFlows3(
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
						flows.Map(func(_ context.Context, i int) int { return i + 1 }),
						flows.Map(func(_ context.Context, i int) int { return i * 2 }),
					),
					sinks.Slice[int](),
				)
			},
			expected: []int{6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := tt.setup()
			res := <-stream.Run(context.Background())
			assert.NoError(t, res.Err)
			assert.Equal(t, tt.expected, res.Value)
			stream.Cancel()
		})
	}
}
