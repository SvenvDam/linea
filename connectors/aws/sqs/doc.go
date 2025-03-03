// Package sqs provides components to interact with Amazon SQS.
//
// It currently offers:
// - Source for reading messages from SQS queues
// - SendFlow for sending messages to SQS queues while preserving the original input
// - DeleteFlow for deleting messages from SQS queues using receipt handles extracted from inputs
//
// Features:
// - SQS message reading with configurable batching and polling
// - SQS message sending with result handling and original input preservation
// - SQS message deletion with flexible receipt handle extraction
//
// This package requires an externally configured AWS client to be passed in, allowing the caller
// to handle authentication and AWS configuration according to their own requirements.
package sqs
