package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/cache"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

type page struct {
	Items []string `json:"items"`
	Total int      `json:"total"`
}

type loader struct {
	calls int
	value page
	err   error
}

func (l *loader) load() (page, error) {
	l.calls++
	return l.value, l.err
}

func newStore(t *testing.T) (*cache.Store, *testutil.MemoryCache) {
	t.Helper()
	m := testutil.NewMemoryCache()
	return cache.New(m, time.Minute, testutil.Logger(t)), m
}

func TestStore_RememberMissThenHit(t *testing.T) {
	s, m := newStore(t)
	l := &loader{value: page{Items: []string{"a", "b"}, Total: 2}}
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		got, err := s.Remember(ctx, "x", l.load)
		if err != nil {
			t.Fatalf("lần %d: %v", i, err)
		}
		if got.Total != 2 || len(got.Items) != 2 {
			t.Fatalf("lần %d: %+v, muốn đúng giá trị load trả về", i, got)
		}
	}
	if l.calls != 1 {
		t.Errorf("load chạy %d lần cho 3 lần Remember cùng name, muốn 1 (lần 2, 3 phải lấy từ cache)", l.calls)
	}
	if m.Keys() == 0 {
		t.Error("cache rỗng sau Remember: chưa Set kết quả vào backend")
	}
}

func TestStore_NamesDoNotCollide(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	a := &loader{value: page{Total: 1}}
	b := &loader{value: page{Total: 2}}
	gotA, _ := s.Remember(ctx, "articles?tag=go", a.load)
	gotB, _ := s.Remember(ctx, "articles?tag=echo", b.load)
	if gotA.Total != 1 || gotB.Total != 2 || a.calls != 1 || b.calls != 1 {
		t.Errorf("hai name khác nhau dùng chung key: A=%+v (%d lần), B=%+v (%d lần)", gotA, a.calls, gotB, b.calls)
	}
}

func TestStore_InvalidateForcesReload(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	l := &loader{value: page{Total: 1}}
	_, _ = s.Remember(ctx, "x", l.load)
	s.Invalidate(ctx)
	l.value = page{Total: 2}
	got, err := s.Remember(ctx, "x", l.load)
	if err != nil || got.Total != 2 || l.calls != 2 {
		t.Errorf("sau Invalidate: %+v, err %v, load %d lần; muốn dữ liệu mới (Total 2), load 2 lần", got, err, l.calls)
	}
}

func TestStore_LoadErrorIsNotCached(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	boom := errors.New("db down")
	l := &loader{err: boom}
	if _, err := s.Remember(ctx, "x", l.load); !errors.Is(err, boom) {
		t.Fatalf("err = %v, muốn trả nguyên lỗi của load", err)
	}
	l.err, l.value = nil, page{Total: 7}
	got, err := s.Remember(ctx, "x", l.load)
	if err != nil || got.Total != 7 || l.calls != 2 {
		t.Errorf("sau khi load hết lỗi: %+v, err %v, load %d lần; muốn load lại (lỗi không được cache)", got, err, l.calls)
	}
}

func TestStore_BackendDownFallsBackToLoad(t *testing.T) {
	s, m := newStore(t)
	m.Fail(errors.New("redis: connection refused"))
	ctx := context.Background()
	l := &loader{value: page{Total: 3}}
	for i := 0; i < 2; i++ {
		got, err := s.Remember(ctx, "x", l.load)
		if err != nil || got.Total != 3 {
			t.Fatalf("backend lỗi, lần %d: %+v, err %v; muốn vẫn trả dữ liệu từ load, không lỗi", i+1, got, err)
		}
	}
	if l.calls != 2 {
		t.Errorf("backend lỗi: load %d lần, muốn 2 (mỗi lần đều đọc thẳng DB)", l.calls)
	}
	s.Invalidate(ctx)
}

func TestStore_NilStorePassesThrough(t *testing.T) {
	var s *cache.Store
	l := &loader{value: page{Total: 5}}
	for i := 0; i < 2; i++ {
		got, err := s.Remember(context.Background(), "x", l.load)
		if err != nil || got.Total != 5 {
			t.Fatalf("Store nil: %+v, err %v", got, err)
		}
	}
	s.Invalidate(context.Background())
	if l.calls != 2 {
		t.Errorf("Store nil: load %d lần, muốn 2 (không cache)", l.calls)
	}
}

func TestStore_WriteDuringLoadDoesNotLeaveStaleData(t *testing.T) {
	s, _ := newStore(t)
	ctx := context.Background()
	calls := 0
	staleLoad := func() (page, error) {
		calls++
		if calls == 1 {
			s.Invalidate(ctx)
			return page{Total: 1}, nil
		}
		return page{Total: 2}, nil
	}
	_, _ = s.Remember(ctx, "x", staleLoad)
	got, _ := s.Remember(ctx, "x", staleLoad)
	if got.Total != 2 {
		t.Errorf("có người ghi (Invalidate) trong lúc đang load: lần sau ra %+v, muốn dữ liệu mới Total 2. "+
			"Tính key (đọc version) TRƯỚC khi load, để dữ liệu cũ nằm dưới version cũ", got)
	}
}

type ctxBackend struct{ cache.Backend }

func (b ctxBackend) Incr(ctx context.Context, key string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return b.Backend.Incr(ctx, key)
}

func TestStore_InvalidateSurvivesCanceledRequest(t *testing.T) {
	m := testutil.NewMemoryCache()
	s := cache.New(ctxBackend{m}, time.Minute, testutil.Logger(t))
	l := &loader{value: page{Total: 1}}
	_, _ = s.Remember(context.Background(), "x", l.load)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s.Invalidate(ctx)

	_, _ = s.Remember(context.Background(), "x", l.load)
	if l.calls != 2 {
		t.Error("Invalidate với context đã hủy (client ngắt kết nối sau khi ghi DB) không tăng được version: " +
			"dùng context.WithoutCancel(ctx)")
	}
}
