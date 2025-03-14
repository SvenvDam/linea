// Package restart provides configuration and utilities for implementing
// restart strategies in streaming applications.
//
// The package is designed to be used with different stream components (such as flows and sinks)
// to provide consistent restart behavior across the application.
//
// # Backoff Strategy
//
// The Config struct implements an exponential backoff strategy with the following features:
//   - Configurable minimum and maximum backoff durations
//   - Exponential increase in delay between retries (base * 2^attempts)
//   - Optional random jitter to prevent synchronized retries
//   - Configurable maximum number of restart attempts
//
// # Benefits of Exponential Backoff with Jitter
//
// This approach offers several advantages for resilient systems:
//   - Exponential backoff reduces pressure on potentially failing resources by
//     increasing the wait time between retries
//   - Random jitter helps avoid thundering herd problems in distributed systems where
//     multiple clients might otherwise retry simultaneously
//   - Configurable maximum backoff prevents unreasonably long wait times
//   - Optional maximum retries helps prevent infinite retry loops
//
// # Usage Example
//
//	import (
//	    "errors"
//	    "time"
//
//	    "github.com/svenvdam/linea/restart"
//	)
//
//	// Create a restart config with 500ms initial backoff, 1 minute max backoff,
//	// 20% random jitter, and limited to 10 restart attempts
//	config := restart.NewConfig(
//	    500 * time.Millisecond,
//	    time.Minute,
//	    0.2,
//	    restart.WithMaxRestarts(10),
//	)
//
//	// Calculate backoff delay for the 3rd attempt (index 2)
//	delay, canRetry := config.NextBackoff(2)
//	if !canRetry {
//	    // Maximum number of retries reached
//	    return errors.New("maximum retries exceeded")
//	}
//
//	// Wait for the calculated backoff duration
//	time.Sleep(delay)
//
// The Config struct is inspired by Akka Streams' RestartSettings and provides
// sophisticated restart behavior including exponential backoff with jitter
// and configurable maximum restart counts.
package restart
