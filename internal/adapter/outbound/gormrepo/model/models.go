// Package model contains GORM persistence models for the outbound adapter.
// These types live exclusively in the adapter layer and must NEVER be imported
// by core/domain or core/service. They are converted to/from domain types at
// the adapter boundary.
package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
)

// User is the GORM persistence model that maps to the "users" table.
type User struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey"`
	Email        string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash string    `gorm:"type:text;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string { return "users" }

// ToDomain converts a GORM User model into a core domain.User.
func (m *User) ToDomain() *domain.User {
	return &domain.User{
		ID:           m.ID,
		Email:        m.Email,
		PasswordHash: m.PasswordHash,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// UserFromDomain converts a core domain.User into a GORM User persistence model.
func UserFromDomain(u *domain.User) *User {
	return &User{
		ID:           u.ID,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
}

// Flowdo is the GORM persistence model that maps to the "flowdos" table.
type Flowdo struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID      uuid.UUID  `gorm:"type:uuid;index;not null"`
	Title       string     `gorm:"type:varchar(255);not null"`
	Description string     `gorm:"type:text"`
	Status      string     `gorm:"type:flowdo_status;not null;default:'pending'"`
	Position    int        `gorm:"not null;default:0"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time `gorm:"index"`
}

func (Flowdo) TableName() string { return "flowdos" }

// ToDomain converts a GORM Flowdo model into a core domain.Flowdo.
func (m *Flowdo) ToDomain() *domain.Flowdo {
	return &domain.Flowdo{
		ID:          m.ID,
		UserID:      m.UserID,
		Title:       m.Title,
		Description: m.Description,
		Status:      domain.Status(m.Status),
		Position:    m.Position,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
		DeletedAt:   m.DeletedAt,
	}
}

// FlowdoFromDomain converts a core domain.Flowdo into a GORM Flowdo persistence model.
func FlowdoFromDomain(t *domain.Flowdo) *Flowdo {
	return &Flowdo{
		ID:          t.ID,
		UserID:      t.UserID,
		Title:       t.Title,
		Description: t.Description,
		Status:      string(t.Status),
		Position:    t.Position,
		CreatedAt:   t.CreatedAt,
		UpdatedAt:   t.UpdatedAt,
		DeletedAt:   t.DeletedAt,
	}
}
