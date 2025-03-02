package sqs

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/connectors/aws/util/test"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
	"github.com/svenvdam/linea/sinks"
)

// TestMessageProcessor tests a complete SQS processing flow:
// 1. Read messages from source queue
// 2. Transform messages
// 3. Write transformed messages to destination queue
// 4. Delete original messages from source queue
func TestMessageProcessor(t *testing.T) {
	// Setup test context with timeout
	ctx := context.Background()

	// Setup localstack container
	awsCfg, container, err := test.SetupLocalstack(ctx)
	require.NoError(t, err)
	defer func() {
		_ = container.Terminate(ctx)
	}()

	// Create SQS client
	sqsClient := sqs.NewFromConfig(*awsCfg)

	// Setup test queues with short visibility timeout to verify actual deletion
	sourceQueueURL, err := setupQueue(ctx, sqsClient, "source-queue", 1) // 1 second visibility timeout
	require.NoError(t, err)
	destQueueURL, err := setupQueue(ctx, sqsClient, "dest-queue", 30) // normal visibility timeout
	require.NoError(t, err)

	// Send test messages to source queue
	testMessages := []string{
		"hello world",
		"test message",
		"process me",
	}
	err = sendTestMessages(ctx, sqsClient, sourceQueueURL, testMessages)
	require.NoError(t, err)

	// Create message processor that:
	// 1. Reads from source queue
	// 2. Transforms messages to uppercase
	// 3. Sends to destination queue
	// 4. Deletes from source queue

	// Create message processor
	processor := createMessageProcessor(sqsClient, sourceQueueURL, destQueueURL)
	defer processor.Cancel()

	// Run the processor for a short time to process messages
	resultChan := processor.Run(ctx)

	// Sleep to allow processing to complete
	time.Sleep(2 * time.Second)

	// Drain the processor
	processor.Drain()
	result := <-resultChan
	assert.True(t, result.Ok, "Expected processor to complete successfully")

	// Read messages from destination queue to verify they were transformed and sent
	receivedMessages, err := receiveAllMessages(ctx, sqsClient, destQueueURL)
	require.NoError(t, err)

	// Verify all messages were properly transformed
	expectedMessages := make([]string, len(testMessages))
	for i, msg := range testMessages {
		expectedMessages[i] = strings.ToUpper(msg)
	}

	// Extract bodies from received messages for easier comparison
	var receivedBodies []string
	for _, msg := range receivedMessages {
		receivedBodies = append(receivedBodies, *msg.Body)
	}

	// Verify that all expected messages are in the destination queue
	assert.ElementsMatch(t, expectedMessages, receivedBodies,
		"Destination queue should contain all transformed messages")

	// Wait for visibility timeout to expire to ensure we're testing actual deletion
	// and not just temporary invisibility
	time.Sleep(2 * time.Second) // Wait longer than the 1-second visibility timeout

	// Read messages from source queue to verify they were deleted
	sourceMessages, err := receiveAllMessages(ctx, sqsClient, sourceQueueURL)
	require.NoError(t, err)

	// Verify the source queue is empty (all messages were deleted)
	assert.Empty(t, sourceMessages, "Source queue should be empty after processing (messages should be deleted, not just hidden)")
}

// setupQueue creates an SQS queue and returns its URL
func setupQueue(ctx context.Context, client *sqs.Client, queueName string, visibilityTimeoutSeconds int) (string, error) {
	resp, err := client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
		Attributes: map[string]string{
			"VisibilityTimeout": fmt.Sprintf("%d", visibilityTimeoutSeconds),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to create queue %s: %w", queueName, err)
	}
	return *resp.QueueUrl, nil
}

// sendTestMessages sends a list of messages to the specified queue
func sendTestMessages(ctx context.Context, client *sqs.Client, queueURL string, messages []string) error {
	for _, msg := range messages {
		_, err := client.SendMessage(ctx, &sqs.SendMessageInput{
			QueueUrl:    aws.String(queueURL),
			MessageBody: aws.String(msg),
		})
		if err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}
	return nil
}

// receiveAllMessages receives all available messages from the specified queue
func receiveAllMessages(ctx context.Context, client *sqs.Client, queueURL string) ([]types.Message, error) {
	var allMessages []types.Message

	// Keep receiving messages until there are no more
	for {
		resp, err := client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     2,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to receive messages: %w", err)
		}

		if len(resp.Messages) == 0 {
			break
		}

		allMessages = append(allMessages, resp.Messages...)
	}

	return allMessages, nil
}

// createMessageProcessor creates a stream that processes messages from SQS
// It reads from source queue, transforms messages, writes to destination queue, and deletes from source
func createMessageProcessor(sqsClient *sqs.Client, sourceQueueURL, destQueueURL string) *core.Stream[struct{}] {
	// Configure source
	sourceConfig := SourceConfig{
		QueueURL:            sourceQueueURL,
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     5,
		VisibilityTimeout:   30,
		PollInterval:        1 * time.Second,
	}

	// Create source that reads from source queue
	source := Source(sqsClient, sourceConfig)

	// Create transformation flow that converts message body to uppercase
	transformFlow := flows.Map(func(msg types.Message) types.Message {
		// Create a new message with uppercase body
		transformedMsg := msg
		if msg.Body != nil {
			upperBody := strings.ToUpper(*msg.Body)
			transformedMsg.Body = &upperBody
		}
		return transformedMsg
	})

	// Create flow to send transformed message to destination queue
	sendConfig := SendFlowConfig{
		QueueURL: destQueueURL,
	}
	sendFlow := SendFlow(sqsClient, sendConfig, func(msg types.Message) *sqs.SendMessageInput {
		return &sqs.SendMessageInput{
			MessageBody: msg.Body,
		}
	})

	// Create flow to delete original message from source queue
	deleteConfig := DeleteFlowConfig{
		QueueURL: sourceQueueURL,
	}
	deleteFlow := DeleteFlow(sqsClient, deleteConfig, func(result SendMessageResult[types.Message]) *string {
		// Extract receipt handle from the original message
		return result.Original.ReceiptHandle
	})

	// Combine everything into a processing pipeline
	return compose.SourceThroughFlowToSink3(
		source,
		transformFlow,
		sendFlow,
		deleteFlow,
		sinks.CancelIf(func(msg DeleteMessageResult[SendMessageResult[types.Message]]) bool {
			return msg.Error != nil
		}),
	)
}
