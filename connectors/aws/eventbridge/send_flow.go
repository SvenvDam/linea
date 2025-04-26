package eventbridge

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/svenvdam/linea/core"
	"github.com/svenvdam/linea/flows"
)

// EventBridgeSendClient defines the interface for EventBridge operations needed by the SendFlow
type EventBridgeSendClient interface {
	PutEvents(
		ctx context.Context,
		params *eventbridge.PutEventsInput,
		optFns ...func(*eventbridge.Options),
	) (*eventbridge.PutEventsOutput, error)
}

// PutEventsResult represents the result of sending events to EventBridge
type PutEventsResult[I any] struct {
	// The original item that was used to create the EventBridge event
	Original I

	// The output from the EventBridge PutEvents operation
	Output *eventbridge.PutEventsOutput
}

// SendFlowConfig holds configuration for the EventBridge send flow
type SendFlowConfig struct {
	// EventBusName is the name of the EventBridge bus to send to
	// If not specified, the default event bus will be used
	EventBusName string
}

// SendFlow creates a Flow that sends events to an EventBridge event bus and passes the results downstream.
// For each input event, it sends it to EventBridge and emits a PutEventsResult containing the
// original input item and the EventBridge response.
// If an error occurs during sending, it will be propagated through the flow's error handling mechanism.
//
// Type Parameters:
//   - I: The type of input items that will be converted to EventBridge events
//
// Parameters:
//   - client: AWS EventBridge client or compatible interface
//   - config: Configuration for the EventBridge send flow
//   - eventsBuilder: Function that transforms an input item into an EventBridge PutEventsInput
//   - opts: Optional FlowOption functions to configure the flow
//
// Returns a Flow that sends events to EventBridge and produces PutEventsResult items
func SendFlow[I any](
	client EventBridgeSendClient,
	config SendFlowConfig,
	eventsBuilder func(I) *eventbridge.PutEventsInput,
	opts ...core.FlowOption,
) *core.Flow[I, PutEventsResult[I]] {
	return flows.TryMap(func(ctx context.Context, elem I) (PutEventsResult[I], error) {
		// Build the events input from the input element
		eventsInput := eventsBuilder(elem)

		// If EventBusName is set in config, ensure it's applied to entries that don't specify one
		if config.EventBusName != "" {
			for i := range eventsInput.Entries {
				if eventsInput.Entries[i].EventBusName == nil {
					eventsInput.Entries[i].EventBusName = &config.EventBusName
				}
			}
		}

		// Send the events to EventBridge using the provided context
		output, err := client.PutEvents(ctx, eventsInput)
		if err != nil {
			return PutEventsResult[I]{}, err
		}

		// Create the result, including the original input item
		return PutEventsResult[I]{
			Original: elem,
			Output:   output,
		}, nil
	}, opts...)
}
