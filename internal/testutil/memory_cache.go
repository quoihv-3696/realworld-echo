package testutil

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/cache"
)

type MemoryCache struct {
	mu   sync.Mutex
	data map[string][]byte
	err  error
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{data: map[string][]byte{}}
}

func (m *MemoryCache) Fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

func (m *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	v, ok := m.data[key]
	if !ok {
		return nil, cache.ErrMiss
	}
	return append([]byte(nil), v...), nil
}

func (m *MemoryCache) Set(_ context.Context, key string, value []byte, _ time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return m.err
	}
	m.data[key] = append([]byte(nil), value...)
	return nil
}

func (m *MemoryCache) Incr(_ context.Context, key string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return 0, m.err
	}
	n, _ := strconv.ParseInt(string(m.data[key]), 10, 64)
	n++
	m.data[key] = []byte(strconv.FormatInt(n, 10))
	return n, nil
}

func (m *MemoryCache) Keys() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.data)
}
