package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

func favoritesOf(pairs ...[2]uint) *fakeFavoriteRepo {
	f := &fakeFavoriteRepo{pairs: map[[2]uint]bool{}}
	for _, p := range pairs {
		f.pairs[p] = true
	}
	return f
}

func TestArticleService_ListFillsFavorites(t *testing.T) {
	favs := favoritesOf([2]uint{minhID, 1}, [2]uint{annaID, 1}, [2]uint{annaID, 2})
	res, err := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, favs, nil).List(context.Background(), minhID, services.ArticleQuery{Limit: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	type state struct {
		favorited bool
		count     int64
	}
	got := map[string]state{}
	for _, a := range res.Articles {
		got[a.Slug] = state{a.Favorited, a.FavoritesCount}
	}
	if got["cua-jake"] != (state{true, 2}) || got["cua-anna"] != (state{false, 1}) {
		t.Errorf("minh xem danh sách: %+v, muốn cua-jake {true 2}, cua-anna {false 1}", got)
	}
	if favs.countCalls != 1 || favs.amongCalls != 1 {
		t.Errorf("gọi CountByArticle %d lần, FavoritedAmong %d lần cho 2 bài; muốn mỗi hàm đúng 1 lần (gom ID bài, tránh N+1)", favs.countCalls, favs.amongCalls)
	}
}

func TestArticleService_ListGuestSkipsFavoritedQuery(t *testing.T) {
	favs := favoritesOf([2]uint{minhID, 1})
	res, err := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, favs, nil).List(context.Background(), 0, services.ArticleQuery{Limit: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, a := range res.Articles {
		if a.Favorited {
			t.Errorf("khách (viewerID 0) mà favorited = true ở bài %s", a.Slug)
		}
		if a.Slug == "cua-jake" && a.FavoritesCount != 1 {
			t.Errorf("khách xem cua-jake: favoritesCount = %d, muốn 1 (số lượt không phụ thuộc người xem)", a.FavoritesCount)
		}
	}
	if favs.amongCalls != 0 {
		t.Errorf("khách mà vẫn gọi FavoritedAmong %d lần, muốn 0", favs.amongCalls)
	}
	if favs.countCalls != 1 {
		t.Errorf("khách: gọi CountByArticle %d lần, muốn 1 (khách vẫn cần favoritesCount)", favs.countCalls)
	}
}

func TestArticleService_ListPassesFavoritedFilter(t *testing.T) {
	repo := &fakeArticleRepo{}
	svc := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{}, nil)
	if _, err := svc.List(context.Background(), 0, services.ArticleQuery{Favorited: "jake", Tag: "go", Limit: 20}); err != nil {
		t.Fatalf("List: %v", err)
	}
	want := repository.ArticleFilter{FavoritedBy: "jake", Tag: "go", Limit: 20}
	if repo.gotFilter != want {
		t.Errorf("filter gửi repo = %+v, muốn %+v", repo.gotFilter, want)
	}
	if _, err := svc.Feed(context.Background(), minhID, services.ArticleQuery{Favorited: "jake", Limit: 20}); err != nil {
		t.Fatalf("Feed: %v", err)
	}
	if repo.gotFilter.FavoritedBy != "" {
		t.Errorf("feed gửi FavoritedBy = %q, muốn rỗng (feed chỉ nhận limit/offset)", repo.gotFilter.FavoritedBy)
	}
}

func TestArticleService_GetFillsFavorites(t *testing.T) {
	favs := favoritesOf([2]uint{minhID, 1}, [2]uint{annaID, 1})
	svc := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, favs, nil)
	res, err := svc.Get(context.Background(), minhID, "cua-jake")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !res.Article.Favorited || res.Article.FavoritesCount != 2 {
		t.Errorf("minh xem cua-jake: favorited %v, count %d; muốn true, 2", res.Article.Favorited, res.Article.FavoritesCount)
	}
	res, err = svc.Get(context.Background(), annaID, "cua-anna")
	if err != nil || res.Article.Favorited || res.Article.FavoritesCount != 0 {
		t.Errorf("anna xem cua-anna: favorited %v, count %d, err %v; muốn false, 0", res.Article.Favorited, res.Article.FavoritesCount, err)
	}
}

func TestArticleService_FavoriteUnfavorite(t *testing.T) {
	favs := favoritesOf([2]uint{annaID, 1})
	svc := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, favs, nil)
	ctx := context.Background()

	res, err := svc.Favorite(ctx, minhID, "cua-jake")
	if err != nil {
		t.Fatalf("Favorite: %v", err)
	}
	if !favs.pairs[[2]uint{minhID, 1}] {
		t.Error("Favorite không gọi repo.Favorite(minh, bài 1)")
	}
	if res.Article.Slug != "cua-jake" || !res.Article.Favorited || res.Article.FavoritesCount != 2 {
		t.Errorf("sau Favorite: slug %q, favorited %v, count %d; muốn cua-jake, true, 2 (đọc số thật từ repo: anna cũng đã thích)",
			res.Article.Slug, res.Article.Favorited, res.Article.FavoritesCount)
	}
	if res.Article.Author.Username != "jake" {
		t.Errorf("response phải là bài đầy đủ, có tác giả jake: %+v", res.Article)
	}

	res, err = svc.Unfavorite(ctx, minhID, "cua-jake")
	if err != nil {
		t.Fatalf("Unfavorite: %v", err)
	}
	if favs.pairs[[2]uint{minhID, 1}] {
		t.Error("Unfavorite không gọi repo.Unfavorite(minh, bài 1)")
	}
	if res.Article.Favorited || res.Article.FavoritesCount != 1 {
		t.Errorf("sau Unfavorite: favorited %v, count %d; muốn false, 1", res.Article.Favorited, res.Article.FavoritesCount)
	}
}

func TestArticleService_FavoriteErrors(t *testing.T) {
	favs := &fakeFavoriteRepo{}
	svc := services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, favs, nil)
	ctx := context.Background()

	_, err := svc.Favorite(ctx, minhID, "khong-co")
	wantAppErr(t, err, http.StatusNotFound, "article", "not found")
	_, err = svc.Unfavorite(ctx, minhID, "khong-co")
	wantAppErr(t, err, http.StatusNotFound, "article", "not found")
	if len(favs.pairs) != 0 {
		t.Errorf("slug không tồn tại mà vẫn ghi favorite: %v", favs.pairs)
	}

	boom := errors.New("db down")
	svc = services.NewArticleService(twoAuthorRepo(), &fakeFollowRepo{}, &fakeFavoriteRepo{err: boom}, nil)
	if _, err := svc.Favorite(ctx, minhID, "cua-jake"); !errors.Is(err, boom) {
		t.Errorf("Favorite: err = %v, muốn trả nguyên lỗi của FavoriteRepository", err)
	}
	if _, err := svc.Unfavorite(ctx, minhID, "cua-jake"); !errors.Is(err, boom) {
		t.Errorf("Unfavorite: err = %v, muốn trả nguyên lỗi của FavoriteRepository", err)
	}
	if _, err := svc.List(ctx, 0, services.ArticleQuery{Limit: 20}); !errors.Is(err, boom) {
		t.Errorf("List: err = %v, muốn trả nguyên lỗi của CountByArticle", err)
	}
}
