package inbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
)

// CreateFlowdoInput carries validated data for creating a new flowdo.
type CreateFlowdoInput struct {
	UserID      uuid.UUID
	Title       string
	Description string
}

// UpdateFlowdoInput carries validated data for updating an existing flowdo.
type UpdateFlowdoInput struct {
	Title       string
	Description string
	Status      domain.Status
}

// ListFlowdosFilter carries pagination and filtering options.
type ListFlowdosFilter struct {
	UserID uuid.UUID
	Status string
	Limit  int
	Offset int
}

// FlowdoListResult wraps a page of flowdos with total count metadata.
type FlowdoListResult struct {
	Items  []*domain.Flowdo
	Total  int
	Limit  int
	Offset int
}

// FlowdoService is the inbound port (use-case contract) for all Flowdo operations.
// Implementations live in core/service/.
//
//go:generate mockery --name=FlowdoService --output=../../../../mocks --outpkg=mocks
type FlowdoService interface {
	Create(ctx context.Context, input CreateFlowdoInput) (*domain.Flowdo, error)
	GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Flowdo, error)
	List(ctx context.Context, filter ListFlowdosFilter) (*FlowdoListResult, error)
	Update(ctx context.Context, id, userID uuid.UUID, input UpdateFlowdoInput) (*domain.Flowdo, error)
	Delete(ctx context.Context, id, userID uuid.UUID) error
	Reorder(ctx context.Context, userID uuid.UUID, orderedIDs []uuid.UUID) error
}
