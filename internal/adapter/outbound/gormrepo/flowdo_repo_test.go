package gormrepo_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/b1ll1eee/flowdo-api/internal/adapter/outbound/gormrepo"
	"github.com/b1ll1eee/flowdo-api/internal/core/domain"
	"github.com/b1ll1eee/flowdo-api/internal/core/port/outbound"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupGORMTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	require.NoError(t, err)

	applyGORMSchema(t, db)
	return db
}

func applyGORMSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	schema := `
		CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
		CREATE TABLE IF NOT EXISTS users (
			id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
			email         VARCHAR(255) NOT NULL UNIQUE,
			password_hash TEXT         NOT NULL,
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		);
		DO $$ BEGIN
			CREATE TYPE flowdo_status AS ENUM ('pending', 'in_progress', 'done');
		EXCEPTION WHEN duplicate_object THEN NULL;
		END $$;
		CREATE TABLE IF NOT EXISTS flowdos (
			id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
			user_id     UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			title       VARCHAR(255) NOT NULL,
			description TEXT         NOT NULL DEFAULT '',
			status      flowdo_status  NOT NULL DEFAULT 'pending',
			created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			deleted_at  TIMESTAMPTZ
		);`
	require.NoError(t, db.Exec(schema).Error)
}

func seedGORMUser(t *testing.T, db *gorm.DB) *domain.User {
	t.Helper()
	user := domain.NewUser(fmt.Sprintf("gorm-%s@example.com", uuid.New()), "hash")
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, password_hash, created_at, updated_at) VALUES (?,?,?,?,?)`,
		user.ID, user.Email, user.PasswordHash, user.CreatedAt, user.UpdatedAt,
	).Error)
	return user
}

func TestGORMFlowdoRepo_SaveAndFindByID(t *testing.T) {
	db := setupGORMTestDB(t)
	user := seedGORMUser(t, db)
	repo := gormrepo.NewFlowdoRepo(db)

	flowdo := domain.NewFlowdo(user.ID, "GORM test flowdo", "description")

	require.NoError(t, repo.Save(context.Background(), flowdo))

	found, err := repo.FindByID(context.Background(), flowdo.ID)
	require.NoError(t, err)
	assert.Equal(t, flowdo.ID, found.ID)
	assert.Equal(t, flowdo.Title, found.Title)
	assert.Equal(t, domain.StatusPending, found.Status)
}

func TestGORMFlowdoRepo_FindByID_NotFound(t *testing.T) {
	db := setupGORMTestDB(t)
	repo := gormrepo.NewFlowdoRepo(db)

	_, err := repo.FindByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrFlowdoNotFound)
}

func TestGORMFlowdoRepo_FindByUserID_Pagination(t *testing.T) {
	db := setupGORMTestDB(t)
	user := seedGORMUser(t, db)
	repo := gormrepo.NewFlowdoRepo(db)
	ctx := context.Background()

	for i := range 3 {
		require.NoError(t, repo.Save(ctx, domain.NewFlowdo(user.ID, fmt.Sprintf("task %d", i), "")))
	}

	tests := []struct {
		name      string
		filter    outbound.FlowdoFilter
		wantCount int
		wantTotal int
	}{
		{
			name:      "all items",
			filter:    outbound.FlowdoFilter{UserID: user.ID, Limit: 10},
			wantCount: 3,
			wantTotal: 3,
		},
		{
			name:      "page size 2",
			filter:    outbound.FlowdoFilter{UserID: user.ID, Limit: 2, Offset: 0},
			wantCount: 2,
			wantTotal: 3,
		},
		{
			name:      "second page",
			filter:    outbound.FlowdoFilter{UserID: user.ID, Limit: 2, Offset: 2},
			wantCount: 1,
			wantTotal: 3,
		},
		{
			name:      "filter by in_progress (none yet)",
			filter:    outbound.FlowdoFilter{UserID: user.ID, Status: "in_progress", Limit: 10},
			wantCount: 0,
			wantTotal: 0,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			items, total, err := repo.FindByUserID(ctx, tc.filter)
			require.NoError(t, err)
			assert.Equal(t, tc.wantCount, len(items))
			assert.Equal(t, tc.wantTotal, total)
		})
	}
}

func TestGORMFlowdoRepo_Update(t *testing.T) {
	db := setupGORMTestDB(t)
	user := seedGORMUser(t, db)
	repo := gormrepo.NewFlowdoRepo(db)
	ctx := context.Background()

	flowdo := domain.NewFlowdo(user.ID, "Original", "")
	require.NoError(t, repo.Save(ctx, flowdo))

	require.NoError(t, flowdo.StartProgress())
	flowdo.Title = "Updated via GORM"
	require.NoError(t, repo.Update(ctx, flowdo))

	found, err := repo.FindByID(ctx, flowdo.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated via GORM", found.Title)
	assert.Equal(t, domain.StatusInProgress, found.Status)
}

func TestGORMFlowdoRepo_SoftDelete(t *testing.T) {
	db := setupGORMTestDB(t)
	user := seedGORMUser(t, db)
	repo := gormrepo.NewFlowdoRepo(db)
	ctx := context.Background()

	flowdo := domain.NewFlowdo(user.ID, "To soft-delete", "")
	require.NoError(t, repo.Save(ctx, flowdo))

	require.NoError(t, repo.SoftDelete(ctx, flowdo.ID))

	// Row is still accessible — service layer checks IsDeleted().
	found, err := repo.FindByID(ctx, flowdo.ID)
	require.NoError(t, err)
	assert.True(t, found.IsDeleted())

	// Deleting again returns not found.
	assert.ErrorIs(t, repo.SoftDelete(ctx, flowdo.ID), domain.ErrFlowdoNotFound)
}

func TestGORMFlowdoRepo_SoftDeleted_ExcludedFromList(t *testing.T) {
	db := setupGORMTestDB(t)
	user := seedGORMUser(t, db)
	repo := gormrepo.NewFlowdoRepo(db)
	ctx := context.Background()

	keep := domain.NewFlowdo(user.ID, "Keep", "")
	del := domain.NewFlowdo(user.ID, "Delete me", "")
	require.NoError(t, repo.Save(ctx, keep))
	require.NoError(t, repo.Save(ctx, del))
	require.NoError(t, repo.SoftDelete(ctx, del.ID))

	items, total, err := repo.FindByUserID(ctx, outbound.FlowdoFilter{UserID: user.ID, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, len(items))
	assert.Equal(t, keep.ID, items[0].ID)
}
