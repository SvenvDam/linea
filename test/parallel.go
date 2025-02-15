package test

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
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

func (p *ParallelTracker) Track(t *testing.T) func() {
	p.mu.Lock()
	p.currentParallelism++
	assert.LessOrEqual(t, p.currentParallelism, p.maxParallelism)
	p.mu.Unlock()

	return func() {
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
