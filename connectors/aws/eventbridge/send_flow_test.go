package eventbridge

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/connectors/aws/eventbridge/mocks"
	"github.com/svenvdam/linea/connectors/aws/util"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
)

func TestSendFlow(t *testing.T) {
	tests := []struct {
		name            string
		config          SendFlowConfig
		input           string
		setupMocks      func(t *testing.T, mock *mocks.MockEventBridgeSendClient)
		expectedResults []PutEventsResult[string]
		expectedErr     error
	}{
		{
			name: "successfully sends event",
			config: SendFlowConfig{
				EventBusName: "test-event-bus",
			},
			input: "test event",
			setupMocks: func(t *testing.T, mockClient *mocks.MockEventBridgeSendClient) {
				testDetail := `{"id":"123","value":"test"}`
				testSource := "test.source"
				testDetailType := "TestEvent"

				expectedInput := &eventbridge.PutEventsInput{
					Entries: []types.PutEventsRequestEntry{
						{
							EventBusName: util.AsPtr("test-event-bus"),
							Source:       util.AsPtr(testSource),
							DetailType:   util.AsPtr(testDetailType),
							Detail:       util.AsPtr(testDetail),
						},
					},
				}

				mockClient.EXPECT().
					PutEvents(mock.Anything, expectedInput).
					Return(&eventbridge.PutEventsOutput{
						FailedEntryCount: 0,
						Entries: []types.PutEventsResultEntry{
							{
								EventId: util.AsPtr("event-id-123"),
							},
						},
					}, nil)
			},
			expectedResults: []PutEventsResult[string]{
				{
					Original: "test event",
					Output: &eventbridge.PutEventsOutput{
						FailedEntryCount: 0,
						Entries: []types.PutEventsResultEntry{
							{
								EventId: util.AsPtr("event-id-123"),
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "handles error from EventBridge",
			config: SendFlowConfig{
				EventBusName: "test-event-bus",
			},
			input: "test event",
			setupMocks: func(t *testing.T, mockClient *mocks.MockEventBridgeSendClient) {
				mockClient.EXPECT().
					PutEvents(mock.Anything, mock.Anything).
					Return(nil, errors.New("eventbridge error"))
			},
			expectedResults: nil,
			expectedErr:     errors.New("eventbridge error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a context for the test
			ctx := context.Background()

			// Set up the mock client
			mockClient := mocks.NewMockEventBridgeSendClient(t)
			tt.setupMocks(t, mockClient)

			// Create an event builder function for strings
			eventBuilder := func(msg string) *eventbridge.PutEventsInput {
				return &eventbridge.PutEventsInput{
					Entries: []types.PutEventsRequestEntry{
						{
							Source:     util.AsPtr("test.source"),
							DetailType: util.AsPtr("TestEvent"),
							Detail:     util.AsPtr(`{"id":"123","value":"test"}`),
						},
					},
				}
			}

			// Create the flow
			flow := SendFlow(mockClient, tt.config, eventBuilder)

			// Create a stream that sends the input through the flow and captures the results
			stream := compose.SourceThroughFlowToSink(
				sources.Slice([]string{tt.input}),
				flow,
				sinks.Slice[PutEventsResult[string]](),
			)

			// Run the stream
			result := <-stream.Run(ctx)

			assert.ElementsMatch(t, tt.expectedResults, result.Value)
			assert.Equal(t, tt.expectedErr, result.Err)
		})
	}
}
