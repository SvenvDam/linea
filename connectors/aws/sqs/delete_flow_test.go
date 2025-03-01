package sqs

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/util"
)

// TestMessage is a simple struct for testing the DeleteFlow
type TestMessage struct {
	ID            string
	ReceiptHandle string
	Content       string
}

func TestDeleteFlow(t *testing.T) {
	tests := []struct {
		name            string
		config          DeleteFlowConfig
		input           TestMessage
		setupMocks      func(t *testing.T, mock *MockSQSDeleteClient)
		expectedResults []DeleteMessageResult[TestMessage]
	}{
		{
			name: "successfully deletes message",
			config: DeleteFlowConfig{
				QueueURL: "https://sqs.example.com/queue",
			},
			input: TestMessage{
				ID:            "msg123",
				ReceiptHandle: "receipt123",
				Content:       "test message",
			},
			setupMocks: func(t *testing.T, mockClient *MockSQSDeleteClient) {
				expectedInput := &sqs.DeleteMessageInput{
					QueueUrl:      util.AsPtr("https://sqs.example.com/queue"),
					ReceiptHandle: util.AsPtr("receipt123"),
				}

				mockClient.EXPECT().
					DeleteMessage(mock.Anything, expectedInput, mock.Anything).
					Return(&sqs.DeleteMessageOutput{}, nil).Once()
			},
			expectedResults: []DeleteMessageResult[TestMessage]{
				{
					Original: TestMessage{
						ID:            "msg123",
						ReceiptHandle: "receipt123",
						Content:       "test message",
					},
					Output: &sqs.DeleteMessageOutput{},
					Error:  nil,
				},
			},
		},
		{
			name: "handles error from SQS",
			config: DeleteFlowConfig{
				QueueURL: "https://sqs.example.com/queue",
			},
			input: TestMessage{
				ID:            "msg123",
				ReceiptHandle: "receipt123",
				Content:       "test message",
			},
			setupMocks: func(t *testing.T, mockClient *MockSQSDeleteClient) {
				expectedInput := &sqs.DeleteMessageInput{
					QueueUrl:      util.AsPtr("https://sqs.example.com/queue"),
					ReceiptHandle: util.AsPtr("receipt123"),
				}

				mockClient.EXPECT().
					DeleteMessage(mock.Anything, expectedInput, mock.Anything).
					Return(nil, errors.New("sqs error")).Once()
			},
			expectedResults: []DeleteMessageResult[TestMessage]{
				{
					Original: TestMessage{
						ID:            "msg123",
						ReceiptHandle: "receipt123",
						Content:       "test message",
					},
					Output: nil,
					Error:  errors.New("sqs error"),
				},
			},
		},
		{
			name: "handles nil receipt handle",
			config: DeleteFlowConfig{
				QueueURL: "https://sqs.example.com/queue",
			},
			input: TestMessage{
				ID:            "msg123",
				ReceiptHandle: "", // Empty receipt handle will result in nil
				Content:       "test message",
			},
			setupMocks: func(t *testing.T, mockClient *MockSQSDeleteClient) {
				// No mock expectations because DeleteMessage should not be called
			},
			expectedResults: []DeleteMessageResult[TestMessage]{
				{
					Original: TestMessage{
						ID:            "msg123",
						ReceiptHandle: "",
						Content:       "test message",
					},
					Output: nil,
					Error:  errors.New("receipt handle is nil"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context for the test
			ctx := context.Background()

			// Set up the mock client
			mockClient := NewMockSQSDeleteClient(t)
			tt.setupMocks(t, mockClient)

			// Create a receipt handle extractor function
			receiptHandleExtractor := func(msg TestMessage) *string {
				if msg.ReceiptHandle == "" {
					return nil
				}
				return util.AsPtr(msg.ReceiptHandle)
			}

			// Create the flow
			flow := DeleteFlow(mockClient, tt.config, receiptHandleExtractor)

			// Create a stream that sends the input through the flow and captures the results
			stream := compose.SourceThroughFlowToSink(
				sources.Slice([]TestMessage{tt.input}),
				flow,
				sinks.Slice[DeleteMessageResult[TestMessage]](),
			)

			// Run the stream
			result := <-stream.Run(ctx)

			// Check that the stream completed successfully
			assert.True(t, result.Ok, "Expected stream to complete successfully")

			// Get the results from the stream result
			resultSlice := result.Value

			// Compare the results using ElementsMatch
			assert.ElementsMatch(t, tt.expectedResults, resultSlice)
		})
	}
}
