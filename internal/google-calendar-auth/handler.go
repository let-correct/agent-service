package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// Handler holds any dependencies your handler needs (DB clients, config, etc.)
type Handler struct {
	logger *slog.Logger
}

// New creates a new Handler with its dependencies.
// Add any clients or config your handler needs as parameters here.
func New(logger *slog.Logger) *Handler {
	return &Handler{
		logger: logger,
	}
}

// Handle is the core Lambda handler function, called by main.go.
func (h *Handler) Handle(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	h.logger.InfoContext(ctx, "received request",
		"method", req.RequestContext.HTTP.Method,
		"path", req.RequestContext.HTTP.Path,
	)

	switch req.RequestContext.HTTP.Method {
	case http.MethodGet:
		return h.handleGet(ctx, req)
	default:
		return response(http.StatusMethodNotAllowed, map[string]string{
			"error": "method not allowed",
		})
	}
}

func (h *Handler) handleGet(_ context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	name := req.QueryStringParameters["name"]
	if name == "" {
		name = "world"
	}

	return response(http.StatusOK, map[string]string{
		"message": "Hello, " + name + "!",
	})
}

// response is a helper to build an APIGatewayV2HTTPResponse with a JSON body.
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
