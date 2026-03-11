package calendarsync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

const maxBatchSize = 10

type sqsClient interface {
	SendMessageBatch(ctx context.Context, params *sqs.SendMessageBatchInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageBatchOutput, error)
}

type Dispatcher struct {
	client   sqsClient
	queueURL string
}

func NewDispatcher(client *sqs.Client, queueURL string) *Dispatcher {
	return &Dispatcher{client: client, queueURL: queueURL}
}

func (d *Dispatcher) Dispatch(ctx context.Context, syncs []*domain.Sync) error {
	for i := 0; i < len(syncs); i += maxBatchSize {
		batch := syncs[i:min(i+maxBatchSize, len(syncs))]
		if err := d.sendBatch(ctx, batch, i); err != nil {
			return err
		}
	}
	return nil
}

func (d *Dispatcher) sendBatch(ctx context.Context, syncs []*domain.Sync, offset int) error {
	entries := make([]types.SendMessageBatchRequestEntry, 0, len(syncs))
	for j, s := range syncs {
		body, err := json.Marshal(s)
		if err != nil {
			return fmt.Errorf("marshal sync message: %w", err)
		}
		entries = append(entries, types.SendMessageBatchRequestEntry{
			Id:          aws.String(fmt.Sprintf("%d", offset+j)),
			MessageBody: aws.String(string(body)),
		})
	}

	out, err := d.client.SendMessageBatch(ctx, &sqs.SendMessageBatchInput{
		QueueUrl: aws.String(d.queueURL),
		Entries:  entries,
	})
	if err != nil {
		return fmt.Errorf("sqs send message batch: %w", err)
	}
	if len(out.Failed) > 0 {
		return fmt.Errorf("sqs: %d messages failed in batch", len(out.Failed))
	}
	return nil
}
