package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/cache"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/services"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func newCacheStore(t *testing.T) *cache.Store {
	t.Helper()
	return cache.New(testutil.NewMemoryCache(), time.Minute, testutil.Logger(t))
}

type countingTagRepo struct {
	names []string
	calls int
}

func (r *countingTagRepo) ListNames(context.Context) ([]string, error) {
	r.calls++
	return r.names, nil
}

func TestTagService_ListUsesCache(t *testing.T) {
	repo := &countingTagRepo{names: []string{"echo", "go"}}
	svc := services.NewTagService(repo, newCacheStore(t))
	for i := 0; i < 3; i++ {
		res, err := svc.List(context.Background())
		if err != nil || len(res.Tags) != 2 {
			t.Fatalf("List lần %d: %+v, %v", i+1, res, err)
		}
	}
	if repo.calls != 1 {
		t.Errorf("3 lần List chạm repo %d lần, muốn 1 (lần 2, 3 lấy từ cache)", repo.calls)
	}
}

func TestArticleService_GuestListCachedViewerNot(t *testing.T) {
	repo := twoAuthorRepo()
	svc := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{}, newCacheStore(t))
	ctx := context.Background()
	q := services.ArticleQuery{Limit: 20}

	for i := 0; i < 2; i++ {
		if _, err := svc.List(ctx, 0, q); err != nil {
			t.Fatal(err)
		}
	}
	if repo.listCalls != 1 {
		t.Errorf("khách gọi List 2 lần cùng query: repo.List chạy %d lần, muốn 1", repo.listCalls)
	}

	other := q
	other.Tag = "go"
	if _, err := svc.List(ctx, 0, other); err != nil {
		t.Fatal(err)
	}
	if repo.listCalls != 2 || repo.gotFilter.Tag != "go" {
		t.Errorf("query khác (tag=go) phải là key khác: repo.List %d lần, filter %+v", repo.listCalls, repo.gotFilter)
	}

	before := repo.listCalls
	for i := 0; i < 2; i++ {
		if _, err := svc.List(ctx, minhID, q); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Feed(ctx, minhID, q); err != nil {
			t.Fatal(err)
		}
	}
	if got := repo.listCalls - before; got != 4 {
		t.Errorf("người đăng nhập: 2 List + 2 Feed chạm repo %d lần, muốn 4 (dữ liệu theo người xem không được cache)", got)
	}
}

func TestArticleService_ListValidatesBeforeCache(t *testing.T) {
	svc := services.NewArticleService(&fakeArticleRepo{}, &fakeFollowRepo{}, &fakeFavoriteRepo{}, newCacheStore(t))
	if _, err := svc.List(context.Background(), 0, services.ArticleQuery{Limit: 0}); err == nil {
		t.Error("limit 0 của khách phải 422 như cũ, kể cả khi có cache")
	}
}

func probe(t *testing.T, store *cache.Store) func() bool {
	t.Helper()
	loads := 0
	load := func() (int, error) { loads++; return loads, nil }
	if _, err := store.Remember(context.Background(), "probe", load); err != nil {
		t.Fatal(err)
	}
	return func() bool {
		_, _ = store.Remember(context.Background(), "probe", load)
		return loads == 2
	}
}

func TestArticleService_WritesInvalidateCache(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name  string
		write func(svc *services.ArticleService) error
	}{
		{"Create", func(svc *services.ArticleService) error {
			_, err := svc.Create(ctx, jakeID, newArticle("Bài mới", "go"))
			return err
		}},
		{"Update", func(svc *services.ArticleService) error {
			a := models.Article{ID: 1, Slug: "cua-jake", AuthorID: jakeID}
			title := "Tiêu đề mới"
			_, err := svc.Update(ctx, &a, dto.UpdateArticleRequest{Article: dto.UpdateArticle{Title: dto.Optional[string]{Set: true, Value: &title}}})
			return err
		}},
		{"Delete", func(svc *services.ArticleService) error {
			return svc.Delete(ctx, &models.Article{ID: 1, AuthorID: jakeID})
		}},
		{"Favorite", func(svc *services.ArticleService) error {
			_, err := svc.Favorite(ctx, minhID, "cua-jake")
			return err
		}},
		{"Unfavorite", func(svc *services.ArticleService) error {
			_, err := svc.Unfavorite(ctx, minhID, "cua-jake")
			return err
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newCacheStore(t)
			svc := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, &fakeFavoriteRepo{}, store)
			invalidated := probe(t, store)
			if err := tc.write(svc); err != nil {
				t.Fatalf("%s: %v", tc.name, err)
			}
			if !invalidated() {
				t.Errorf("%s thành công mà không Invalidate: khách sẽ thấy dữ liệu cũ tới hết TTL", tc.name)
			}
		})
	}
}

