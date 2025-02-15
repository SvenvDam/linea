// Package test provides testing utilities for stream processing components.
//
// This package contains helper functions and types for testing stream processing
// pipelines.
//
// These utilities are designed to work with Go's testing package and the
// stream processing components from other packages.
//
// Example:
//
//	seen := make([]int, 0)
//	flow := test.CaptureItems(&seen)
//	// or
//	flow := test.AssertEachItem(t, func(t *testing.T, item int) {
//	    assert.Greater(t, item, 0)
//	})
package test
