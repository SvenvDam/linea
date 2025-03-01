package sqs

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sources"
)

// SQSClient defines the interface for SQS operations needed by the Source
type SQSClient interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
}

// SourceConfig holds configuration for the SQS source
type SourceConfig struct {
	// QueueURL is the URL of the SQS queue to read from
	QueueURL string

	// MaxNumberOfMessages is the maximum number of messages to receive at once (1-10)
	// If not specified, defaults to 10
	MaxNumberOfMessages int32

	// WaitTimeSeconds is the duration (in seconds) to wait for messages (0-20)
	// If not specified, defaults to 20 (long polling)
	WaitTimeSeconds int32

	// VisibilityTimeout is the duration (in seconds) that messages are hidden from subsequent retrieve requests
	// If not specified, defaults to 30 seconds
	VisibilityTimeout int32

	// PollInterval is the duration to wait between polling attempts when no messages are received
	// If not specified, defaults to 1 second
	PollInterval time.Duration
}

// Source creates a Source that reads messages from an SQS queue.
// It continuously polls the queue and emits messages until the context is canceled or an error occurs.
//
// Parameters:
//   - client: AWS SQS client or compatible interface
//   - config: Configuration for the SQS source
//   - opts: Optional configuration options for the source
//
// Returns a Source that produces SQS messages
func Source(
	client SQSClient,
	config SourceConfig,
	opts ...core.SourceOption,
) *core.Source[types.Message] {

	// Create a polling function that returns:
	// - a pointer to a slice of messages from the SQS queue (or nil if no messages)
	// - a boolean indicating if there are likely more messages (more)
	// - an error if one occurred during polling
	pollFunc := func(ctx context.Context) (*[]types.Message, bool, error) {
		// Poll SQS for messages
		resp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &config.QueueURL,
			MaxNumberOfMessages: config.MaxNumberOfMessages,
			WaitTimeSeconds:     config.WaitTimeSeconds,
			VisibilityTimeout:   config.VisibilityTimeout,
		})

		// If there was an error, return nil and the error
		if err != nil {
			return nil, false, err
		}

		// If there are no messages, return nil but no error
		if len(resp.Messages) == 0 {
			return nil, false, nil
		}

		// If the number of messages received is equal to the max number of messages,
		// there are likely more messages to receive, so return true for more
		messages := resp.Messages
		return &messages, len(resp.Messages) == int(config.MaxNumberOfMessages), nil
	}

	// Use sources.Poll to create a source that emits slices of messages
	sliceSource := sources.Poll(pollFunc, config.PollInterval, opts...)

	// Create a stream that connects the slice source to a Flatten flow
	// This will convert the source of message slices to a source of individual messages
	return compose.SourceThroughFlow(
		sliceSource,
		flows.Flatten[types.Message](),
	)
}
