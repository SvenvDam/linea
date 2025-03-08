package sqs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/connectors/aws/sqs/mocks"
	"github.com/svenvdam/linea/connectors/aws/util"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/test"
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
		expectedResult []types.Message
		duration       time.Duration
		expectedErr    error
		setupMocks     func(t *testing.T, mock *mocks.MockSQSReceiveClient)
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
			expectedResult: []types.Message{testMsg1, testMsg2, testMsg3},
			duration:       200 * time.Millisecond,
			expectedErr:    nil,
			setupMocks: func(t *testing.T, mockClient *mocks.MockSQSReceiveClient) {
				expectedInput := &sqs.ReceiveMessageInput{
					QueueUrl:            util.AsPtr("https://sqs.example.com/queue"),
					MaxNumberOfMessages: 5,
					WaitTimeSeconds:     5,
					VisibilityTimeout:   30,
				}

				// First call returns all three test messages
				firstCall := mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything)
				firstCall.Return(&sqs.ReceiveMessageOutput{
					Messages: []types.Message{testMsg1, testMsg2, testMsg3},
				}, nil).Once()

				// Allow additional calls with empty responses
				mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything).
					Return(&sqs.ReceiveMessageOutput{
						Messages: []types.Message{},
					}, nil).
					Maybe()
			},
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
			expectedResult: []types.Message{testMsg1},
			duration:       150 * time.Millisecond,
			expectedErr:    nil,
			setupMocks: func(t *testing.T, mockClient *mocks.MockSQSReceiveClient) {
				expectedInput := &sqs.ReceiveMessageInput{
					QueueUrl:            util.AsPtr("https://sqs.example.com/queue"),
					MaxNumberOfMessages: 5,
					WaitTimeSeconds:     1,
					VisibilityTimeout:   30,
				}

				// First call returns empty response
				firstCall := mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything)
				firstCall.Return(&sqs.ReceiveMessageOutput{
					Messages: []types.Message{},
				}, nil).Once()

				// Second call returns message 1
				secondCall := mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything)
				secondCall.Return(&sqs.ReceiveMessageOutput{
					Messages: []types.Message{testMsg1},
				}, nil).Once()

				// Allow additional calls with empty responses
				mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything).
					Return(&sqs.ReceiveMessageOutput{
						Messages: []types.Message{},
					}, nil).
					Maybe()
			},
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
			expectedResult: []types.Message{testMsg1},
			duration:       150 * time.Millisecond,
			expectedErr:    errors.New("connection error"),
			setupMocks: func(t *testing.T, mockClient *mocks.MockSQSReceiveClient) {
				expectedInput := &sqs.ReceiveMessageInput{
					QueueUrl:            util.AsPtr("https://sqs.example.com/queue"),
					MaxNumberOfMessages: 5,
					WaitTimeSeconds:     1,
					VisibilityTimeout:   30,
				}

				// First call returns message 1
				firstCall := mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything)
				firstCall.Return(&sqs.ReceiveMessageOutput{
					Messages: []types.Message{testMsg1},
				}, nil).Once()

				// Second call returns an error
				secondCall := mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything)
				secondCall.Return(nil, errors.New("connection error")).Once()

				// No additional calls expected because the stream should be cancelled
			},
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
			expectedResult: []types.Message{testMsg1, testMsg2},
			duration:       100 * time.Millisecond,
			expectedErr:    nil,
			setupMocks: func(t *testing.T, mockClient *mocks.MockSQSReceiveClient) {
				expectedInput := &sqs.ReceiveMessageInput{
					QueueUrl:            util.AsPtr("https://sqs.example.com/queue"),
					MaxNumberOfMessages: 2,
					WaitTimeSeconds:     1,
					VisibilityTimeout:   30,
				}

				// First call returns messages 1 and 2
				firstCall := mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything)
				firstCall.Return(&sqs.ReceiveMessageOutput{
					Messages: []types.Message{testMsg1, testMsg2},
				}, nil).Once()

				// Allow additional calls with empty responses
				mockClient.EXPECT().
					ReceiveMessage(mock.Anything, expectedInput, mock.Anything).
					Return(&sqs.ReceiveMessageOutput{
						Messages: []types.Message{},
					}, nil).
					Maybe()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context that will be used to run the test
			ctx := context.Background()

			mockClient := mocks.NewMockSQSReceiveClient(t)
			tt.setupMocks(t, mockClient)

			source := Source(mockClient, tt.config)

			stream := compose.SourceThroughFlowToSink(
				source,
				test.CheckItems(t, func(t *testing.T, elems []types.Message) {
					assert.Equal(t, tt.expectedResult, elems)
				}),
				sinks.Noop[types.Message](),
			)

			// Run the stream for the specified duration
			resultChan := stream.Run(ctx)
			time.Sleep(tt.duration)
			stream.Drain()
			result := <-resultChan

			assert.Equal(t, tt.expectedErr, result.Err)
		})
	}
}
