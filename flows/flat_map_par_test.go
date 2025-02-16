package flows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestFlatMapPar(t *testing.T) {
	maxParallelism := 2

	tests := []struct {
		name        string
		input       []int
		setup       func() func(int) []string
		parallelism int
		want        []string
	}{
		{
			name:  "flattens and transforms items in parallel",
			input: []int{1, 2, 3, 4},
			setup: func() func(i int) []string {
				parTracker := test.NewParallelTracker()
				mapper := func(i int) []string {
					parallelism, cleanup := parTracker.Track()
					defer cleanup()

					assert.LessOrEqual(t, parallelism, maxParallelism)

					time.Sleep(50 * time.Millisecond) // simulate work
					if i%2 == 0 {
						return []string{
							"even" + string(rune(i+'0')),
							"even" + string(rune(i+'0')),
						}
					}
					return []string{"odd" + string(rune(i+'0'))}
				}
				return mapper
			},
			parallelism: maxParallelism,
			want: []string{
				"odd1",
				"even2", "even2",
				"odd3",
				"even4", "even4",
			},
		},
		{
			name:  "handles empty input",
			input: []int{},
			setup: func() func(i int) []string {
				parTracker := test.NewParallelTracker()
				mapper := func(i int) []string {
					parallelism, cleanup := parTracker.Track()
					assert.LessOrEqual(t, parallelism, maxParallelism)
					defer cleanup()
					t.Error("mapper should not be called for empty input")
					return nil
				}
				return mapper
			},
			parallelism: maxParallelism,
			want:        []string{},
		},
		{
			name:  "handles mapper returning empty slices",
			input: []int{1, 2, 3},
			setup: func() func(i int) []string {
				parTracker := test.NewParallelTracker()
				mapper := func(i int) []string {
					parallelism, cleanup := parTracker.Track()
					defer cleanup()

					assert.LessOrEqual(t, parallelism, maxParallelism)

					time.Sleep(50 * time.Millisecond) // simulate work
					if i == 2 {
						return []string{"middle"}
					}
					return []string{}
				}
				return mapper
			},
			parallelism: maxParallelism,
			want:        []string{"middle"},
		},
		{
			name:  "handles nil slices from mapper",
			input: []int{1, 2},
			setup: func() func(i int) []string {
				parTracker := test.NewParallelTracker()
				mapper := func(i int) []string {
					parallelism, cleanup := parTracker.Track()
					defer cleanup()

					assert.LessOrEqual(t, parallelism, maxParallelism)

					time.Sleep(50 * time.Millisecond) // simulate work
					if i == 1 {
						return nil
					}
					return []string{"valid"}
				}
				return mapper
			},
			parallelism: maxParallelism,
			want:        []string{"valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			mapper := tt.setup()

			before := make([]int, 0)
			after := make([]string, 0)

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CaptureItems(&before),
				FlatMapPar(mapper, tt.parallelism),
				test.CaptureItems(&after),
				sinks.Noop[string](),
			)

			res := <-stream.Run(ctx)
			assert.True(t, res.Ok)
			assert.Equal(t, tt.input, before)
			assert.ElementsMatch(t, tt.want, after) // Use ElementsMatch since order is not guaranteed
		})
	}
}
