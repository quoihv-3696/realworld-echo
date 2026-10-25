package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultPort        = "8080"
	defaultDatabaseURL = "postgres://realworld:realworld@localhost:5433/realworld?sslmode=disable"
	defaultCacheTTL    = time.Minute
)

const DevJWTSecret = "dev-only-secret-change-me-32-bytes-min"

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	RedisURL    string
	CacheTTL    time.Duration
}

func Load(envFile string) (Config, error) {
	if err := godotenv.Load(envFile); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}, fmt.Errorf("đọc %s: %w", envFile, err)
	}

	ttl := defaultCacheTTL
	if v := os.Getenv("CACHE_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("CACHE_TTL=%q: cần một khoảng thời gian dương, ví dụ 60s", v)
		}
		ttl = d
	}

	return Config{
		Port:        getenv("PORT", defaultPort),
		DatabaseURL: getenv("DATABASE_URL", defaultDatabaseURL),
		JWTSecret:   getenv("JWT_SECRET", DevJWTSecret),
		RedisURL:    os.Getenv("REDIS_URL"),
		CacheTTL:    ttl,
	}, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
