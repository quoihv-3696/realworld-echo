package cache_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/cache"
)

const defaultTestRedisURL = "redis://localhost:6380/15"

func testRedis(t *testing.T) (*cache.Redis, string) {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		url = defaultTestRedisURL
	}
	r, err := cache.NewRedis(url, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("NewRedis(%s): %v", url, err)
	}
	t.Cleanup(func() { _ = r.Close() })
	if err := r.Ping(context.Background()); err != nil {
		t.Fatalf("không kết nối được Redis test %s: %v\nĐã chạy `docker compose up -d --wait` chưa?", url, err)
	}
	return r, fmt.Sprintf("test:%s:%d:", t.Name(), time.Now().UnixNano())
}

func TestRedis_GetMissing(t *testing.T) {
	r, prefix := testRedis(t)
	if _, err := r.Get(context.Background(), prefix+"khong-co"); !errors.Is(err, cache.ErrMiss) {
		t.Errorf("Get key không có: err = %v, muốn cache.ErrMiss (đổi redis.Nil thành ErrMiss)", err)
	}
}

func TestRedis_SetGetExpire(t *testing.T) {
	r, prefix := testRedis(t)
	ctx := context.Background()
	key := prefix + "k"
	if err := r.Set(ctx, key, []byte(`{"tags":["go"]}`), time.Second); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := r.Get(ctx, key)
	if err != nil || string(got) != `{"tags":["go"]}` {
		t.Fatalf("Get sau Set: %q, %v", got, err)
	}
	time.Sleep(1100 * time.Millisecond)
	if _, err := r.Get(ctx, key); !errors.Is(err, cache.ErrMiss) {
		t.Errorf("Get sau khi hết TTL 1s: err = %v, muốn ErrMiss (Set có truyền ttl không?)", err)
	}
}

func TestRedis_Incr(t *testing.T) {
	r, prefix := testRedis(t)
	ctx := context.Background()
	key := prefix + "ver"
	if err := r.Set(ctx, key, []byte("0"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	for want := int64(1); want <= 2; want++ {
		n, err := r.Incr(ctx, key)
		if err != nil || n != want {
			t.Fatalf("Incr lần %d = %d, %v; muốn %d", want, n, err, want)
		}
	}
	got, err := r.Get(ctx, key)
	if err != nil || string(got) != "2" {
		t.Errorf("Get sau 2 lần Incr = %q, %v; muốn \"2\"", got, err)
	}
}

func TestRedis_DownFailsFast(t *testing.T) {
	r, err := cache.NewRedis("redis://localhost:6399/0", 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	start := time.Now()
	_, err = r.Get(context.Background(), "k")
	if err == nil || errors.Is(err, cache.ErrMiss) {
		t.Errorf("Redis không chạy: err = %v, muốn lỗi kết nối (không phải ErrMiss)", err)
	}
	if d := time.Since(start); d > 500*time.Millisecond {
		t.Errorf("Redis không chạy mà Get mất %v, muốn trả lỗi ngay", d)
	}
}
