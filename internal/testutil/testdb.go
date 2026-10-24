package testutil

import (
	"context"
	"os"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/quoihv-3696/realworld-echo/internal/database"
)

const (
	defaultTestDSN = "postgres://realworld:realworld@localhost:5433/realworld_test?sslmode=disable"
	lockKey        = 20261019
)

func DB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := database.Open(dsn, logger.Silent)
	if err != nil {
		t.Fatalf("không kết nối được DB test %s: %v\nĐã chạy `docker compose up -d --wait` chưa? Báo database \"realworld_test\" does not exist thì chạy `docker compose down -v` rồi up lại.", dsn, err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", lockKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.ExecContext(ctx, "SELECT pg_advisory_unlock($1)", lockKey)
		_ = conn.Close()
		_ = sqlDB.Close()
	})

	var name string
	if err := db.Raw("SELECT current_database()").Scan(&name).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(name, "_test") {
		t.Fatalf("DB %q không có đuôi _test: test sẽ xóa sạch dữ liệu nên từ chối chạy. Kiểm tra TEST_DATABASE_URL.", name)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate DB test: %v", err)
	}
	if err := db.Exec("TRUNCATE users, tags, articles, article_tags, comments, follows, favorites RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("dọn DB test: %v", err)
	}
	return db
}
