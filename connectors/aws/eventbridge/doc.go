// Package eventbridge provides components to interact with Amazon EventBridge.
//
// It currently offers:
// - SendFlow for publishing events to EventBridge while preserving the original input
//
// Features:
// - EventBridge event publishing with result handling and original input preservation
//
// This package requires an externally configured AWS client to be passed in, allowing the caller
// to handle authentication and AWS configuration according to their own requirements.
package eventbridge
