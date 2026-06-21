package inbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
)

// RegisterInput carries validated registration data.
type RegisterInput struct {
	Email    string
	Password string
}

// LoginInput carries validated login credentials.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult is returned after successful login or registration.
type AuthResult struct {
	User  *domain.User
	Token string
}

// AuthService is the inbound port (use-case contract) for authentication.
//
//go:generate mockery --name=AuthService --output=../../../../mocks --outpkg=mocks
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error)
}
