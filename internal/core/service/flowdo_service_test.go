package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/outbound"
	"github.com/b1ll1eee/flowdo-api/internal/core/service"
	"github.com/b1ll1eee/flowdo-api/mocks"
)

func TestFlowdoService_Create(t *testing.T) {
	tests := []struct {
		name      string
		input     inbound.CreateFlowdoInput
		setupMock func(repo *mocks.FlowdoRepository)
		wantErr   bool
	}{
		{
			name: "successfully creates flowdo",
			input: inbound.CreateFlowdoInput{
				UserID:      uuid.New(),
				Title:       "Buy milk",
				Description: "2% fat",
			},
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Flowdo")).Return(nil).Once()
			},
			wantErr: false,
		},
		{
			name: "returns error when repo fails",
			input: inbound.CreateFlowdoInput{
				UserID: uuid.New(),
				Title:  "Buy milk",
			},
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("Save", mock.Anything, mock.AnythingOfType("*domain.Flowdo")).Return(assert.AnError).Once()
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFlowdoRepository(t)
			tc.setupMock(repo)

			svc := service.NewFlowdoService(repo)
			got, err := svc.Create(context.Background(), tc.input)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
				assert.Equal(t, tc.input.Title, got.Title)
				assert.Equal(t, domain.StatusPending, got.Status)
			}
		})
	}
}

func TestFlowdoService_GetByID(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	flowdoID := uuid.New()

	now := time.Now().UTC()
	existingFlowdo := &domain.Flowdo{
		ID:        flowdoID,
		UserID:    ownerID,
		Title:     "Test",
		Status:    domain.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	deletedFlowdo := &domain.Flowdo{
		ID:        flowdoID,
		UserID:    ownerID,
		Title:     "Deleted",
		Status:    domain.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
		DeletedAt: &now,
	}

	tests := []struct {
		name      string
		id        uuid.UUID
		userID    uuid.UUID
		setupMock func(repo *mocks.FlowdoRepository)
		wantErr   error
	}{
		{
			name:   "returns flowdo for owner",
			id:     flowdoID,
			userID: ownerID,
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByID", context.Background(), flowdoID).Return(existingFlowdo, nil)
			},
		},
		{
			name:   "returns unauthorized for non-owner",
			id:     flowdoID,
			userID: otherID,
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByID", context.Background(), flowdoID).Return(existingFlowdo, nil)
			},
			wantErr: domain.ErrUnauthorized,
		},
		{
			name:   "returns not found when repo returns not found",
			id:     flowdoID,
			userID: ownerID,
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByID", context.Background(), flowdoID).Return(nil, domain.ErrFlowdoNotFound)
			},
			wantErr: domain.ErrFlowdoNotFound,
		},
		{
			name:   "returns deleted error for soft-deleted flowdo",
			id:     flowdoID,
			userID: ownerID,
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByID", context.Background(), flowdoID).Return(deletedFlowdo, nil)
			},
			wantErr: domain.ErrFlowdoDeleted,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFlowdoRepository(t)
			tc.setupMock(repo)

			svc := service.NewFlowdoService(repo)
			got, err := svc.GetByID(context.Background(), tc.id, tc.userID)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, got)
			}
		})
	}
}

func TestFlowdoService_Update_StatusTransitions(t *testing.T) {
	ownerID := uuid.New()
	flowdoID := uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name          string
		currentStatus domain.Status
		targetStatus  domain.Status
		wantErr       error
	}{
		{
			name:          "pending → in_progress allowed",
			currentStatus: domain.StatusPending,
			targetStatus:  domain.StatusInProgress,
		},
		{
			name:          "in_progress → done allowed",
			currentStatus: domain.StatusInProgress,
			targetStatus:  domain.StatusDone,
		},
		{
			name:          "pending → done allowed",
			currentStatus: domain.StatusPending,
			targetStatus:  domain.StatusDone,
		},
		{
			name:          "done → pending allowed",
			currentStatus: domain.StatusDone,
			targetStatus:  domain.StatusPending,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			existingFlowdo := &domain.Flowdo{
				ID:        flowdoID,
				UserID:    ownerID,
				Title:     "Test",
				Status:    tc.currentStatus,
				CreatedAt: now,
				UpdatedAt: now,
			}

			repo := mocks.NewFlowdoRepository(t)
			repo.On("FindByID", context.Background(), flowdoID).Return(existingFlowdo, nil)

			if tc.wantErr == nil {
				repo.On("Update", mock.Anything, mock.AnythingOfType("*domain.Flowdo")).Return(nil)
			}

			svc := service.NewFlowdoService(repo)
			got, err := svc.Update(context.Background(), flowdoID, ownerID, inbound.UpdateFlowdoInput{
				Status: tc.targetStatus,
			})

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.targetStatus, got.Status)
			}
		})
	}
}

func TestFlowdoService_Delete(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	flowdoID := uuid.New()
	now := time.Now().UTC()

	existingFlowdo := &domain.Flowdo{
		ID:        flowdoID,
		UserID:    ownerID,
		Title:     "Test",
		Status:    domain.StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}

	tests := []struct {
		name      string
		userID    uuid.UUID
		setupMock func(repo *mocks.FlowdoRepository)
		wantErr   error
	}{
		{
			name:   "owner can delete",
			userID: ownerID,
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByID", context.Background(), flowdoID).Return(existingFlowdo, nil)
				repo.On("SoftDelete", context.Background(), flowdoID).Return(nil)
			},
		},
		{
			name:   "non-owner cannot delete",
			userID: otherID,
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByID", context.Background(), flowdoID).Return(existingFlowdo, nil)
			},
			wantErr: domain.ErrUnauthorized,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFlowdoRepository(t)
			tc.setupMock(repo)

			svc := service.NewFlowdoService(repo)
			err := svc.Delete(context.Background(), flowdoID, tc.userID)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestFlowdoService_List(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	flowdos := []*domain.Flowdo{
		{ID: uuid.New(), UserID: userID, Title: "A", Status: domain.StatusPending, CreatedAt: now, UpdatedAt: now},
		{ID: uuid.New(), UserID: userID, Title: "B", Status: domain.StatusDone, CreatedAt: now, UpdatedAt: now},
	}

	tests := []struct {
		name      string
		filter    inbound.ListFlowdosFilter
		setupMock func(repo *mocks.FlowdoRepository)
		wantTotal int
		wantErr   bool
	}{
		{
			name:   "returns paginated flowdos",
			filter: inbound.ListFlowdosFilter{UserID: userID, Limit: 10, Offset: 0},
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByUserID", context.Background(), outbound.FlowdoFilter{
					UserID: userID, Limit: 10, Offset: 0,
				}).Return(flowdos, 2, nil)
			},
			wantTotal: 2,
		},
		{
			name:   "repo error propagates",
			filter: inbound.ListFlowdosFilter{UserID: userID, Limit: 10, Offset: 0},
			setupMock: func(repo *mocks.FlowdoRepository) {
				repo.On("FindByUserID", context.Background(), outbound.FlowdoFilter{
					UserID: userID, Limit: 10, Offset: 0,
				}).Return(nil, 0, assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewFlowdoRepository(t)
			tc.setupMock(repo)

			svc := service.NewFlowdoService(repo)
			result, err := svc.List(context.Background(), tc.filter)

			if tc.wantErr {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantTotal, result.Total)
			}
		})
	}
}

