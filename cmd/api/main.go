package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm/logger"

	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/cache"
	"github.com/quoihv-3696/realworld-echo/internal/config"
	"github.com/quoihv-3696/realworld-echo/internal/database"
	"github.com/quoihv-3696/realworld-echo/internal/server"
)

const redisTimeout = 100 * time.Millisecond

func main() {
	cfg, err := config.Load(".env")
	if err != nil {
		log.Fatalf("cấu hình: %v", err)
	}

	db, err := database.Open(cfg.DatabaseURL, logger.Info)
	if err != nil {
		log.Fatalf("kết nối PostgreSQL: %v (đã chạy `docker compose up -d` chưa?)", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}
	seeded, err := database.Seed(context.Background(), db)
	if err != nil {
		log.Fatalf("seed: %v", err)
	}
	if seeded {
		log.Printf("đã thêm dữ liệu mẫu (mật khẩu mọi user: %s)", database.SeedPassword)
	}

	if cfg.JWTSecret == config.DevJWTSecret {
		log.Printf("CẢNH BÁO: JWT_SECRET chưa đặt, đang dùng khóa dev công khai; ai cũng giả được token. Chỉ dùng trên máy dev")
	}
	jwt, err := auth.NewJWT(cfg.JWTSecret, auth.DefaultTokenTTL)
	if err != nil {
		log.Fatalf("JWT_SECRET: %v", err)
	}

	store, closeCache := openCache(cfg)
	defer closeCache()

	e := server.New(db, jwt, store)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{Address: ":" + cfg.Port, GracefulTimeout: 10 * time.Second}
	if err := sc.Start(ctx, e); err != nil {
		log.Fatal(err)
	}
}

func openCache(cfg config.Config) (*cache.Store, func()) {
	if cfg.RedisURL == "" {
		log.Printf("REDIS_URL trống: chạy không cache")
		return nil, func() {}
	}
	r, err := cache.NewRedis(cfg.RedisURL, redisTimeout)
	if err != nil {
		log.Fatalf("REDIS_URL sai định dạng (không in giá trị vì có thể chứa mật khẩu)")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := r.Ping(ctx); err != nil {
		log.Printf("Redis chưa chạy (%v): app vẫn chạy, đọc thẳng DB tới khi Redis lên", err)
	} else {
		log.Printf("cache Redis bật, TTL %v", cfg.CacheTTL)
	}
	return cache.New(r, cfg.CacheTTL, slog.Default()), func() { _ = r.Close() }
}
