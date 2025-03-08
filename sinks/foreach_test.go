package sinks

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sources"
)

func TestForEach(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(context.Context) context.Context
		elements []int
		effect   func() (func(int), *sync.Map)
	}{
		{
			name:     "applies effect to all elements",
			setup:    func(ctx context.Context) context.Context { return ctx },
			elements: []int{1, 2, 3},
			effect: func() (func(int), *sync.Map) {
				seen := &sync.Map{}
				return func(i int) {
					seen.Store(i, true)
				}, seen
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctx = tt.setup(ctx)

			fn, seen := tt.effect()
			stream := compose.SourceToSink(
				sources.Slice(tt.elements),
				ForEach(fn),
			)

			result := <-stream.Run(ctx)
			assert.NoError(t, result.Err)

			for _, elem := range tt.elements {
				val, ok := seen.Load(elem)
				assert.True(t, ok)
				assert.True(t, val.(bool))
			}
		})
	}
}
