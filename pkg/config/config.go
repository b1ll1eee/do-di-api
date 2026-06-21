package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration sourced from environment variables.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

// AppConfig holds HTTP server settings.
type AppConfig struct {
	Port        string
	Environment string
	// CORSOrigins is a comma-separated list of allowed origins, e.g. "http://localhost:3000,https://app.example.com".
	// Use "*" to allow all origins (development only).
	CORSOrigins []string
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// JWTConfig holds JWT signing settings.
type JWTConfig struct {
	Secret string
	TTL    time.Duration
}

// Load reads configuration from a .env file (if present) then from the environment.
func Load() (*Config, error) {
	// .env is optional — in production the env vars are injected by the runtime.
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Port:        getEnv("APP_PORT", "8080"),
			Environment: getEnv("APP_ENV", "development"),
			CORSOrigins: getEnvSlice("CORS_ORIGINS", []string{"*"}),
		},
		Database: DatabaseConfig{
			DSN:             mustGetEnvOneOf("DATABASE_DSN", "DATABASE_URL"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			Secret: mustGetEnv("JWT_SECRET"),
			TTL:    getEnvDuration("JWT_TTL", 24*time.Hour),
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustGetEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
}

// mustGetEnvOneOf returns the value of the first key that is non-empty.
// Panics with a combined message if none are set. Useful for supporting
// multiple naming conventions (e.g. DATABASE_DSN vs DATABASE_URL on Railway).
func mustGetEnvOneOf(keys ...string) string {
	for _, key := range keys {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	panic(fmt.Sprintf("at least one of the required environment variables must be set: %s", strings.Join(keys, ", ")))
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}

func getEnvSlice(key string, fallback []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	parts := strings.Split(v, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
