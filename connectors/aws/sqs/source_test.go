package sqs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/test"
	"github.com/svenvdam/linea/util"
)

// Test message variables to reduce duplication
var (
	testMsg1 = types.Message{
		MessageId:     util.AsPtr("msg1"),
		Body:          util.AsPtr("message 1"),
		ReceiptHandle: util.AsPtr("receipt1"),
	}
	testMsg2 = types.Message{
		MessageId:     util.AsPtr("msg2"),
		Body:          util.AsPtr("message 2"),
		ReceiptHandle: util.AsPtr("receipt2"),
	}
	testMsg3 = types.Message{
		MessageId:     util.AsPtr("msg3"),
		Body:          util.AsPtr("message 3"),
		ReceiptHandle: util.AsPtr("receipt3"),
	}
)

func TestSource(t *testing.T) {
	tests := []struct {
		name           string
		config         SourceConfig
		mockResponses  []mockResponse
		expectedResult []types.Message
		duration       time.Duration
		expectError    bool
	}{
		{
			name: "successfully polls messages",
			config: SourceConfig{
				QueueURL:            "https://sqs.example.com/queue",
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     5,
				VisibilityTimeout:   30,
				PollInterval:        50 * time.Millisecond,
			},
			mockResponses: []mockResponse{
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{testMsg1, testMsg2},
					},
					err: nil,
				},
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{testMsg3},
					},
					err: nil,
				},
			},
			expectedResult: []types.Message{testMsg1, testMsg2, testMsg3},
			duration:       200 * time.Millisecond,
			expectError:    false,
		},
		{
			name: "handles empty responses",
			config: SourceConfig{
				QueueURL:            "https://sqs.example.com/queue",
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     1,
				VisibilityTimeout:   30,
				PollInterval:        50 * time.Millisecond,
			},
			mockResponses: []mockResponse{
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{},
					},
					err: nil,
				},
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{testMsg1},
					},
					err: nil,
				},
			},
			expectedResult: []types.Message{testMsg1},
			duration:       150 * time.Millisecond,
			expectError:    false,
		},
		{
			name: "cancels stream on error",
			config: SourceConfig{
				QueueURL:            "https://sqs.example.com/queue",
				MaxNumberOfMessages: 5,
				WaitTimeSeconds:     1,
				VisibilityTimeout:   30,
				PollInterval:        50 * time.Millisecond,
			},
			mockResponses: []mockResponse{
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{testMsg1},
					},
					err: nil,
				},
				{
					output: nil,
					err:    errors.New("connection error"),
				},
				// This response should never be reached because the stream should be cancelled
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{testMsg2},
					},
					err: nil,
				},
			},
			expectedResult: []types.Message{testMsg1},
			duration:       150 * time.Millisecond,
			expectError:    true,
		},
		{
			name: "respects max number of messages",
			config: SourceConfig{
				QueueURL:            "https://sqs.example.com/queue",
				MaxNumberOfMessages: 2,
				WaitTimeSeconds:     1,
				VisibilityTimeout:   30,
				PollInterval:        50 * time.Millisecond,
			},
			mockResponses: []mockResponse{
				{
					output: &sqs.ReceiveMessageOutput{
						Messages: []types.Message{testMsg1, testMsg2},
					},
					err: nil,
				},
			},
			expectedResult: []types.Message{testMsg1, testMsg2},
			duration:       100 * time.Millisecond,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context that will be used to run the test
			ctx := context.Background()

			// Initialize a slice to capture the messages processed
			captured := make([]types.Message, 0)

			// Set up the mock client with the responses for this test
			mockClient := &MockSQSClient{
				ReceiveMessageFunc: createMockReceiveMessageFunc(tt.mockResponses),
			}

			// Create the source using our mock client
			source := Source(mockClient, tt.config)

			// Create a stream that collects the messages
			stream := compose.SourceThroughFlowToSink(
				source,
				test.CaptureItems(&captured),
				sinks.Noop[types.Message](),
			)

			// Run the stream for the specified duration
			resultChan := stream.Run(ctx)
			time.Sleep(tt.duration)
			stream.Drain()
			result := <-resultChan

			// Check if we expected an error
			if tt.expectError {
				assert.False(t, result.Ok, "Expected stream to fail due to error")
			} else {
				assert.True(t, result.Ok, "Expected stream to complete successfully")
			}

			assert.ElementsMatch(t, tt.expectedResult, captured)
		})
	}
}

// Helper types and functions

type MockSQSClient struct {
	ReceiveMessageFunc func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
}

var _ SQSReceiveClient = (*MockSQSClient)(nil)

func (m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return m.ReceiveMessageFunc(ctx, params, optFns...)
}

type mockResponse struct {
	output *sqs.ReceiveMessageOutput
	err    error
}

func createMockReceiveMessageFunc(responses []mockResponse) func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	responseIndex := 0

	return func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
		// If we've gone through all responses, stop returning messages
		// This prevents continuous polling in the test
		if responseIndex >= len(responses) {
			// Return empty response to indicate no more messages
			return &sqs.ReceiveMessageOutput{Messages: []types.Message{}}, nil
		}

		// Get the current response and increment the index
		response := responses[responseIndex]
		responseIndex++

		return response.output, response.err
	}
}
