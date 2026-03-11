package calendardispatcher

import (
	"context"
	"encoding/json"
	"log/slog"
)

type calendarDispatcher interface {
	Handle(ctx context.Context) error
}

type Handler struct {
	logger     *slog.Logger
	dispatcher calendarDispatcher
}

func New(logger *slog.Logger, dispatcher calendarDispatcher) *Handler {
	return &Handler{logger: logger, dispatcher: dispatcher}
}

func (h *Handler) Handle(ctx context.Context, _ json.RawMessage) error {
	if err := h.dispatcher.Handle(ctx); err != nil {
		h.logger.ErrorContext(ctx, "failed to dispatch calendar sync", "error", err)
		return err
	}
	return nil
}
