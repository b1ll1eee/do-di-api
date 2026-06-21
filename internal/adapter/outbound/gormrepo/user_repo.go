package gormrepo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/outbound/gormrepo/model"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/outbound"
	"gorm.io/gorm"
)

// userRepo implements outbound.UserRepository using GORM.
type userRepo struct {
	db *gorm.DB
}

// NewUserRepo constructs a GORM-backed UserRepository.
func NewUserRepo(db *gorm.DB) outbound.UserRepository {
	return &userRepo{db: db}
}

func (r *userRepo) Save(ctx context.Context, user *domain.User) error {
	m := model.UserFromDomain(user)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("gormUserRepo.Save: %w", err)
	}
	return nil
}

func (r *userRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var m model.User
	err := r.db.WithContext(ctx).First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("gormUserRepo.FindByID: %w", err)
	}
	return m.ToDomain(), nil
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m model.User
	err := r.db.WithContext(ctx).First(&m, "email = ?", email).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("gormUserRepo.FindByEmail: %w", err)
	}
	return m.ToDomain(), nil
}
