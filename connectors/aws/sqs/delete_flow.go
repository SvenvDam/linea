package sqs

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
)

// SQSDeleteClient defines the interface for SQS operations needed by the DeleteFlow
type SQSDeleteClient interface {
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// DeleteMessageResult represents the result of deleting a message from SQS
type DeleteMessageResult[I any] struct {
	// The original item that was used to extract the receipt handle
	Original I

	// The output from the SQS DeleteMessage operation
	Output *sqs.DeleteMessageOutput

	// Any error that occurred during the delete operation
	Error error
}

// DeleteFlowConfig holds configuration for the SQS delete flow
type DeleteFlowConfig struct {
	// QueueURL is the URL of the SQS queue to delete from
	QueueURL string
}

// DeleteFlow creates a Flow that deletes messages from an SQS queue and passes the results downstream.
// For each input item, it extracts the receipt handle using the provided function, deletes the message
// from SQS, and emits a DeleteMessageResult containing the original input item, the SQS response,
// and any error that occurred.
//
// Type Parameters:
//   - I: The type of input items that contain or can be used to extract receipt handles
//
// Parameters:
//   - client: AWS SQS client or compatible interface
//   - config: Configuration for the SQS delete flow
//   - receiptHandleExtractor: Function that extracts a receipt handle from an input item
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that deletes messages from SQS and produces DeleteMessageResult items
func DeleteFlow[I any](
	client SQSDeleteClient,
	config DeleteFlowConfig,
	receiptHandleExtractor func(I) *string,
	opts ...core.FlowOption,
) *core.Flow[I, DeleteMessageResult[I]] {
	return flows.Map(func(elem I) DeleteMessageResult[I] {
		// Extract the receipt handle from the input element
		receiptHandle := receiptHandleExtractor(elem)

		// If receipt handle is nil, create an error result
		if receiptHandle == nil {
			return DeleteMessageResult[I]{
				Original: elem,
				Output:   nil,
				Error:    errors.New("receipt handle is nil"),
			}
		}

		// Create the delete message input
		deleteInput := &sqs.DeleteMessageInput{
			QueueUrl:      &config.QueueURL,
			ReceiptHandle: receiptHandle,
		}

		// Delete the message from SQS
		output, err := client.DeleteMessage(context.Background(), deleteInput)

		// Create the result, including the original input item
		return DeleteMessageResult[I]{
			Original: elem,
			Output:   output,
			Error:    err,
		}
	}, opts...)
}
