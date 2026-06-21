package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Status represents the lifecycle state of a Flowdo item.
type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrFlowdoNotFound            = errors.New("flowdo not found")
	ErrFlowdoDeleted             = errors.New("flowdo has been deleted")
	ErrUnauthorized            = errors.New("unauthorized: flowdo belongs to another user")
)

// Flowdo is the core domain entity. It has zero framework dependencies.
type Flowdo struct {
	ID          uuid.UUID  `db:"id"`
	UserID      uuid.UUID  `db:"user_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Status      Status     `db:"status"`
	Position    int        `db:"position"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

// NewFlowdo constructs a new Flowdo with default pending status.
func NewFlowdo(userID uuid.UUID, title, description string) *Flowdo {
	now := time.Now().UTC()
	return &Flowdo{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       title,
		Description: description,
		Status:      StatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// StartProgress transitions the flowdo from pending → in_progress.
// Returns ErrInvalidStatusTransition if the current state does not allow it.
func (t *Flowdo) StartProgress() error {
	if t.Status != StatusPending {
		return ErrInvalidStatusTransition
	}
	t.Status = StatusInProgress
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// MarkDone transitions the flowdo from in_progress → done.
// Returns ErrInvalidStatusTransition if the current state does not allow it.
func (t *Flowdo) MarkDone() error {
	if t.Status != StatusInProgress {
		return ErrInvalidStatusTransition
	}
	t.Status = StatusDone
	t.UpdatedAt = time.Now().UTC()
	return nil
}

// IsDeleted reports whether the flowdo has been soft-deleted.
func (t *Flowdo) IsDeleted() bool {
	return t.DeletedAt != nil
}

// BelongsTo reports whether the flowdo belongs to the given user.
func (t *Flowdo) BelongsTo(userID uuid.UUID) bool {
	return t.UserID == userID
}

// IsValidStatus checks whether a given string is a recognised status value.
func IsValidStatus(s string) bool {
	switch Status(s) {
	case StatusPending, StatusInProgress, StatusDone:
		return true
	}
	return false
}
