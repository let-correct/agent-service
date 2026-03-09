package calendarworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-lambda-go/events"
	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

type calendarSyncer interface {
	Handle(ctx context.Context, cmd *domain.Sync) error
}

type Handler struct {
	logger *slog.Logger
	syncer calendarSyncer
}

func New(logger *slog.Logger, syncer calendarSyncer) *Handler {
	return &Handler{logger: logger, syncer: syncer}
}

func (h *Handler) Handle(ctx context.Context, event events.SQSEvent) (events.SQSEventResponse, error) {
	var resp events.SQSEventResponse

	for _, record := range event.Records {
		if err := h.processRecord(ctx, record); err != nil {
			h.logger.ErrorContext(ctx, "failed to process record",
				"messageId", record.MessageId,
				"error", err,
			)
			resp.BatchItemFailures = append(resp.BatchItemFailures, events.SQSBatchItemFailure{
				ItemIdentifier: record.MessageId,
			})
		}
	}

	return resp, nil
}

func (h *Handler) processRecord(ctx context.Context, record events.SQSMessage) error {
	var sync domain.Sync
	if err := json.Unmarshal([]byte(record.Body), &sync); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}
	return h.syncer.Handle(ctx, &sync)
}
