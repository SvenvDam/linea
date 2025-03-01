package sqs

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/svenvdam/linea/util"
)

// MockSQSDeleteClient is a mock implementation of SQSDeleteClient for testing
type MockSQSDeleteClient struct {
	DeleteMessageFunc func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// DeleteMessage implements the SQSDeleteClient interface
func (m *MockSQSDeleteClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return m.DeleteMessageFunc(ctx, params, optFns...)
}

// TestMessage is a simple struct for testing the DeleteFlow
type TestMessage struct {
	ID            string
	ReceiptHandle string
	Content       string
}

func TestDeleteFlow(t *testing.T) {
	tests := []struct {
		name           string
		config         DeleteFlowConfig
		input          TestMessage
		mockResponse   *sqs.DeleteMessageOutput
		mockError      error
		expectedInput  *sqs.DeleteMessageInput
		expectNilError bool
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
			mockResponse: &sqs.DeleteMessageOutput{},
			mockError:    nil,
			expectedInput: &sqs.DeleteMessageInput{
				QueueUrl:      util.AsPtr("https://sqs.example.com/queue"),
				ReceiptHandle: util.AsPtr("receipt123"),
			},
			expectNilError: true,
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
			mockResponse: nil,
			mockError:    errors.New("sqs error"),
			expectedInput: &sqs.DeleteMessageInput{
				QueueUrl:      util.AsPtr("https://sqs.example.com/queue"),
				ReceiptHandle: util.AsPtr("receipt123"),
			},
			expectNilError: false,
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
			mockResponse:   nil,
			mockError:      nil,
			expectedInput:  nil, // No call to DeleteMessage expected
			expectNilError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context for the test
			ctx := context.Background()

			// Track if DeleteMessage was called
			deleteMessageCalled := false

			// Create a mock client with the expected response
			mockClient := &MockSQSDeleteClient{
				DeleteMessageFunc: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
					// Mark that DeleteMessage was called
					deleteMessageCalled = true

					// If we don't expect a call to DeleteMessage, fail the test
					if tt.expectedInput == nil {
						t.Fatal("DeleteMessage was called but no call was expected")
					}

					// Verify the input matches what we expect
					assert.Equal(t, tt.expectedInput.QueueUrl, params.QueueUrl)
					assert.Equal(t, tt.expectedInput.ReceiptHandle, params.ReceiptHandle)

					return tt.mockResponse, tt.mockError
				},
			}

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

			// Check that we got exactly one result
			assert.Len(t, resultSlice, 1, "Expected exactly one result")

			// Check the result matches what we expect
			result0 := resultSlice[0]

			// Check the original input is preserved
			assert.Equal(t, tt.input, result0.Original)

			// Check if DeleteMessage was called when expected
			if tt.expectedInput != nil {
				assert.True(t, deleteMessageCalled, "Expected DeleteMessage to be called")
			} else {
				assert.False(t, deleteMessageCalled, "Expected DeleteMessage not to be called")
			}

			// Check the output
			if tt.mockResponse == nil {
				assert.Nil(t, result0.Output)
			} else {
				assert.NotNil(t, result0.Output)
			}

			// Check the error
			if tt.expectNilError {
				assert.Nil(t, result0.Error)
			} else {
				assert.NotNil(t, result0.Error)
			}
		})
	}
}
