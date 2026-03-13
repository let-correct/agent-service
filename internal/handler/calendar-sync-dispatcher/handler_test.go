package calendardispatcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockCalendarDispatcher struct {
	mock.Mock
}

func (m *mockCalendarDispatcher) Handle(ctx context.Context) error {
	return m.Called(ctx).Error(0)
}

func newTestHandler(dispatcher *mockCalendarDispatcher) *Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(logger, dispatcher)
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mockCalendarDispatcher)
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "successful dispatch returns no error",
			setupMock: func(m *mockCalendarDispatcher) {
				m.On("Handle", mock.Anything).Return(nil)
			},
		},
		{
			name: "dispatcher error is propagated",
			setupMock: func(m *mockCalendarDispatcher) {
				m.On("Handle", mock.Anything).Return(errors.New("dispatch failed"))
			},
			wantErr:       true,
			wantErrSubstr: "dispatch failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockCalendarDispatcher{}
			tt.setupMock(m)

			h := newTestHandler(m)
			err := h.Handle(context.Background(), nil)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}
