// Package util provides internal utility functions for stream processing operations.
//
// This package contains helper functions used across other packages to implement
// common stream processing patterns. Key functionalities include:
//
// Note: This package is primarily intended for internal use by other linea packages.
// While its functions are exported, only use this if you know what you are doing.
// External code should prefer using the high-level components from the core,
// flows, sources, and sinks packages.
//
// Example internal usage:
//
//	util.ProcessLoop(ctx, in, out, func(item T) {
//	    // Process single item
//	    util.Send(ctx, processedItem, out)
//	}, func() {
//	    // Cleanup after all items processed
//	})
package util
