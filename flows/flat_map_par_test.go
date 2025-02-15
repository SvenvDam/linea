package flows

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestFlatMapPar(t *testing.T) {
	maxParallelism := 2
	parTracker := test.NewParallelTracker(maxParallelism)

	tests := []struct {
		name        string
		input       []int
		mapper      func(int) []string
		parallelism int
		want        []string
	}{
		{
			name:  "flattens and transforms items in parallel",
			input: []int{1, 2, 3, 4},
			mapper: func(i int) []string {
				defer parTracker.Track(t)()
				time.Sleep(50 * time.Millisecond) // simulate work
				if i%2 == 0 {
					return []string{
						"even" + string(rune(i+'0')),
						"even" + string(rune(i+'0')),
					}
				}
				return []string{"odd" + string(rune(i+'0'))}
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
			mapper: func(i int) []string {
				defer parTracker.Track(t)()
				t.Error("mapper should not be called for empty input")
				return nil
			},
			parallelism: maxParallelism,
			want:        []string{},
		},
		{
			name:  "handles mapper returning empty slices",
			input: []int{1, 2, 3},
			mapper: func(i int) []string {
				defer parTracker.Track(t)()
				time.Sleep(50 * time.Millisecond) // simulate work
				if i == 2 {
					return []string{"middle"}
				}
				return []string{}
			},
			parallelism: maxParallelism,
			want:        []string{"middle"},
		},
		{
			name:  "handles nil slices from mapper",
			input: []int{1, 2},
			mapper: func(i int) []string {
				defer parTracker.Track(t)()
				time.Sleep(50 * time.Millisecond) // simulate work
				if i == 1 {
					return nil
				}
				return []string{"valid"}
			},
			parallelism: maxParallelism,
			want:        []string{"valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			parTracker.Reset()

			before := make([]int, 0)
			after := make([]string, 0)

			stream := core.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CaptureItems(&before),
				FlatMapPar(tt.mapper, tt.parallelism),
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
