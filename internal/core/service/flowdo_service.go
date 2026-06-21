package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/inbound"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/outbound"
)

// flowdoService implements inbound.FlowdoService.
type flowdoService struct {
	flowdoRepo outbound.FlowdoRepository
}

// NewFlowdoService constructs a FlowdoService with the given repository.
func NewFlowdoService(flowdoRepo outbound.FlowdoRepository) inbound.FlowdoService {
	return &flowdoService{flowdoRepo: flowdoRepo}
}

func (s *flowdoService) Create(ctx context.Context, input inbound.CreateFlowdoInput) (*domain.Flowdo, error) {
	flowdo := domain.NewFlowdo(input.UserID, input.Title, input.Description)

	if err := s.flowdoRepo.Save(ctx, flowdo); err != nil {
		return nil, fmt.Errorf("flowdoService.Create save: %w", err)
	}

	return flowdo, nil
}

func (s *flowdoService) GetByID(ctx context.Context, id, userID uuid.UUID) (*domain.Flowdo, error) {
	flowdo, err := s.flowdoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("flowdoService.GetByID find: %w", err)
	}

	if flowdo.IsDeleted() {
		return nil, domain.ErrFlowdoDeleted
	}

	if !flowdo.BelongsTo(userID) {
		return nil, domain.ErrUnauthorized
	}

	return flowdo, nil
}

func (s *flowdoService) List(ctx context.Context, filter inbound.ListFlowdosFilter) (*inbound.FlowdoListResult, error) {
	items, total, err := s.flowdoRepo.FindByUserID(ctx, outbound.FlowdoFilter{
		UserID: filter.UserID,
		Status: filter.Status,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	})
	if err != nil {
		return nil, fmt.Errorf("flowdoService.List find: %w", err)
	}

	return &inbound.FlowdoListResult{
		Items:  items,
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
	}, nil
}

func (s *flowdoService) Update(ctx context.Context, id, userID uuid.UUID, input inbound.UpdateFlowdoInput) (*domain.Flowdo, error) {
	flowdo, err := s.flowdoRepo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("flowdoService.Update find: %w", err)
	}

	if flowdo.IsDeleted() {
		return nil, domain.ErrFlowdoDeleted
	}

	if !flowdo.BelongsTo(userID) {
		return nil, domain.ErrUnauthorized
	}

	if input.Status != "" {
		flowdo.Status = input.Status
		flowdo.UpdatedAt = time.Now().UTC()
	}

	if input.Title != "" {
		flowdo.Title = input.Title
	}
	if input.Description != "" {
		flowdo.Description = input.Description
	}

	if err := s.flowdoRepo.Update(ctx, flowdo); err != nil {
		return nil, fmt.Errorf("flowdoService.Update persist: %w", err)
	}

	return flowdo, nil
}

func (s *flowdoService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	flowdo, err := s.flowdoRepo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("flowdoService.Delete find: %w", err)
	}

	if flowdo.IsDeleted() {
		return domain.ErrFlowdoDeleted
	}

	if !flowdo.BelongsTo(userID) {
		return domain.ErrUnauthorized
	}

	if err := s.flowdoRepo.SoftDelete(ctx, id); err != nil {
		return fmt.Errorf("flowdoService.Delete soft-delete: %w", err)
	}

	return nil
}

func (s *flowdoService) Reorder(ctx context.Context, userID uuid.UUID, orderedIDs []uuid.UUID) error {
	if err := s.flowdoRepo.Reorder(ctx, userID, orderedIDs); err != nil {
		return fmt.Errorf("flowdoService.Reorder: %w", err)
	}
	return nil
}

