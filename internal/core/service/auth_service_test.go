package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/internal/core/service"
	"github.com/b1ll1eee/flowdo-api/mocks"
	"golang.org/x/crypto/bcrypt"
)

const (
	testJWTSecret = "test-secret-key-at-least-32-chars!!"
	testJWTTTL    = 1 * time.Hour
)

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name      string
		input     inbound.RegisterInput
		setupMock func(repo *mocks.UserRepository)
		wantErr   error
	}{
		{
			name:  "successfully registers new user",
			input: inbound.RegisterInput{Email: "alice@example.com", Password: "password123"},
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("FindByEmail", mock.Anything, "alice@example.com").
					Return(nil, domain.ErrUserNotFound)
				repo.On("Save", mock.Anything, mock.AnythingOfType("*domain.User")).Return(nil)
			},
		},
		{
			name:  "returns conflict when email already exists",
			input: inbound.RegisterInput{Email: "existing@example.com", Password: "password123"},
			setupMock: func(repo *mocks.UserRepository) {
				existing := &domain.User{Email: "existing@example.com"}
				repo.On("FindByEmail", mock.Anything, "existing@example.com").
					Return(existing, nil)
			},
			wantErr: domain.ErrEmailAlreadyExists,
		},
		{
			name:  "propagates repo save error",
			input: inbound.RegisterInput{Email: "new@example.com", Password: "password123"},
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("FindByEmail", mock.Anything, "new@example.com").
					Return(nil, domain.ErrUserNotFound)
				repo.On("Save", mock.Anything, mock.AnythingOfType("*domain.User")).Return(assert.AnError)
			},
			wantErr: assert.AnError,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewUserRepository(t)
			tc.setupMock(repo)

			svc := service.NewAuthService(repo, testJWTSecret, testJWTTTL)
			result, err := svc.Register(context.Background(), tc.input)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Token)
				assert.Equal(t, tc.input.Email, result.User.Email)
			}
		})
	}
}

func TestAuthService_Login(t *testing.T) {
	password := "securePass1!"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)

	existingUser := &domain.User{
		Email:        "user@example.com",
		PasswordHash: string(hash),
	}

	tests := []struct {
		name      string
		input     inbound.LoginInput
		setupMock func(repo *mocks.UserRepository)
		wantErr   error
	}{
		{
			name:  "successful login",
			input: inbound.LoginInput{Email: "user@example.com", Password: password},
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("FindByEmail", mock.Anything, "user@example.com").
					Return(existingUser, nil)
			},
		},
		{
			name:  "wrong password returns invalid credentials",
			input: inbound.LoginInput{Email: "user@example.com", Password: "wrongpassword"},
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("FindByEmail", mock.Anything, "user@example.com").
					Return(existingUser, nil)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
		{
			name:  "unknown email returns invalid credentials",
			input: inbound.LoginInput{Email: "ghost@example.com", Password: password},
			setupMock: func(repo *mocks.UserRepository) {
				repo.On("FindByEmail", mock.Anything, "ghost@example.com").
					Return(nil, domain.ErrUserNotFound)
			},
			wantErr: domain.ErrInvalidCredentials,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			repo := mocks.NewUserRepository(t)
			tc.setupMock(repo)

			svc := service.NewAuthService(repo, testJWTSecret, testJWTTTL)
			result, err := svc.Login(context.Background(), tc.input)

			if tc.wantErr != nil {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotEmpty(t, result.Token)
			}
		})
	}
}

func TestAuthService_ValidateToken(t *testing.T) {
	repo := mocks.NewUserRepository(t)
	svc := service.NewAuthService(repo, testJWTSecret, testJWTTTL)

	// Generate a valid token via login.
	password := "securePass1!"
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	user := domain.NewUser("token@example.com", string(hash))

	repo.On("FindByEmail", mock.Anything, "token@example.com").Return(user, nil)
	result, err := svc.Login(context.Background(), inbound.LoginInput{
		Email:    "token@example.com",
		Password: password,
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:  "valid token returns user id",
			token: result.Token,
		},
		{
			name:    "invalid token returns error",
			token:   "not.a.valid.token",
			wantErr: true,
		},
		{
			name:    "empty token returns error",
			token:   "",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			userID, err := svc.ValidateToken(context.Background(), tc.token)
			if tc.wantErr {
				require.Error(t, err)
				assert.Empty(t, userID)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, userID)
			}
		})
	}
}

