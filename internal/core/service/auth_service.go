package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/outbound"
	"golang.org/x/crypto/bcrypt"
)

// jwtClaims holds the standard JWT claims plus our custom user_id.
type jwtClaims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// authService implements inbound.AuthService.
type authService struct {
	userRepo  outbound.UserRepository
	jwtSecret string
	jwtTTL    time.Duration
}

// NewAuthService constructs an AuthService.
func NewAuthService(
	userRepo outbound.UserRepository,
	jwtSecret string,
	jwtTTL time.Duration,
) inbound.AuthService {
	return &authService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

func (s *authService) Register(ctx context.Context, input inbound.RegisterInput) (*inbound.AuthResult, error) {
	existing, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil && err != domain.ErrUserNotFound {
		return nil, fmt.Errorf("authService.Register check email: %w", err)
	}
	if existing != nil {
		return nil, domain.ErrEmailAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("authService.Register hash password: %w", err)
	}

	user := domain.NewUser(input.Email, string(hash))

	if err := s.userRepo.Save(ctx, user); err != nil {
		return nil, fmt.Errorf("authService.Register save user: %w", err)
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("authService.Register generate token: %w", err)
	}

	return &inbound.AuthResult{User: user, Token: token}, nil
}

func (s *authService) Login(ctx context.Context, input inbound.LoginInput) (*inbound.AuthResult, error) {
	user, err := s.userRepo.FindByEmail(ctx, input.Email)
	if err != nil {
		// Normalise not-found into invalid credentials to prevent user enumeration.
		if err == domain.ErrUserNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, fmt.Errorf("authService.Login find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	token, err := s.generateToken(user.ID)
	if err != nil {
		return nil, fmt.Errorf("authService.Login generate token: %w", err)
	}

	return &inbound.AuthResult{User: user, Token: token}, nil
}

func (s *authService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("authService.GetMe find: %w", err)
	}
	return user, nil
}

func (s *authService) ValidateToken(_ context.Context, tokenString string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("authService.ValidateToken parse: %w", err)
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return uuid.Nil, fmt.Errorf("authService.ValidateToken invalid claims")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("authService.ValidateToken parse user_id: %w", err)
	}

	return userID, nil
}

func (s *authService) generateToken(userID uuid.UUID) (string, error) {
	claims := &jwtClaims{
		UserID: userID.String(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}
