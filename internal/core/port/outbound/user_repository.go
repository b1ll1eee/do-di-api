package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
)

// UserRepository is the outbound port for user persistence.
//
//go:generate mockery --name=UserRepository --output=../../../../mocks --outpkg=mocks
type UserRepository interface {
	Save(ctx context.Context, user *domain.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
}
