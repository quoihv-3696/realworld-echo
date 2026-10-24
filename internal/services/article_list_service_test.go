package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

const (
	annaID = 8
	minhID = 9
)

func twoAuthorRepo() *fakeArticleRepo {
	return &fakeArticleRepo{
		articles: []models.Article{
			{ID: 1, Slug: "cua-jake", AuthorID: jakeID, Author: models.User{ID: jakeID, Username: "jake"}},
			{ID: 2, Slug: "cua-anna", AuthorID: annaID, Author: models.User{ID: annaID, Username: "anna"}},
		},
		total:   2,
		authors: map[uint]models.User{jakeID: {ID: jakeID, Username: "jake"}, annaID: {ID: annaID, Username: "anna"}},
	}
}

func minhFollowsJake() *fakeFollowRepo {
	return &fakeFollowRepo{pairs: map[[2]uint]bool{{minhID, jakeID}: true}}
}

func TestArticleService_ListPassesFilter(t *testing.T) {
	repo := &fakeArticleRepo{}
	svc := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{})
	if _, err := svc.List(context.Background(), 0, services.ArticleQuery{Tag: "go", Author: "jake", Limit: 5, Offset: 10}); err != nil {
		t.Fatalf("List: %v", err)
	}
	want := repository.ArticleFilter{Tag: "go", Author: "jake", Limit: 5, Offset: 10}
	if repo.gotFilter != want {
		t.Errorf("filter gửi repo = %+v, muốn %+v", repo.gotFilter, want)
	}
}

func TestArticleService_ListValidatesPage(t *testing.T) {
	svc := services.NewArticleService(&fakeArticleRepo{}, &fakeFollowRepo{}, &fakeFavoriteRepo{})
	cases := []struct {
		q          services.ArticleQuery
		field, msg string
	}{
		{services.ArticleQuery{Limit: 0}, "limit", "must be between 1 and 100"},
		{services.ArticleQuery{Limit: 101}, "limit", "must be between 1 and 100"},
		{services.ArticleQuery{Limit: 20, Offset: -1}, "offset", "must not be negative"},
	}
	for _, tc := range cases {
		_, err := svc.List(context.Background(), 0, tc.q)
		wantAppErr(t, err, http.StatusUnprocessableEntity, tc.field, tc.msg)
		_, err = svc.Feed(context.Background(), minhID, tc.q)
		wantAppErr(t, err, http.StatusUnprocessableEntity, tc.field, tc.msg)
	}
	if _, err := svc.List(context.Background(), 0, services.ArticleQuery{Limit: 100}); err != nil {
		t.Errorf("limit 100: %v, muốn hợp lệ", err)
	}
}

func TestArticleService_ListFollowingForViewer(t *testing.T) {
	follows := minhFollowsJake()
	res, err := services.NewArticleService(twoAuthorRepo(), follows, &fakeFavoriteRepo{}).List(context.Background(), minhID, services.ArticleQuery{Limit: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := map[string]bool{}
	for _, a := range res.Articles {
		got[a.Author.Username] = a.Author.Following
	}
	if !got["jake"] || got["anna"] {
		t.Errorf("following theo tác giả = %v, muốn jake true, anna false", got)
	}
	if follows.amongCalls != 1 {
		t.Errorf("gọi FollowingAmong %d lần cho 2 bài, muốn đúng 1 (gom ID tác giả, tránh N+1)", follows.amongCalls)
	}
}

func TestArticleService_ListGuestSkipsFollowQuery(t *testing.T) {
	follows := minhFollowsJake()
	res, err := services.NewArticleService(twoAuthorRepo(), follows, &fakeFavoriteRepo{}).List(context.Background(), 0, services.ArticleQuery{Limit: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, a := range res.Articles {
		if a.Author.Following {
			t.Errorf("khách (viewerID 0) mà following = true ở bài %s", a.Slug)
		}
	}
	if follows.amongCalls != 0 {
		t.Errorf("khách mà vẫn gọi FollowingAmong %d lần, muốn 0", follows.amongCalls)
	}
}

func TestArticleService_Feed(t *testing.T) {
	repo := twoAuthorRepo()
	res, err := services.NewArticleService(repo, minhFollowsJake(), &fakeFavoriteRepo{}).Feed(context.Background(), minhID, services.ArticleQuery{Tag: "bo-qua", Author: "bo-qua", Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("Feed: %v", err)
	}
	want := repository.ArticleFilter{FollowedBy: minhID, Limit: 20}
	if repo.gotFilter != want {
		t.Errorf("filter của feed = %+v, muốn %+v (chỉ FollowedBy + phân trang; feed bỏ qua tag/author)", repo.gotFilter, want)
	}
	if res.ArticlesCount != 2 || len(res.Articles) != 2 {
		t.Errorf("feed = %d bài, articlesCount %d; muốn 2 và 2", len(res.Articles), res.ArticlesCount)
	}
}

func TestArticleService_GetFollowing(t *testing.T) {
	repo := twoAuthorRepo()
	svc := services.NewArticleService(repo, minhFollowsJake(), &fakeFavoriteRepo{})
	res, err := svc.Get(context.Background(), minhID, "cua-jake")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !res.Article.Author.Following {
		t.Error("minh follow jake mà Get bài của jake ra following = false")
	}
	res, err = svc.Get(context.Background(), 0, "cua-jake")
	if err != nil || res.Article.Author.Following {
		t.Errorf("khách xem bài: following = %v, err %v; muốn false", res.Article.Author.Following, err)
	}
}

func TestArticleService_ListPassesFollowErrors(t *testing.T) {
	boom := errors.New("db down")
	_, err := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{err: boom}, &fakeFavoriteRepo{}).List(context.Background(), minhID, services.ArticleQuery{Limit: 20})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, muốn trả nguyên lỗi của FollowRepository", err)
	}
}
