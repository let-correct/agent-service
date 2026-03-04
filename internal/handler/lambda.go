package handler

import (
	"context"
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/troysnowden/agent-service/internal/application"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type Handler struct {
	logger       *slog.Logger
	initiateAuth *application.InitiateAuth
	exchangeCode *application.ExchangeCode
}

func New(logger *slog.Logger, initiateAuth *application.InitiateAuth, exchangeCode *application.ExchangeCode) *Handler {
	return &Handler{
		logger:       logger,
		initiateAuth: initiateAuth,
		exchangeCode: exchangeCode,
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
	case "POST /auth/{provider}/callback":
		return h.handleExchangeCode(ctx, req)
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

type exchangeCodeRequest struct {
	Code  string `json:"code"`
	State string `json:"state"`
	Email string `json:"email"`
}

func (h *Handler) handleExchangeCode(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	provider := oauth2.ProviderID(req.PathParameters["provider"])

	var body exchangeCodeRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return response(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	cmd := application.NewExchangeCodeCommand(body.Email, provider, body.Code, body.State)
	err := h.exchangeCode.Handle(ctx, cmd)
	if errors.Is(err, application.ErrUnsupportedProvider) {
		return response(http.StatusBadRequest, map[string]string{"error": "unsupported provider"})
	}
	if errors.Is(err, oauth2.ErrStateNotFound) {
		return response(http.StatusBadRequest, map[string]string{"error": "invalid or expired state"})
	}
	if err != nil {
		h.logger.ErrorContext(ctx, "exchange code failed", "error", err)
		return response(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return response(http.StatusOK, map[string]string{})
}

func response(statusCode int, body any) (events.APIGatewayV2HTTPResponse, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(body); err != nil {
		return events.APIGatewayV2HTTPResponse{StatusCode: http.StatusInternalServerError}, err
	}

	return events.APIGatewayV2HTTPResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: buf.String(),
	}, nil
}
