package cache

import (
	"context"
	"errors"
	"time"
)

var ErrMiss = errors.New("cache: không có key")

type Backend interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
}
