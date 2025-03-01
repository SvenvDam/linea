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

func TestSendFlow(t *testing.T) {
	tests := []struct {
		name            string
		config          SendFlowConfig
		input           string
		setupMocks      func(t *testing.T, mock *MockSQSSendClient)
		expectedResults []SendMessageResult[string]
	}{
		{
			name: "successfully sends message",
			config: SendFlowConfig{
				QueueURL:     "https://sqs.example.com/queue",
				DelaySeconds: 5,
			},
			input: "test message",
			setupMocks: func(t *testing.T, mockClient *MockSQSSendClient) {
				expectedInput := &sqs.SendMessageInput{
					QueueUrl:     util.AsPtr("https://sqs.example.com/queue"),
					MessageBody:  util.AsPtr("test message"),
					DelaySeconds: 5,
				}

				mockClient.EXPECT().
					SendMessage(mock.Anything, expectedInput, mock.Anything).
					Return(&sqs.SendMessageOutput{
						MessageId: util.AsPtr("msg123"),
					}, nil).Once()
			},
			expectedResults: []SendMessageResult[string]{
				{
					Original: "test message",
					Output: &sqs.SendMessageOutput{
						MessageId: util.AsPtr("msg123"),
					},
					Error: nil,
				},
			},
		},
		{
			name: "handles error from SQS",
			config: SendFlowConfig{
				QueueURL: "https://sqs.example.com/queue",
			},
			input: "test message",
			setupMocks: func(t *testing.T, mockClient *MockSQSSendClient) {
				expectedInput := &sqs.SendMessageInput{
					QueueUrl:    util.AsPtr("https://sqs.example.com/queue"),
					MessageBody: util.AsPtr("test message"),
				}

				mockClient.EXPECT().
					SendMessage(mock.Anything, expectedInput, mock.Anything).
					Return(nil, errors.New("sqs error")).Once()
			},
			expectedResults: []SendMessageResult[string]{
				{
					Original: "test message",
					Output:   nil,
					Error:    errors.New("sqs error"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context for the test
			ctx := context.Background()

			// Set up the mock client
			mockClient := NewMockSQSSendClient(t)
			tt.setupMocks(t, mockClient)

			// Create a message builder function for strings
			stringMessageBuilder := func(msg string) *sqs.SendMessageInput {
				return &sqs.SendMessageInput{
					MessageBody: util.AsPtr(msg),
				}
			}

			// Create the flow
			flow := SendFlow(mockClient, tt.config, stringMessageBuilder)

			// Create a stream that sends the input through the flow and captures the results
			stream := compose.SourceThroughFlowToSink(
				sources.Slice([]string{tt.input}),
				flow,
				sinks.Slice[SendMessageResult[string]](),
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
