package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
)

// FlowdoFilter carries query parameters for listing flowdos.
type FlowdoFilter struct {
	UserID uuid.UUID
	Status string
	Limit  int
	Offset int
}

// FlowdoRepository is the outbound port for flowdo persistence.
// Implementations live in adapter/outbound/.
//
//go:generate mockery --name=FlowdoRepository --output=../../../../mocks --outpkg=mocks
type FlowdoRepository interface {
	Save(ctx context.Context, flowdo *domain.Flowdo) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Flowdo, error)
	FindByUserID(ctx context.Context, filter FlowdoFilter) ([]*domain.Flowdo, int, error)
	Update(ctx context.Context, flowdo *domain.Flowdo) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, userID uuid.UUID, orderedIDs []uuid.UUID) error
}
