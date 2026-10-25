package cache

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"
)

const versionKey = "cache:version"

type Store struct {
	backend Backend
	ttl     time.Duration
	log     *slog.Logger
}

func New(backend Backend, ttl time.Duration, log *slog.Logger) *Store {
	if log == nil {
		log = slog.Default()
	}
	return &Store{backend: backend, ttl: ttl, log: log}
}

func (s *Store) Remember[T any](ctx context.Context, name string, load func() (T, error)) (T, error) {
	if s == nil {
		return load()
	}
	key, err := s.key(ctx, name)
	if err != nil {
		s.log.WarnContext(ctx, "cache: không đọc được version, đọc thẳng DB", "err", err)
		return load()
	}
	raw, err := s.backend.Get(ctx, key)
	if err == nil {
		var v T
		if err := json.Unmarshal(raw, &v); err == nil {
			return v, nil
		}
		s.log.WarnContext(ctx, "cache: dữ liệu hỏng, đọc lại DB", "key", key)
	} else if !errors.Is(err, ErrMiss) {
		s.log.WarnContext(ctx, "cache: Get lỗi, đọc thẳng DB", "key", key, "err", err)
	}

	v, err := load()
	if err != nil {
		return v, err
	}
	raw, err = json.Marshal(v)
	if err == nil {
		err = s.backend.Set(ctx, key, raw, s.ttl)
	}
	if err != nil {
		s.log.WarnContext(ctx, "cache: Set lỗi", "key", key, "err", err)
	}
	return v, nil
}

func (s *Store) Invalidate(ctx context.Context) {
	if s == nil {
		return
	}
	if _, err := s.backend.Incr(context.WithoutCancel(ctx), versionKey); err != nil {
		s.log.WarnContext(ctx, "cache: không tăng được version, dữ liệu cũ sống tới hết TTL", "err", err)
	}
}

func (s *Store) key(ctx context.Context, name string) (string, error) {
	v, err := s.backend.Get(ctx, versionKey)
	if errors.Is(err, ErrMiss) {
		return "v0:" + name, nil
	}
	if err != nil {
		return "", err
	}
	return "v" + string(v) + ":" + name, nil
}
