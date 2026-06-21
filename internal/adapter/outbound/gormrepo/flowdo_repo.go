package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/outbound/gormrepo/model"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/outbound"
	"gorm.io/gorm"
)

// flowdoRepo implements outbound.FlowdoRepository using GORM.
type flowdoRepo struct {
	db *gorm.DB
}

// NewFlowdoRepo constructs a GORM-backed FlowdoRepository.
func NewFlowdoRepo(db *gorm.DB) outbound.FlowdoRepository {
	return &flowdoRepo{db: db}
}

func (r *flowdoRepo) Save(ctx context.Context, flowdo *domain.Flowdo) error {
	m := model.FlowdoFromDomain(flowdo)
	if err := r.db.WithContext(ctx).Create(m).Error; err != nil {
		return fmt.Errorf("gormFlowdoRepo.Save: %w", err)
	}
	return nil
}

func (r *flowdoRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Flowdo, error) {
	var m model.Flowdo
	// Unscoped so we can return soft-deleted records (service layer decides what to do).
	err := r.db.WithContext(ctx).Unscoped().First(&m, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrFlowdoNotFound
		}
		return nil, fmt.Errorf("gormFlowdoRepo.FindByID: %w", err)
	}
	return m.ToDomain(), nil
}

func (r *flowdoRepo) FindByUserID(ctx context.Context, f outbound.FlowdoFilter) ([]*domain.Flowdo, int, error) {
	q := r.db.WithContext(ctx).Model(&model.Flowdo{}).
		Where("user_id = ? AND deleted_at IS NULL", f.UserID)

	if f.Status != "" {
		q = q.Where("status = ?", f.Status)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("gormFlowdoRepo.FindByUserID count: %w", err)
	}

	var rows []model.Flowdo
	err := q.Order("position ASC, created_at DESC").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&rows).Error
	if err != nil {
		return nil, 0, fmt.Errorf("gormFlowdoRepo.FindByUserID find: %w", err)
	}

	flowdos := make([]*domain.Flowdo, 0, len(rows))
	for i := range rows {
		flowdos = append(flowdos, rows[i].ToDomain())
	}

	return flowdos, int(total), nil
}

func (r *flowdoRepo) Update(ctx context.Context, flowdo *domain.Flowdo) error {
	result := r.db.WithContext(ctx).Model(&model.Flowdo{}).
		Where("id = ? AND deleted_at IS NULL", flowdo.ID).
		Updates(map[string]any{
			"title":       flowdo.Title,
			"description": flowdo.Description,
			"status":      string(flowdo.Status),
			"updated_at":  flowdo.UpdatedAt,
		})

	if result.Error != nil {
		return fmt.Errorf("gormFlowdoRepo.Update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrFlowdoNotFound
	}
	return nil
}

func (r *flowdoRepo) Reorder(ctx context.Context, userID uuid.UUID, orderedIDs []uuid.UUID) error {
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("gormFlowdoRepo.Reorder begin tx: %w", tx.Error)
	}

	for i, id := range orderedIDs {
		result := tx.Model(&model.Flowdo{}).
			Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
			Update("position", i)
		if result.Error != nil {
			tx.Rollback()
			return fmt.Errorf("gormFlowdoRepo.Reorder update %s: %w", id, result.Error)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("gormFlowdoRepo.Reorder commit: %w", err)
	}
	return nil
}

func (r *flowdoRepo) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).Model(&model.Flowdo{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", now)

	if result.Error != nil {
		return fmt.Errorf("gormFlowdoRepo.SoftDelete: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrFlowdoNotFound
	}
	return nil
}
