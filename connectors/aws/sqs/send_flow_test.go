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

// MockSQSSendClient is a mock implementation of SQSSendClient for testing
type MockSQSSendClient struct {
	SendMessageFunc func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// SendMessage implements the SQSSendClient interface
func (m *MockSQSSendClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return m.SendMessageFunc(ctx, params, optFns...)
}

func TestSendFlow(t *testing.T) {
	tests := []struct {
		name          string
		config        SendFlowConfig
		input         string
		mockResponse  *sqs.SendMessageOutput
		mockError     error
		expectedInput *sqs.SendMessageInput
	}{
		{
			name: "successfully sends message",
			config: SendFlowConfig{
				QueueURL:     "https://sqs.example.com/queue",
				DelaySeconds: 5,
			},
			input: "test message",
			mockResponse: &sqs.SendMessageOutput{
				MessageId: util.AsPtr("msg123"),
			},
			mockError: nil,
			expectedInput: &sqs.SendMessageInput{
				QueueUrl:     util.AsPtr("https://sqs.example.com/queue"),
				MessageBody:  util.AsPtr("test message"),
				DelaySeconds: 5,
			},
		},
		{
			name: "handles error from SQS",
			config: SendFlowConfig{
				QueueURL: "https://sqs.example.com/queue",
			},
			input:        "test message",
			mockResponse: nil,
			mockError:    errors.New("sqs error"),
			expectedInput: &sqs.SendMessageInput{
				QueueUrl:    util.AsPtr("https://sqs.example.com/queue"),
				MessageBody: util.AsPtr("test message"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context for the test
			ctx := context.Background()

			// Create a mock client with the expected response
			mockClient := &MockSQSSendClient{
				SendMessageFunc: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
					// Verify the input matches what we expect
					assert.Equal(t, tt.expectedInput.QueueUrl, params.QueueUrl)
					assert.Equal(t, tt.expectedInput.MessageBody, params.MessageBody)
					assert.Equal(t, tt.expectedInput.DelaySeconds, params.DelaySeconds)

					return tt.mockResponse, tt.mockError
				},
			}

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

			// Check that we got exactly one result
			assert.Len(t, resultSlice, 1, "Expected exactly one result")

			// Check the result matches what we expect
			result0 := resultSlice[0]

			// Check the original input is preserved
			assert.Equal(t, tt.input, result0.Original)

			// Check the output
			if tt.mockResponse == nil {
				assert.Nil(t, result0.Output)
			} else {
				assert.Equal(t, tt.mockResponse.MessageId, result0.Output.MessageId)
			}

			// Check the error
			if tt.mockError == nil {
				assert.Nil(t, result0.Error)
			} else {
				assert.Equal(t, tt.mockError.Error(), result0.Error.Error())
			}
		})
	}
}
