package auth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	apioauth2 "github.com/troysnowden/agent-service/internal/application/oauth2"
	"github.com/troysnowden/agent-service/internal/domain/oauth2"
)

var errInternal = errors.New("something went wrong")

// -- mocks --

type mockAuthInitiator struct {
	result apioauth2.InitiateAuthResult
	err    error
}

func (m *mockAuthInitiator) Handle(_ context.Context, _ apioauth2.InitiateAuthCommand) (apioauth2.InitiateAuthResult, error) {
	return m.result, m.err
}

type mockCodeExchanger struct{ err error }

func (m *mockCodeExchanger) Handle(_ context.Context, _ apioauth2.ExchangeCodeCommand) error {
	return m.err
}

type mockTokenGetter struct {
	result apioauth2.GetTokenResult
	err    error
}

func (m *mockTokenGetter) Handle(_ context.Context, _ apioauth2.GetTokenCommand) (apioauth2.GetTokenResult, error) {
	return m.result, m.err
}

func newLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// -- tests --

func TestHandle_UnknownRoute(t *testing.T) {
	h := New(newLogger(), &mockAuthInitiator{}, &mockCodeExchanger{}, &mockTokenGetter{})
	resp, err := h.Handle(context.Background(), events.APIGatewayV2HTTPRequest{RouteKey: "DELETE /unknown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
}

func TestHandleInitiateAuth(t *testing.T) {
	tests := []struct {
		name       string
		initiator  *mockAuthInitiator
		wantStatus int
		wantURL    string
	}{
		{
			name:       "success returns 200 with url",
			initiator:  &mockAuthInitiator{result: apioauth2.InitiateAuthResult{URL: "https://auth.example.com/authorize"}},
			wantStatus: http.StatusOK,
			wantURL:    "https://auth.example.com/authorize",
		},
		{
			name:       "unsupported provider returns 400",
			initiator:  &mockAuthInitiator{err: apioauth2.ErrUnsupportedProvider},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal error returns 500",
			initiator:  &mockAuthInitiator{err: errInternal},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(newLogger(), tt.initiator, &mockCodeExchanger{}, &mockTokenGetter{})
			resp, err := h.Handle(
				context.Background(),
				events.APIGatewayV2HTTPRequest{
					RouteKey:       "GET /auth/{provider}",
					PathParameters: map[string]string{"provider": "google"},
				},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantURL != "" {
				if got := parseBody(t, resp.Body)["url"]; got != tt.wantURL {
					t.Errorf("url = %q, want %q", got, tt.wantURL)
				}
			}
		})
	}
}

func TestHandleExchangeCode(t *testing.T) {
	validBody := `{"code":"auth-code","state":"state-token"}`

	tests := []struct {
		name       string
		email      string
		body       string
		exchanger  *mockCodeExchanger
		wantStatus int
	}{
		{
			name:       "success returns 200",
			email:      "user@example.com",
			body:       validBody,
			exchanger:  &mockCodeExchanger{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing email claim returns 401",
			email:      "",
			body:       validBody,
			exchanger:  &mockCodeExchanger{},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid json body returns 400",
			email:      "user@example.com",
			body:       "not-json",
			exchanger:  &mockCodeExchanger{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported provider returns 400",
			email:      "user@example.com",
			body:       validBody,
			exchanger:  &mockCodeExchanger{err: apioauth2.ErrUnsupportedProvider},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid state returns 400",
			email:      "user@example.com",
			body:       validBody,
			exchanger:  &mockCodeExchanger{err: oauth2.ErrStateNotFound},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "internal error returns 500",
			email:      "user@example.com",
			body:       validBody,
			exchanger:  &mockCodeExchanger{err: errInternal},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(newLogger(), &mockAuthInitiator{}, tt.exchanger, &mockTokenGetter{})
			resp, err := h.Handle(
				context.Background(),
				events.APIGatewayV2HTTPRequest{
					RouteKey:       "POST /auth/{provider}/callback",
					PathParameters: map[string]string{"provider": "google"},
					RequestContext: events.APIGatewayV2HTTPRequestContext{
						Authorizer: &events.APIGatewayV2HTTPRequestContextAuthorizerDescription{
							JWT: &events.APIGatewayV2HTTPRequestContextAuthorizerJWTDescription{
								Claims: map[string]string{"email": tt.email},
							},
						},
					},
					Body: tt.body,
				},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestHandleGetToken(t *testing.T) {
	validResult := apioauth2.GetTokenResult{
		AccessToken: "access-abc",
		ExpiresAt:   time.Now().Add(time.Hour),
	}

	tests := []struct {
		name       string
		email      string
		getter     *mockTokenGetter
		wantStatus int
		wantToken  string
	}{
		{
			name:       "success returns 200 with access_token",
			email:      "user@example.com",
			getter:     &mockTokenGetter{result: validResult},
			wantStatus: http.StatusOK,
			wantToken:  "access-abc",
		},
		{
			name:       "missing email query param returns 400",
			email:      "",
			getter:     &mockTokenGetter{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "unsupported provider returns 400",
			email:      "user@example.com",
			getter:     &mockTokenGetter{err: apioauth2.ErrUnsupportedProvider},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "token not found returns 404",
			email:      "user@example.com",
			getter:     &mockTokenGetter{err: oauth2.ErrTokenNotFound},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error returns 500",
			email:      "user@example.com",
			getter:     &mockTokenGetter{err: errInternal},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := New(newLogger(), &mockAuthInitiator{}, &mockCodeExchanger{}, tt.getter)
			queryParams := map[string]string{}
			if tt.email != "" {
				queryParams["email"] = tt.email
			}
			resp, err := h.Handle(
				context.Background(),
				events.APIGatewayV2HTTPRequest{
					RouteKey:              "GET /tokens/{provider}",
					PathParameters:        map[string]string{"provider": "google"},
					QueryStringParameters: queryParams,
				},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.wantStatus)
			}
			if tt.wantToken != "" {
				if got := parseBody(t, resp.Body)["access_token"]; got != tt.wantToken {
					t.Errorf("access_token = %q, want %q", got, tt.wantToken)
				}
			}
		})
	}
}

// -- helpers --

func parseBody(t *testing.T, body string) map[string]string {
	t.Helper()
	var m map[string]string
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("failed to parse response body %q: %v", body, err)
	}
	return m
}
