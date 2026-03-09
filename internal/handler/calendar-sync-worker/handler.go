package calendarworker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-lambda-go/events"
	calendarsync "github.com/troysnowden/agent-service/internal/application/calendar/sync"
)

type calendarSyncer interface {
	Handle(ctx context.Context, cmd calendarsync.SyncCommand) error
}

type Handler struct {
	logger *slog.Logger
	syncer calendarSyncer
}

type syncMessage struct {
	Email        string `json:"email"`
	CalendarID   string `json:"calendar_id"`
	SyncToken    string `json:"sync_token"`
	LastSyncedAt string `json:"last_synced_at"` // RFC3339, empty = zero time (full sync)
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
	var msg syncMessage
	if err := json.Unmarshal([]byte(record.Body), &msg); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	var lastSyncedAt time.Time
	if msg.LastSyncedAt != "" {
		t, err := time.Parse(time.RFC3339, msg.LastSyncedAt)
		if err != nil {
			return fmt.Errorf("parse last_synced_at: %w", err)
		}
		lastSyncedAt = t
	}

	return h.syncer.Handle(ctx, calendarsync.SyncCommand{
		Email:        msg.Email,
		CalendarID:   msg.CalendarID,
		SyncToken:    msg.SyncToken,
		LastSyncedAt: lastSyncedAt,
	})
}
