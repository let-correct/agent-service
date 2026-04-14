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

const (
	maxBatchSize = 10
	maxRetries   = 3
)

type eventbridgeClient interface {
	PutEvents(ctx context.Context, params *eventbridge.PutEventsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutEventsOutput, error)
}

type Eventbridge struct {
	eventbridge eventbridgeClient
	busName     string
}

func New(eb *eventbridge.Client, busName string) *Eventbridge {
	return &Eventbridge{eventbridge: eb, busName: busName}
}

func (e *Eventbridge) Publish(ctx context.Context, events []domain.Event) error {
	if len(events) == 0 {
		return nil
	}

	entries := make([]types.PutEventsRequestEntry, 0, len(events))
	for _, event := range events {
		detail, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		entries = append(entries, types.PutEventsRequestEntry{
			Detail:       aws.String(string(detail)),
			DetailType:   aws.String(string(event.DetailType)),
			EventBusName: aws.String(e.busName),
			Source:       aws.String(event.Metadata.Source),
		})
	}

	for i := 0; i < len(entries); i += maxBatchSize {
		batch := entries[i:min(i+maxBatchSize, len(entries))]
		if err := e.putWithRetry(ctx, batch); err != nil {
			return err
		}
	}

	return nil
}

func (e *Eventbridge) putWithRetry(ctx context.Context, entries []types.PutEventsRequestEntry) error {
	for attempt := range maxRetries {
		out, err := e.eventbridge.PutEvents(ctx, &eventbridge.PutEventsInput{Entries: entries})
		if err != nil {
			return fmt.Errorf("put events: %w", err)
		}
		if out.FailedEntryCount == 0 {
			return nil
		}

		var failed []types.PutEventsRequestEntry
		for i, result := range out.Entries {
			if result.ErrorCode != nil {
				failed = append(failed, entries[i])
			}
		}
		entries = failed

		if attempt == maxRetries-1 {
			return fmt.Errorf("eventbridge: %d events failed after %d attempts", len(entries), maxRetries)
		}
	}
	return nil
}
