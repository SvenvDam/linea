package sources

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/sinks"
)

func TestChan(t *testing.T) {
	tests := []struct {
		name   string
		ch     chan int
		action func(ch chan int, stream *core.Stream[[]int])
		want   core.Item[[]int]
	}{
		{
			name: "emits all elements from channel",
			ch:   make(chan int, 3),
			action: func(ch chan int, stream *core.Stream[[]int]) {
				ch <- 1
				ch <- 2
				ch <- 3
				close(ch)
			},
			want: core.Item[[]int]{Value: []int{1, 2, 3}},
		},
		{
			name: "handles empty channel",
			ch:   make(chan int),
			action: func(ch chan int, stream *core.Stream[[]int]) {
				close(ch)
			},
			want: core.Item[[]int]{Value: []int{}},
		},
		{
			name: "cancels when context is cancelled",
			ch:   make(chan int, 3),
			action: func(ch chan int, stream *core.Stream[[]int]) {
				go func() {
					defer close(ch)
					ch <- 1
					ch <- 2
					stream.Cancel()
					time.Sleep(20 * time.Millisecond)
					ch <- 3
				}()
			},
			want: core.Item[[]int]{Err: context.Canceled},
		},
		{
			name: "gracefully stops when stream completes",
			ch:   make(chan int, 3),
			action: func(ch chan int, stream *core.Stream[[]int]) {
				go func() {
					defer close(ch)
					ch <- 1
					ch <- 2
					time.Sleep(20 * time.Millisecond) // give time for processing
					stream.Drain()
					time.Sleep(20 * time.Millisecond) // give time to shut down
					ch <- 3                           // should not be processed
				}()
			},
			want: core.Item[[]int]{Value: []int{1, 2}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ch := tt.ch

			stream := compose.SourceToSink(
				Chan(ch),
				sinks.Slice[int](),
			)

			resChan := stream.Run(ctx)

			tt.action(ch, stream)

			res := <-resChan
			assert.Equal(t, tt.want, res)
		})
	}
}
