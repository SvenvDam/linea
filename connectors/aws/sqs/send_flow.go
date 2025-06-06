package sqs

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
)

// SQSSendClient defines the interface for SQS operations needed by the SendFlow
type SQSSendClient interface {
	SendMessage(
		ctx context.Context,
		params *sqs.SendMessageInput,
		optFns ...func(*sqs.Options),
	) (*sqs.SendMessageOutput, error)
}

// SendMessageResult represents the result of sending a message to SQS
type SendMessageResult[I any] struct {
	// The original item that was used to create the SQS message
	Original I

	// The output from the SQS SendMessage operation
	Output *sqs.SendMessageOutput
}

// SendFlowConfig holds configuration for the SQS send flow
type SendFlowConfig struct {
	// QueueURL is the URL of the SQS queue to send to
	QueueURL string

	// DelaySeconds is the length of time, in seconds, for which to delay a specific message
	// Valid values: 0 to 900 (15 minutes)
	// If not specified, the default value for the queue applies
	DelaySeconds int32
}

// SendFlow creates a Flow that sends messages to an SQS queue and passes the results downstream.
// For each input message, it sends it to SQS and emits a SendMessageResult containing the
// original input item and the SQS response.
// If an error occurs during sending, it will be propagated through the flow's error handling mechanism.
//
// Type Parameters:
//   - I: The type of input items that will be converted to SQS messages
//
// Parameters:
//   - client: AWS SQS client or compatible interface
//   - config: Configuration for the SQS send flow
//   - messageBuilder: Function that transforms an input item into an SQS SendMessageInput
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that sends messages to SQS and produces SendMessageResult items
func SendFlow[I any](
	client SQSSendClient,
	config SendFlowConfig,
	messageBuilder func(I) *sqs.SendMessageInput,
	opts ...core.FlowOption,
) *core.Flow[I, SendMessageResult[I]] {
	return flows.TryMap(func(ctx context.Context, elem I) (SendMessageResult[I], error) {
		// Build the message input from the input element
		msgInput := messageBuilder(elem)

		// If QueueURL is not set in the input, use the one from config
		if msgInput.QueueUrl == nil {
			msgInput.QueueUrl = &config.QueueURL
		}

		// If DelaySeconds is not set in the input and is set in config, use the one from config
		if msgInput.DelaySeconds == 0 && config.DelaySeconds > 0 {
			msgInput.DelaySeconds = config.DelaySeconds
		}

		// Send the message to SQS using the provided context
		output, err := client.SendMessage(ctx, msgInput)
		if err != nil {
			return SendMessageResult[I]{}, err
		}

		// Create the result, including the original input item
		return SendMessageResult[I]{
			Original: elem,
			Output:   output,
		}, nil
	}, opts...)
}
