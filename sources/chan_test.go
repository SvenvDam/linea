package sources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
)

func TestChan(t *testing.T) {
	tests := []struct {
		name     string
		elements []int
		want     []int
	}{
		{
			name:     "emits all elements from channel",
			elements: []int{1, 2, 3},
			want:     []int{1, 2, 3},
		},
		{
			name:     "handles empty channel",
			elements: []int{},
			want:     []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			ch := make(chan int)
			go func() {
				for _, elem := range tt.elements {
					ch <- elem
				}
				close(ch)
			}()

			stream := compose.SourceToSink(
				Chan(ch),
				sinks.Slice[int](),
			)

			res := <-stream.Run(ctx)
			assert.NoError(t, res.Err)
			assert.Equal(t, tt.want, res.Value)
		})
	}
}
