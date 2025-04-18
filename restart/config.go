package restart

import (
	"math"
	"math/rand"
	"time"
)

// Config defines how a stream should restart on failure.
// It provides exponential backoff with optional jitter and configurable retry limits.
type Config struct {
	// minBackoff is the minimum (initial) duration until the stream will be restarted after failure
	minBackoff time.Duration

	// maxBackoff is the maximum duration that the exponential backoff is capped to
	maxBackoff time.Duration

	// randomFactor adds additional random delay as a percentage of the calculated backoff
	// For example, 0.2 adds up to 20% additional random delay
	// Set to 0 to disable random factor
	randomFactor float64

	// maxRestarts is the maximum number of restarts allowed
	// A value of nil means unlimited restarts
	maxRestarts *uint
}

// Option is a function that configures a Config
type Option func(*Config)

// WithMaxRestarts sets the maximum number of restart attempts.
// Once this limit is reached, NextBackoff will return (0, false).
// This is useful for preventing infinite restart loops in case of persistent failures.
func WithMaxRestarts(n uint) Option {
	return func(c *Config) {
		c.maxRestarts = &n
	}
}

// NewConfig creates a new Config with the specified options.
//
// Parameters:
//   - minBackoff: The initial backoff duration after the first failure
//   - maxBackoff: The maximum backoff duration that will not be exceeded
//   - randomFactor: A factor between 0.0 and 1.0 to add randomness to backoff duration
//   - opts: Optional configuration options like WithMaxRestarts
//
// Example:
//
//	// Config with 1s initial backoff, 1m max backoff, 20% jitter, and max 5 restarts
//	config := NewConfig(time.Second, time.Minute, 0.2, WithMaxRestarts(5))
func NewConfig(minBackoff time.Duration, maxBackoff time.Duration, randomFactor float64, opts ...Option) *Config {
	// Set default values
	c := &Config{
		minBackoff:   minBackoff,
		maxBackoff:   maxBackoff,
		randomFactor: randomFactor,
		maxRestarts:  nil,
	}

	// Apply all options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// NextBackoff calculates the next backoff duration based on the number of attempts
// using exponential backoff with jitter.
//
// The formula used is: min(maxBackoff, minBackoff * 2^attempts) + random jitter
// where jitter is a random value between 0 and (backoff * randomFactor).
//
// Parameters:
//   - attempts: The number of restart attempts that have already occurred (0-based)
//
// Returns:
//   - time.Duration: The calculated backoff duration
//   - bool: false if max restarts has been reached, true otherwise
func (c *Config) NextBackoff(attempts uint) (time.Duration, bool) {
	if c.maxRestarts != nil && attempts >= *c.maxRestarts {
		return 0, false
	}

	// Calculate exponential backoff: minBackoff * 2^attempts, capped at maxBackoff
	backoff := math.Min(
		float64(c.maxBackoff),
		float64(c.minBackoff)*math.Pow(2, float64(attempts)),
	)

	// Add random jitter if RandomFactor > 0
	if c.randomFactor > 0 {
		//nolint:gosec // G404: using math/rand for non-security jitter is intentional
		jitter := backoff * c.randomFactor * rand.Float64()
		backoff += jitter
	}

	return time.Duration(backoff), true
}
