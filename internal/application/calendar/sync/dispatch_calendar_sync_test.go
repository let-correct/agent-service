package calendarsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	domain "github.com/troysnowden/agent-service/internal/domain/calendar/sync"
)

type mockSyncStateRepository struct {
	mock.Mock
}

func (m *mockSyncStateRepository) FindAll(ctx context.Context) ([]*domain.Sync, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Sync), args.Error(1)
}

type mockMessageDispatcher struct {
	mock.Mock
}

func (m *mockMessageDispatcher) Dispatch(ctx context.Context, syncs []*domain.Sync) error {
	return m.Called(ctx, syncs).Error(0)
}

func newTestCalendarSyncDispatcher(repo *mockSyncStateRepository, dispatcher *mockMessageDispatcher) *CalendarSyncDispatcher {
	return &CalendarSyncDispatcher{syncStates: repo, dispatcher: dispatcher}
}

func someSync(email string) *domain.Sync {
	return domain.NewSync(email, "primary", "", time.Time{})
}

func TestDispatchCalendarSync_Handle(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func(*mockSyncStateRepository, *mockMessageDispatcher)
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "syncs are found and dispatched",
			setupMock: func(repo *mockSyncStateRepository, dispatcher *mockMessageDispatcher) {
				syncs := []*domain.Sync{someSync("a@example.com"), someSync("b@example.com")}
				repo.On("FindAll", mock.Anything).Return(syncs, nil)
				dispatcher.On("Dispatch", mock.Anything, syncs).Return(nil)
			},
		},
		{
			name: "empty result skips dispatch",
			setupMock: func(repo *mockSyncStateRepository, dispatcher *mockMessageDispatcher) {
				repo.On("FindAll", mock.Anything).Return([]*domain.Sync{}, nil)
			},
		},
		{
			name: "nil result skips dispatch",
			setupMock: func(repo *mockSyncStateRepository, dispatcher *mockMessageDispatcher) {
				repo.On("FindAll", mock.Anything).Return(nil, nil)
			},
		},
		{
			name: "FindAll error is propagated",
			setupMock: func(repo *mockSyncStateRepository, dispatcher *mockMessageDispatcher) {
				repo.On("FindAll", mock.Anything).Return(nil, errors.New("dynamo unavailable"))
			},
			wantErr:       true,
			wantErrSubstr: "find all sync states",
		},
		{
			name: "Dispatch error is propagated",
			setupMock: func(repo *mockSyncStateRepository, dispatcher *mockMessageDispatcher) {
				syncs := []*domain.Sync{someSync("a@example.com")}
				repo.On("FindAll", mock.Anything).Return(syncs, nil)
				dispatcher.On("Dispatch", mock.Anything, syncs).Return(errors.New("sqs unavailable"))
			},
			wantErr:       true,
			wantErrSubstr: "sqs unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSyncStateRepository{}
			dispatcher := &mockMessageDispatcher{}
			tt.setupMock(repo, dispatcher)

			uc := newTestCalendarSyncDispatcher(repo, dispatcher)
			err := uc.Handle(context.Background())

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErrSubstr)
			} else {
				require.NoError(t, err)
			}

			repo.AssertExpectations(t)
			dispatcher.AssertExpectations(t)
		})
	}
}
