package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/troysnowden/let-correct-viewing/internal/application"
	"github.com/troysnowden/let-correct-viewing/internal/domain/oauth2"
)

type Handler struct {
	logger       *slog.Logger
	initiateAuth *application.InitiateAuth
}

func New(logger *slog.Logger, initiateAuth *application.InitiateAuth) *Handler {
	return &Handler{
		logger:       logger,
		initiateAuth: initiateAuth,
	}
}

func (h *Handler) Handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	h.logger.InfoContext(ctx, "received request",
		"method", req.RequestContext.HTTP.Method,
		"path", req.RequestContext.HTTP.Path,
	)

	switch req.RouteKey {
	case "GET /auth/{provider}":
		return h.handleInitiateAuth(ctx, req)
	default:
		return response(http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *Handler) handleInitiateAuth(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	provider := oauth2.ProviderID(req.PathParameters["provider"])

	result, err := h.initiateAuth.Handle(ctx, application.NewInitiateAuthCommand(provider))
	if errors.Is(err, application.ErrUnsupportedProvider) {
		return response(http.StatusBadRequest, map[string]string{"error": "unsupported provider"})
	}
	if err != nil {
		h.logger.ErrorContext(ctx, "initiate auth failed", "error", err)
		return response(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return response(http.StatusOK, map[string]string{"url": result.URL})
}

func response(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(b),
	}, nil
}
