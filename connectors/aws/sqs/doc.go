// Package sqs provides components to interact with Amazon SQS.
//
// It currently offers a Source for reading messages from SQS queues.
//
// Features:
// - SQS message reading with configurable batching and polling
//
// This package requires an externally configured AWS client to be passed in, allowing the caller
// to handle authentication and AWS configuration according to their own requirements.
package sqs
