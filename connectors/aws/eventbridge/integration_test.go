package eventbridge

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/require"
	"github.com/svenvdam/linea/compose"
	"github.com/svenvdam/linea/connectors/aws/util"
	"github.com/svenvdam/linea/connectors/aws/util/test"
	"github.com/svenvdam/linea/sinks"
	"github.com/svenvdam/linea/sources"
	"github.com/testcontainers/testcontainers-go"
)

// TestEvent is the structure used for test events
type TestEvent struct {
	ID      string    `json:"id"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// testInfrastructure holds all the AWS resources and clients needed for the test
type testInfrastructure struct {
	ctx               context.Context
	container         testcontainers.Container
	eventBridgeClient *eventbridge.Client
	sqsClient         *sqs.Client
	eventBusName      string
	queueURL          string
}

// TestEventBridgeIntegration demonstrates how to use the EventBridge producer
// and verifies that events are actually delivered to EventBridge
func TestEventBridgeIntegration(t *testing.T) {
	// Setup test context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup infrastructure
	infra := setupTestInfrastructure(t, ctx)
	defer teardownTestInfrastructure(ctx, infra)

	// Create EventBridge rule to forward events to SQS
	createForwardingRule(t, infra)

	// Prepare and send test events
	events := createTestEvents()
	sendResults := sendTestEvents(t, infra, events)
	verifyPutEventsResults(t, events, sendResults)

	// Verify events were delivered to SQS
	receivedEvents := receiveEventsFromSQS(t, infra, len(events))
	verifyReceivedEvents(t, events, receivedEvents)
}

// setupTestInfrastructure sets up the LocalStack container, clients, and AWS resources
func setupTestInfrastructure(t *testing.T, ctx context.Context) *testInfrastructure {
	t.Helper()

	// Setup localstack container for testing
	awsCfg, container, err := test.SetupLocalstack(ctx)
	require.NoError(t, err)

	// Create clients
	eventBridgeClient := eventbridge.NewFromConfig(*awsCfg)
	sqsClient := sqs.NewFromConfig(*awsCfg)

	// Create event bus for testing
	eventBusName := "test-event-bus"
	_, err = eventBridgeClient.CreateEventBus(ctx, &eventbridge.CreateEventBusInput{
		Name: &eventBusName,
	})
	require.NoError(t, err)

	// Create SQS queue to receive events
	queueName := "test-event-queue"
	createQueueResponse, err := sqsClient.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: &queueName,
	})
	require.NoError(t, err)
	require.NotNil(t, createQueueResponse.QueueUrl)

	return &testInfrastructure{
		ctx:               ctx,
		container:         container,
		eventBridgeClient: eventBridgeClient,
		sqsClient:         sqsClient,
		eventBusName:      eventBusName,
		queueURL:          *createQueueResponse.QueueUrl,
	}
}

// teardownTestInfrastructure cleans up test resources
func teardownTestInfrastructure(ctx context.Context, infra *testInfrastructure) {
	if infra.container != nil {
		_ = infra.container.Terminate(ctx)
	}
}

// createForwardingRule creates an EventBridge rule that forwards events to SQS
func createForwardingRule(t *testing.T, infra *testInfrastructure) {
	t.Helper()

	// Get queue ARN for the EventBridge rule target
	getQueueAttributesResponse, err := infra.sqsClient.GetQueueAttributes(infra.ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &infra.queueURL,
		AttributeNames: []sqstypes.QueueAttributeName{
			sqstypes.QueueAttributeNameQueueArn,
		},
	})
	require.NoError(t, err)
	queueARN := getQueueAttributesResponse.Attributes[string(sqstypes.QueueAttributeNameQueueArn)]
	require.NotEmpty(t, queueARN, "Queue ARN should not be empty")

	// Create rule
	ruleName := "test-forward-to-sqs"
	eventPattern := `{"source": ["test.source"], "detail-type": ["TestEvent"]}`

	_, err = infra.eventBridgeClient.PutRule(infra.ctx, &eventbridge.PutRuleInput{
		Name:         &ruleName,
		EventBusName: &infra.eventBusName,
		EventPattern: &eventPattern,
		State:        types.RuleStateEnabled,
	})
	require.NoError(t, err)

	// Add SQS as a target for the rule
	_, err = infra.eventBridgeClient.PutTargets(infra.ctx, &eventbridge.PutTargetsInput{
		Rule:         &ruleName,
		EventBusName: &infra.eventBusName,
		Targets: []types.Target{
			{
				Id:  aws.String("SQSTarget"),
				Arn: aws.String(queueARN),
			},
		},
	})
	require.NoError(t, err)

	// Wait a moment for the rule to become active
	time.Sleep(200 * time.Millisecond)
}

// createTestEvents returns test events to send to EventBridge
func createTestEvents() []TestEvent {
	return []TestEvent{
		{
			ID:      "event-1",
			Message: "Test message 1",
			Time:    time.Now().Round(time.Second), // Round to seconds for easier comparison
		},
		{
			ID:      "event-2",
			Message: "Test message 2",
			Time:    time.Now().Round(time.Second), // Round to seconds for easier comparison
		},
	}
}

// sendTestEvents sends the test events to EventBridge
func sendTestEvents(t *testing.T, infra *testInfrastructure, events []TestEvent) []PutEventsResult[TestEvent] {
	t.Helper()

	// Configure EventBridge producer
	config := SendFlowConfig{
		EventBusName: infra.eventBusName,
	}

	// Create a flow to send events to EventBridge
	flow := SendFlow(infra.eventBridgeClient, config, func(event TestEvent) *eventbridge.PutEventsInput {
		// Convert the event to JSON for the Detail field
		detailBytes, err := json.Marshal(event)
		require.NoError(t, err)
		detailJSON := string(detailBytes)

		return &eventbridge.PutEventsInput{
			Entries: []types.PutEventsRequestEntry{
				{
					// EventBusName will be set from the config
					Source:     util.AsPtr("test.source"),
					DetailType: util.AsPtr("TestEvent"),
					Detail:     util.AsPtr(detailJSON),
				},
			},
		}
	})

	// Create a stream that sends test events through the flow
	stream := compose.SourceThroughFlowToSink(
		sources.Slice(events),
		flow,
		sinks.Slice[PutEventsResult[TestEvent]](),
	)

	// Run the stream
	result := <-stream.Run(infra.ctx)
	require.NoError(t, result.Err)
	return result.Value
}

// verifyPutEventsResults verifies the responses from EventBridge
func verifyPutEventsResults(t *testing.T, events []TestEvent, results []PutEventsResult[TestEvent]) {
	t.Helper()

	require.Len(t, results, len(events))
	for i, res := range results {
		require.Equal(t, events[i], res.Original)
		require.Equal(t, int32(0), res.Output.FailedEntryCount)
		require.Len(t, res.Output.Entries, 1)
		require.NotEmpty(t, res.Output.Entries[0].EventId)
	}
}

// receiveEventsFromSQS polls the SQS queue to receive the forwarded events
func receiveEventsFromSQS(t *testing.T, infra *testInfrastructure, expectedCount int) []TestEvent {
	t.Helper()

	// Wait for events to be processed by EventBridge and forwarded to SQS
	// This can take a moment, so we'll add a delay and then poll
	time.Sleep(200 * time.Millisecond)

	// Now verify that the events were actually received in the SQS queue
	receivedEvents := make([]TestEvent, 0, expectedCount)

	// Poll SQS until we get the expected number of messages or timeout
	for attempts := 0; len(receivedEvents) < expectedCount && attempts < 10; attempts++ {
		// Receive messages from SQS
		receiveResponse, err := infra.sqsClient.ReceiveMessage(infra.ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            &infra.queueURL,
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     1,
		})
		require.NoError(t, err)

		// Process received messages
		for _, msg := range receiveResponse.Messages {
			require.NotNil(t, msg.Body)

			// The SQS message contains the entire EventBridge event
			var ebEvent struct {
				Detail TestEvent `json:"detail"`
			}
			err = json.Unmarshal([]byte(*msg.Body), &ebEvent)
			require.NoError(t, err)

			receivedEvents = append(receivedEvents, ebEvent.Detail)

			// Delete the message from the queue
			_, err = infra.sqsClient.DeleteMessage(infra.ctx, &sqs.DeleteMessageInput{
				QueueUrl:      &infra.queueURL,
				ReceiptHandle: msg.ReceiptHandle,
			})
			require.NoError(t, err)
		}

		if len(receivedEvents) < expectedCount {
			// Wait a bit before polling again
			time.Sleep(1 * time.Second)
		}
	}

	// Verify that we received the correct number of events
	require.Len(t, receivedEvents, expectedCount, "Did not receive the expected number of events in SQS")
	return receivedEvents
}

// verifyReceivedEvents verifies that the received events match the sent events
func verifyReceivedEvents(t *testing.T, sentEvents []TestEvent, receivedEvents []TestEvent) {
	t.Helper()

	// Verify the content of the received events (order may not be preserved)
	for _, sentEvent := range sentEvents {
		found := false
		for _, receivedEvent := range receivedEvents {
			if sentEvent.ID == receivedEvent.ID {
				require.Equal(t, sentEvent.Message, receivedEvent.Message)
				// Check that times are roughly equal (might have some serialization differences)
				timeDiff := sentEvent.Time.Sub(receivedEvent.Time)
				require.LessOrEqual(t, timeDiff.Abs(), time.Second, "Event times should be approximately equal")
				found = true
				break
			}
		}
		require.True(t, found, "Could not find event with ID %s in received events", sentEvent.ID)
	}
}
