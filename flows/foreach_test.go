package flows

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/test"
)

func TestForEach(t *testing.T) {
	tests := []struct {
		name   string
		input  []int
		want   []int
		effect func() (func(int), *sync.Map)
	}{
		{
			name:  "applies side effect and passes through",
			input: []int{1, 2, 3},
			want:  []int{1, 2, 3},
			effect: func() (func(int), *sync.Map) {
				seen := &sync.Map{}
				return func(i int) {
					seen.Store(i, true)
				}, seen
			},
		},
		{
			name:  "handles empty input",
			input: []int{},
			want:  []int{},
			effect: func() (func(int), *sync.Map) {
				return func(i int) {
					t.Error("should not apply effect to any items")
				}, nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			fn, seen := tt.effect()

			stream := compose.SourceThroughFlowToSink3(
				sources.Slice(tt.input),
				test.CheckItems(t, func(t *testing.T, elems []int) {
					assert.Equal(t, tt.input, elems)
				}),
				ForEach(fn),
				test.CheckItems(t, func(t *testing.T, elems []int) {
					assert.Equal(t, tt.want, elems)
				}),
				sinks.Noop[int](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
			for _, item := range tt.input {
				val, ok := seen.Load(item)
				assert.True(t, ok)
				assert.True(t, val.(bool))
			}
		})
	}
}
