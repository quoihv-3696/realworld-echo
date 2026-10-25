package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/config"
)

var keys = []string{"PORT", "DATABASE_URL", "JWT_SECRET", "REDIS_URL", "CACHE_TTL"}

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "")
		os.Unsetenv(k)
	}
}

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_DefaultsWithoutEnvFile(t *testing.T) {
	clearEnv(t)
	cfg, err := config.Load(filepath.Join(t.TempDir(), "khong-co.env"))
	if err != nil {
		t.Fatalf("Load khi không có file .env: %v, muốn không lỗi (file .env là tùy chọn)", err)
	}
	if cfg.Port != "8080" || !strings.Contains(cfg.DatabaseURL, "localhost:5433/realworld") || cfg.JWTSecret == "" {
		t.Errorf("giá trị mặc định sai: %+v", cfg)
	}
	if cfg.RedisURL != "" {
		t.Errorf("RedisURL mặc định = %q, muốn rỗng (không cache)", cfg.RedisURL)
	}
	if cfg.CacheTTL != time.Minute {
		t.Errorf("CacheTTL mặc định = %v, muốn 1m0s", cfg.CacheTTL)
	}
}

func TestLoad_ReadsEnvFile(t *testing.T) {
	clearEnv(t)
	p := writeEnv(t, "# ghi chú\nPORT=9090\nREDIS_URL=redis://localhost:6380/0\nCACHE_TTL=30s\n")
	cfg, err := config.Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9090" || cfg.RedisURL != "redis://localhost:6380/0" || cfg.CacheTTL != 30*time.Second {
		t.Errorf("không đọc đúng .env: %+v", cfg)
	}
}

func TestLoad_RealEnvWinsOverFile(t *testing.T) {
	clearEnv(t)
	t.Setenv("PORT", "7070")
	p := writeEnv(t, "PORT=9090\n")
	cfg, err := config.Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "7070" {
		t.Errorf("Port = %q, muốn 7070: biến môi trường thật phải thắng .env (godotenv.Load, không phải Overload)", cfg.Port)
	}
}

func TestLoad_InvalidCacheTTL(t *testing.T) {
	for _, bad := range []string{"abc", "-5s", "0s"} {
		t.Run(bad, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("CACHE_TTL", bad)
			if _, err := config.Load(filepath.Join(t.TempDir(), "khong-co.env")); err == nil {
				t.Errorf("CACHE_TTL=%q: muốn lỗi", bad)
			}
		})
	}
}

func TestLoad_BrokenEnvFile(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	if _, err := config.Load(dir); err == nil {
		t.Error("Load một thư mục thay vì file: muốn lỗi (chỉ bỏ qua lỗi \"file không tồn tại\")")
	}
}
