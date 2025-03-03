package flows

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestMapPar(t *testing.T) {
	maxParallelism := 2

	tests := []struct {
		name        string
		input       []int
		setup       func() func(int) string
		parallelism int
		want        []string
	}{
		{
			name: "maps integers to strings in parallel",
			input: func() []int {
				items := make([]int, 25)
				for i := range items {
					items[i] = i + 1
				}
				return items
			}(),
			setup: func() func(i int) string {
				parTracker := test.NewParallelTracker()
				mapper := func(i int) string {
					parallelism, cleanup := parTracker.Track()
					defer cleanup()

					assert.LessOrEqual(t, parallelism, maxParallelism)

					time.Sleep(50 * time.Millisecond) // simulate work
					return strconv.Itoa(i)
				}

				return mapper
			},
			parallelism: maxParallelism,
			want: func() []string {
				items := make([]string, 25)
				for i := range items {
					items[i] = strconv.Itoa(i + 1)
				}
				return items
			}(),
		},
		{
			name:  "handles errors",
			input: []int{1, 2, 3},
			setup: func() func(i int) string {
				parTracker := test.NewParallelTracker()
				mapper := func(i int) string {
					parallelism, cleanup := parTracker.Track()
					defer cleanup()

					assert.LessOrEqual(t, parallelism, maxParallelism)

					time.Sleep(50 * time.Millisecond) // simulate work
					return strconv.Itoa(i)
				}
				return mapper
			},
			parallelism: maxParallelism,
			want:        []string{"1", "2", "3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mapper := tt.setup()

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CheckItems(t, func(t *testing.T, seen []int) {
					assert.Equal(t, tt.input, seen)
				}),
				MapPar(mapper, tt.parallelism),
				test.CheckItems(t, func(t *testing.T, seen []string) {
					assert.ElementsMatch(t, tt.want, seen)
				}),
				sinks.Noop[string](),
			)

			res := <-stream.Run(ctx)
			assert.True(t, res.Ok)
		})
	}
}
