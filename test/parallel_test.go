package test

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewParallelTracker verifies that NewParallelTracker creates a properly initialized tracker
func TestNewParallelTracker(t *testing.T) {
	tracker := NewParallelTracker()

	// Track once to verify initial parallelism is 1
	current, _ := tracker.Track()
	assert.Equal(t, 1, current, "Initial parallelism value should be 1")
}

// TestTrack verifies that Track increments and decrements counters correctly
func TestTrack(t *testing.T) {
	tests := []struct {
		name         string
		trackCount   int
		doneCount    int
		expectFinal  int
		expectPanics bool
	}{
		{
			name:         "single track and done",
			trackCount:   1,
			doneCount:    1,
			expectFinal:  0,
			expectPanics: false,
		},
		{
			name:         "multiple tracks with all done",
			trackCount:   3,
			doneCount:    3,
			expectFinal:  0,
			expectPanics: false,
		},
		{
			name:         "incomplete done calls",
			trackCount:   2,
			doneCount:    1,
			expectFinal:  1,
			expectPanics: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := NewParallelTracker()

			// Keep track of all done functions
			var doneFuncs []func()

			// Track the specified number of times
			for i := 0; i < tt.trackCount; i++ {
				current, done := tracker.Track()
				assert.Equal(t, i+1, current, "Current count should match expected value")
				doneFuncs = append(doneFuncs, done)
			}

			// Call done functions up to the specified limit
			for i := 0; i < tt.doneCount; i++ {
				doneFuncs[i]()
			}

			// Check the final parallelism value using a new track call
			if tt.trackCount > 0 {
				current, done := tracker.Track()
				defer done()
				assert.Equal(t, tt.expectFinal+1, current, "Final count should match expected value")
			}
		})
	}
}

// TestTrackMultiple verifies concurrent tracking behavior
func TestTrackMultiple(t *testing.T) {
	tracker := NewParallelTracker()

	// Track multiple goroutines concurrently
	const goroutineCount = 10
	var wg sync.WaitGroup

	// Access to shared storage needs to be synchronized
	var mutex sync.Mutex
	maxObserved := 0

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Track this goroutine
			current, done := tracker.Track()

			// Update max observed value
			mutex.Lock()
			if current > maxObserved {
				maxObserved = current
			}
			mutex.Unlock()

			// Sleep a bit to ensure overlap
			time.Sleep(10 * time.Millisecond)

			// Signal completion
			done()
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify maximum parallelism observed
	assert.LessOrEqual(t, goroutineCount, maxObserved, "Max parallelism should be at least goroutineCount")
}
