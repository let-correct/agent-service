package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	apioauth2 "github.com/troysnowden/agent-service/internal/application/oauth2"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

type authInitiator interface {
	Handle(ctx context.Context, cmd apioauth2.InitiateAuthCommand) (apioauth2.InitiateAuthResult, error)
}

type codeExchanger interface {
	Handle(ctx context.Context, cmd apioauth2.ExchangeCodeCommand) error
}

type tokenGetter interface {
	Handle(ctx context.Context, cmd apioauth2.GetTokenCommand) (apioauth2.GetTokenResult, error)
}

type Handler struct {
	logger       *slog.Logger
	initiateAuth authInitiator
	exchangeCode codeExchanger
	getToken     tokenGetter
}

func New(logger *slog.Logger, initiateAuth authInitiator, exchangeCode codeExchanger, getToken tokenGetter) *Handler {
	return &Handler{
		logger:       logger,
		initiateAuth: initiateAuth,
		exchangeCode: exchangeCode,
		getToken:     getToken,
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
	case "GET /tokens/{provider}":
		return h.handleGetToken(ctx, req)
	default:
		return response(http.StatusNotFound, map[string]string{"error": "not found"})
	}
}

func (h *Handler) handleInitiateAuth(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	provider := oauth2.ProviderID(req.PathParameters["provider"])

	result, err := h.initiateAuth.Handle(ctx, apioauth2.NewInitiateAuthCommand(provider))
	if errors.Is(err, apioauth2.ErrUnsupportedProvider) {
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
}

func (h *Handler) handleExchangeCode(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	provider := oauth2.ProviderID(req.PathParameters["provider"])

	email := req.RequestContext.Authorizer.JWT.Claims["email"]
	if email == "" {
		return response(http.StatusUnauthorized, map[string]string{"error": "missing email claim"})
	}

	var body exchangeCodeRequest
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return response(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	cmd := apioauth2.NewExchangeCodeCommand(email, provider, body.Code, body.State)
	err := h.exchangeCode.Handle(ctx, cmd)
	if errors.Is(err, apioauth2.ErrUnsupportedProvider) {
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

type getTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   string `json:"expires_at"`
}

func (h *Handler) handleGetToken(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	provider := oauth2.ProviderID(req.PathParameters["provider"])

	email := req.QueryStringParameters["email"]
	if email == "" {
		return response(http.StatusBadRequest, map[string]string{"error": "missing email query parameter"})
	}

	result, err := h.getToken.Handle(ctx, apioauth2.NewGetTokenCommand(email, provider))
	if errors.Is(err, apioauth2.ErrUnsupportedProvider) {
		return response(http.StatusBadRequest, map[string]string{"error": "unsupported provider"})
	}
	if errors.Is(err, oauth2.ErrTokenNotFound) {
		return response(http.StatusNotFound, map[string]string{"error": "token not found"})
	}
	if err != nil {
		h.logger.ErrorContext(ctx, "get token failed", "error", err)
		return response(http.StatusInternalServerError, map[string]string{"error": "internal error"})
	}

	return response(http.StatusOK, getTokenResponse{
		AccessToken: result.AccessToken,
		ExpiresAt:   result.ExpiresAt.Format(time.RFC3339),
	})
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
