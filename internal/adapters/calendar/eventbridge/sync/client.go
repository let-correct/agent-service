package calendarsync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

type eventbridgeClient interface {
	PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type Eventbridge struct {
	eventbridge eventbridgeClient
	busName     string
}

func (e *Eventbridge) Publish(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	entries := make([]types.PutEventsRequestEntry, 0, len(events))
	for _, event := range events {
		detail, err := json.Marshal(event.Payload)
		if err != nil {
			return fmt.Errorf("marshal event payload: %w", err)
		}
		entries = append(entries, types.PutEventsRequestEntry{
			Detail:       aws.String(string(detail)),
			DetailType:   aws.String(string(event.DetailType)),
			EventBusName: aws.String(e.busName),
			Source:       aws.String(event.Metadata.Source),
		})
	}

	out, err := e.eventbridge.PutEvents(ctx, &eventbridge.PutEventsInput{Entries: entries})
	if err != nil {
		return fmt.Errorf("put events: %w", err)
	}
	if out.FailedEntryCount > 0 {
		return fmt.Errorf("eventbridge: %d events failed to publish", out.FailedEntryCount)
	}

	return nil
}
