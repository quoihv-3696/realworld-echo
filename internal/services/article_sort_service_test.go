package services_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

func TestArticleService_ListSort(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", repository.SortNewest},
		{"newest", repository.SortNewest},
		{"oldest", repository.SortOldest},
		{"popular", repository.SortPopular},
	}
	for _, tc := range cases {
		repo := &fakeArticleRepo{}
		svc := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{}, nil)
		if _, err := svc.List(context.Background(), 0, services.ArticleQuery{Sort: tc.in, Limit: 20}); err != nil {
			t.Fatalf("sort %q: %v", tc.in, err)
		}
		if repo.gotFilter.Sort != tc.want {
			t.Errorf("sort %q: filter.Sort = %q, muốn %q (rỗng là mặc định newest)", tc.in, repo.gotFilter.Sort, tc.want)
		}
	}
}

func TestArticleService_ListRejectsUnknownSort(t *testing.T) {
	for _, bad := range []string{"title", "created_at DESC", "Newest", "title; DROP TABLE users"} {
		repo := &fakeArticleRepo{}
		svc := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{}, nil)
		_, err := svc.List(context.Background(), 0, services.ArticleQuery{Sort: bad, Limit: 20})
		wantAppErr(t, err, http.StatusUnprocessableEntity, "sort", "must be one of newest, oldest, popular")
		if repo.listCalls != 0 {
			t.Errorf("sort %q không hợp lệ mà vẫn gọi repo.List", bad)
		}
	}
}

func TestArticleService_FeedIgnoresSort(t *testing.T) {
	repo := &fakeArticleRepo{}
	svc := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{}, nil)
	if _, err := svc.Feed(context.Background(), minhID, services.ArticleQuery{Sort: "khong-hop-le", Limit: 20}); err != nil {
		t.Fatalf("Feed với sort lạ: %v, muốn bỏ qua (feed chỉ nhận limit, offset)", err)
	}
	if repo.gotFilter.Sort != "" {
		t.Errorf("feed gửi Sort = %q, muốn rỗng", repo.gotFilter.Sort)
	}
}
