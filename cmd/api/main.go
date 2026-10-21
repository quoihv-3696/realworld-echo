package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm/logger"

	"github.com/quoihv-3696/realworld-echo/internal/database"
	"github.com/quoihv-3696/realworld-echo/internal/server"
)

func main() {
	dsn := getenv("DATABASE_URL", "postgres://realworld:realworld@localhost:5433/realworld?sslmode=disable")
	port := getenv("PORT", "8080")

	db, err := database.Open(dsn, logger.Info)
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

	e := server.New(db)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	sc := echo.StartConfig{Address: ":" + port, GracefulTimeout: 10 * time.Second}
	if err := sc.Start(ctx, e); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
