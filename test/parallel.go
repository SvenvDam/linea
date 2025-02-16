package test

import (
	"sync/atomic"
)

type ParallelTracker struct {
	currentParallelism atomic.Int32
}

func NewParallelTracker() *ParallelTracker {
	return &ParallelTracker{}
}

func (p *ParallelTracker) Track() (int, func()) {
	current := p.currentParallelism.Add(1)

	return int(current), func() {
		p.currentParallelism.Add(-1)
	}
}
