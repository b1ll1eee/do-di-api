package gormrepo

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// DBConfig holds connection-pool settings shared with the GORM adapter.
type DBConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Debug           bool
}

// Open creates and configures a *gorm.DB using the provided DSN.
// Call db.Close() (on the underlying *sql.DB) when shutting down.
func Open(cfg DBConfig) (*gorm.DB, error) {
	logLevel := logger.Silent
	if cfg.Debug {
		logLevel = logger.Info
	}

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		DisableForeignKeyConstraintWhenMigrating: true,
		// Disable automatic soft-delete via gorm.Model.DeletedAt — we manage
		// deleted_at ourselves so the field name matches our raw SQL adapter.
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gormrepo.Open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gormrepo.Open underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	return db, nil
}
