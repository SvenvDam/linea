package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestNewConfig validates that NewConfig correctly initializes a Config with the provided options
func TestNewConfig(t *testing.T) {
	tests := []struct {
		name         string
		minBackoff   time.Duration
		maxBackoff   time.Duration
		randomFactor float64
		opts         []Option
		expected     Config
	}{
		{
			name:         "default_config",
			minBackoff:   time.Second,
			maxBackoff:   time.Minute,
			randomFactor: 0.2,
			opts:         nil,
			expected: Config{
				minBackoff:   time.Second,
				maxBackoff:   time.Minute,
				randomFactor: 0.2,
				maxRetries:   nil,
			},
		},
		{
			name:         "with_max_retries",
			minBackoff:   500 * time.Millisecond,
			maxBackoff:   30 * time.Second,
			randomFactor: 0.1,
			opts:         []Option{WithMaxRetries(3)},
			expected: Config{
				minBackoff:   500 * time.Millisecond,
				maxBackoff:   30 * time.Second,
				randomFactor: 0.1,
				maxRetries:   ptrUint(3),
			},
		},
		{
			name:         "zero_max_retries",
			minBackoff:   time.Second,
			maxBackoff:   time.Minute,
			randomFactor: 0,
			opts:         []Option{WithMaxRetries(0)},
			expected: Config{
				minBackoff:   time.Second,
				maxBackoff:   time.Minute,
				randomFactor: 0,
				maxRetries:   ptrUint(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig(tt.minBackoff, tt.maxBackoff, tt.randomFactor, tt.opts...)

			assert.Equal(t, tt.expected.minBackoff, config.minBackoff)
			assert.Equal(t, tt.expected.maxBackoff, config.maxBackoff)
			assert.InDelta(t, tt.expected.randomFactor, config.randomFactor, 1e-9)

			if tt.expected.maxRetries == nil {
				assert.Nil(t, config.maxRetries)
			} else {
				assert.NotNil(t, config.maxRetries)
				assert.Equal(t, *tt.expected.maxRetries, *config.maxRetries)
			}
		})
	}
}

func TestConfig_NextBackoff(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		attempts    uint
		minExpected time.Duration
		maxExpected time.Duration // For tests with random factor
		expectOk    bool
	}{
		{
			name:        "initial_backoff",
			config:      NewConfig(time.Second, time.Minute, 0),
			attempts:    0,
			minExpected: time.Second,
			maxExpected: time.Second,
			expectOk:    true,
		},
		{
			name:        "exponential_backoff",
			config:      NewConfig(time.Second, time.Minute, 0),
			attempts:    2,
			minExpected: 4 * time.Second, // 1s * 2^2
			maxExpected: 4 * time.Second,
			expectOk:    true,
		},
		{
			name:        "capped_backoff",
			config:      NewConfig(time.Second, 5*time.Second, 0),
			attempts:    10, // Would be 1024s without cap
			minExpected: 5 * time.Second,
			maxExpected: 5 * time.Second,
			expectOk:    true,
		},
		{
			name:        "with_jitter",
			config:      NewConfig(time.Second, time.Minute, 0.5), // Up to 50% additional random delay
			attempts:    1,                                        // 2s base backoff
			minExpected: 2 * time.Second,
			maxExpected: 3 * time.Second, // 2s + 50% max jitter
			expectOk:    true,
		},
		{
			name:        "limited_retries",
			config:      NewConfig(time.Second, time.Minute, 0, WithMaxRetries(5)),
			attempts:    5, // Equal to max retries
			minExpected: 0,
			maxExpected: 0,
			expectOk:    false,
		},
		{
			name:        "unlimited_retries",
			config:      NewConfig(time.Second, time.Minute, 0), // Default is unlimited
			attempts:    100,                                    // Much higher than default
			minExpected: time.Minute,                            // Should be capped at max backoff
			maxExpected: time.Minute,
			expectOk:    true,
		},
		{
			name:        "edge_case_min_equals_max_backoff",
			config:      NewConfig(10*time.Second, 10*time.Second, 0),
			attempts:    3,
			minExpected: 10 * time.Second, // Should always be 10s regardless of attempts
			maxExpected: 10 * time.Second,
			expectOk:    true,
		},
		{
			name:        "high_random_factor",
			config:      NewConfig(time.Second, time.Minute, 1.0), // 100% jitter
			attempts:    1,                                        // 2s base backoff
			minExpected: 2 * time.Second,
			maxExpected: 4 * time.Second, // Up to double with 100% jitter
			expectOk:    true,
		},
		{
			name:        "zero_random_factor",
			config:      NewConfig(time.Second, time.Minute, 0),
			attempts:    3,
			minExpected: 8 * time.Second, // 1s * 2^3
			maxExpected: 8 * time.Second, // No jitter, so exact value
			expectOk:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backoff, ok := tt.config.NextBackoff(tt.attempts)
			assert.Equal(t, tt.expectOk, ok)
			if ok {
				assert.GreaterOrEqual(t, backoff, tt.minExpected)
				assert.LessOrEqual(t, backoff, tt.maxExpected)
			} else {
				assert.Equal(t, time.Duration(0), backoff)
			}
		})
	}
}

func TestWithMaxRetries(t *testing.T) {
	tests := []struct {
		name        string
		maxRetries  uint
		attempts    uint
		expectRetry bool
	}{
		{
			name:        "under_limit",
			maxRetries:  5,
			attempts:    4,
			expectRetry: true,
		},
		{
			name:        "at_limit",
			maxRetries:  5,
			attempts:    5,
			expectRetry: false,
		},
		{
			name:        "over_limit",
			maxRetries:  5,
			attempts:    6,
			expectRetry: false,
		},
		{
			name:        "zero_limit",
			maxRetries:  0,
			attempts:    0,
			expectRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewConfig(time.Second, time.Minute, 0, WithMaxRetries(tt.maxRetries))
			_, ok := config.NextBackoff(tt.attempts)
			assert.Equal(t, tt.expectRetry, ok)
		})
	}
}

// Helper function to create a pointer to a uint
func ptrUint(n uint) *uint {
	return &n
}
