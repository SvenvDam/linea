package test

import (
	"sync"
	"testing"
)

type ParallelTracker struct {
	mu                 sync.Mutex
	currentParallelism int
	maxParallelism     int
}

func NewParallelTracker(maxParallelism int) *ParallelTracker {
	return &ParallelTracker{
		maxParallelism:     maxParallelism,
		currentParallelism: 0,
		mu:                 sync.Mutex{},
	}
}

func (p *ParallelTracker) Track(t *testing.T) (int, func()) {
	p.mu.Lock()
	p.currentParallelism++
	p.mu.Unlock()

	return p.currentParallelism, func() {
		p.mu.Lock()
		p.currentParallelism--
		p.mu.Unlock()
	}
}

func (p *ParallelTracker) Reset() {
	p.mu.Lock()
	p.currentParallelism = 0
	p.mu.Unlock()
}