func TestArticleService_FailedWriteKeepsCache(t *testing.T) {
	ctx := context.Background()
	boom := errors.New("db down")
	cases := []struct {
		name  string
		repo  func() *fakeArticleRepo
		favs  *fakeFavoriteRepo
		write func(svc *services.ArticleService) error
	}{
		{"Create", func() *fakeArticleRepo { r := twoAuthorRepo(); r.createErrs = []error{boom}; return r }, &fakeFavoriteRepo{},
			func(svc *services.ArticleService) error {
				_, err := svc.Create(ctx, jakeID, newArticle("Bài mới"))
				return err
			}},
		{"Update", func() *fakeArticleRepo { r := twoAuthorRepo(); r.err = boom; return r }, &fakeFavoriteRepo{},
			func(svc *services.ArticleService) error {
				title := "Tiêu đề mới"
				_, err := svc.Update(ctx, &models.Article{ID: 1, Slug: "cua-jake", AuthorID: jakeID},
					dto.UpdateArticleRequest{Article: dto.UpdateArticle{Title: dto.Optional[string]{Set: true, Value: &title}}})
				return err
			}},
		{"Delete", func() *fakeArticleRepo { r := twoAuthorRepo(); r.err = boom; return r }, &fakeFavoriteRepo{},
			func(svc *services.ArticleService) error {
				return svc.Delete(ctx, &models.Article{ID: 1, AuthorID: jakeID})
			}},
		{"Favorite", twoAuthorRepo, &fakeFavoriteRepo{err: boom},
			func(svc *services.ArticleService) error {
				_, err := svc.Favorite(ctx, minhID, "cua-jake")
				return err
			}},
		{"Unfavorite", twoAuthorRepo, &fakeFavoriteRepo{err: boom},
			func(svc *services.ArticleService) error {
				_, err := svc.Unfavorite(ctx, minhID, "cua-jake")
				return err
			}},
		{"Favorite slug không có", twoAuthorRepo, &fakeFavoriteRepo{},
			func(svc *services.ArticleService) error {
				_, err := svc.Favorite(ctx, minhID, "khong-co")
				return err
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := newCacheStore(t)
			svc := services.NewArticleService(tc.repo(), &fakeFollowRepo{}, tc.favs, store)
			invalidated := probe(t, store)
			if err := tc.write(svc); err == nil {
				t.Fatalf("%s: muốn lỗi (repo được cài lỗi)", tc.name)
			}
			if invalidated() {
				t.Errorf("%s: ghi thất bại mà vẫn Invalidate. Gọi Invalidate SAU khi repo ghi thành công, không phải trước", tc.name)
			}
		})
	}
}

func TestUserService_UpdateInvalidatesCache(t *testing.T) {
	store := newCacheStore(t)
	svc := services.NewUserService(newFakeUserRepo(), testutil.JWT(t), store)
	register(t, svc, "jake", "password123")
	invalidated := probe(t, store)
	bio := "bio mới"
	if _, err := svc.Update(context.Background(), 1, dto.UpdateUserRequest{User: dto.UpdateUser{Bio: dto.Optional[string]{Set: true, Value: &bio}}}); err != nil {
		t.Fatal(err)
	}
	if !invalidated() {
		t.Error("PUT /api/user đổi bio/username/image mà không Invalidate: danh sách bài đã cache cho khách vẫn hiện tác giả cũ")
	}
}

func TestUserService_FailedUpdateKeepsCache(t *testing.T) {
	store := newCacheStore(t)
	svc := services.NewUserService(newFakeUserRepo(), testutil.JWT(t), store)
	register(t, svc, "jake", "password123")
	register(t, svc, "anna", "password123")
	invalidated := probe(t, store)
	taken := "anna"
	if _, err := svc.Update(context.Background(), 1, dto.UpdateUserRequest{User: dto.UpdateUser{Username: dto.Optional[string]{Set: true, Value: &taken}}}); err == nil {
		t.Fatal("đổi username trùng anna: muốn lỗi 409")
	}
	if invalidated() {
		t.Error("cập nhật user thất bại (trùng username) mà vẫn Invalidate: gọi sau khi lưu thành công")
	}
}
