package repository_test

import (
	"context"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestFavoriteRepository_FavoriteUnfavorite(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	a1 := testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime)
	repo := repository.NewFavoriteRepository(db)
	ctx := context.Background()

	count := func() int64 {
		var n int64
		db.Model(&models.Favorite{}).Count(&n)
		return n
	}
	for i := 0; i < 2; i++ {
		if err := repo.Favorite(ctx, jake.ID, a1.ID); err != nil {
			t.Fatalf("Favorite lần %d: %v (favorite lại không được lỗi: ON CONFLICT DO NOTHING)", i+1, err)
		}
	}
	if n := count(); n != 1 {
		t.Errorf("favorite 2 lần ra %d dòng, muốn 1", n)
	}
	var users int64
	db.Model(&models.User{}).Count(&users)
	if users != 1 {
		t.Errorf("bảng users có %d dòng sau Favorite, muốn 1 (thiếu Omit(\"User\", \"Article\")?)", users)
	}
	for i := 0; i < 2; i++ {
		if err := repo.Unfavorite(ctx, jake.ID, a1.ID); err != nil {
			t.Fatalf("Unfavorite lần %d: %v (bỏ khi chưa favorite không được lỗi)", i+1, err)
		}
	}
	if n := count(); n != 0 {
		t.Errorf("sau Unfavorite còn %d dòng, muốn 0", n)
	}
}

func TestFavoriteRepository_CountByArticle(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	a1 := testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime)
	a2 := testutil.CreateArticle(t, db, jake, "a2", testutil.BaseTime)
	a3 := testutil.CreateArticle(t, db, anna, "a3", testutil.BaseTime)
	testutil.CreateFavorite(t, db, jake, a1)
	testutil.CreateFavorite(t, db, anna, a1)
	testutil.CreateFavorite(t, db, anna, a2)
	repo := repository.NewFavoriteRepository(db)

	got, err := repo.CountByArticle(context.Background(), []uint{a1.ID, a2.ID, a3.ID})
	if err != nil {
		t.Fatalf("CountByArticle: %v", err)
	}
	if len(got) != 2 || got[a1.ID] != 2 || got[a2.ID] != 1 {
		t.Errorf("CountByArticle([a1 a2 a3]) = %v, muốn {a1: 2, a2: 1} (a3 không có lượt nào thì không có key). "+
			"Toàn 0: alias trong Select khác tên field của struct nhận Scan", got)
	}
	empty, err := repo.CountByArticle(context.Background(), nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("CountByArticle với danh sách rỗng = %v, %v; muốn map rỗng (khác nil), không lỗi", empty, err)
	}
}

func TestFavoriteRepository_FavoritedAmong(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	a1 := testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime)
	a2 := testutil.CreateArticle(t, db, jake, "a2", testutil.BaseTime)
	a3 := testutil.CreateArticle(t, db, anna, "a3", testutil.BaseTime)
	testutil.CreateFavorite(t, db, anna, a1)
	testutil.CreateFavorite(t, db, anna, a3)
	testutil.CreateFavorite(t, db, jake, a2)
	repo := repository.NewFavoriteRepository(db)

	got, err := repo.FavoritedAmong(context.Background(), anna.ID, []uint{a1.ID, a2.ID, a3.ID})
	if err != nil {
		t.Fatalf("FavoritedAmong: %v", err)
	}
	if len(got) != 2 || !got[a1.ID] || !got[a3.ID] {
		t.Errorf("FavoritedAmong(anna, [a1 a2 a3]) = %v, muốn {a1: true, a3: true}", got)
	}
	empty, err := repo.FavoritedAmong(context.Background(), anna.ID, nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("FavoritedAmong với danh sách rỗng = %v, %v; muốn map rỗng (khác nil), không lỗi", empty, err)
	}
}

func TestFavoriteRepository_UsesRequestContext(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	a1 := testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime)
	repo := repository.NewFavoriteRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s với context đã hủy phải trả lỗi (quên WithContext(ctx)?)", name)
		}
	}
	check("Favorite", repo.Favorite(ctx, jake.ID, a1.ID))
	check("Unfavorite", repo.Unfavorite(ctx, jake.ID, a1.ID))
	_, err := repo.CountByArticle(ctx, []uint{a1.ID})
	check("CountByArticle", err)
	_, err = repo.FavoritedAmong(ctx, jake.ID, []uint{a1.ID})
	check("FavoritedAmong", err)
}

func TestArticleRepository_FilterFavoritedBy(t *testing.T) {
	f := newFilterFixture(t)
	var a1, a3, a4 models.Article
	f.db.Where("slug = ?", "a1").First(&a1)
	f.db.Where("slug = ?", "a3").First(&a3)
	f.db.Where("slug = ?", "a4").First(&a4)
	testutil.CreateFavorite(t, f.db, f.minh, a1)
	testutil.CreateFavorite(t, f.db, f.minh, a3)
	testutil.CreateFavorite(t, f.db, f.jake, a3)
	testutil.CreateFavorite(t, f.db, f.jake, a4)

	cases := []struct {
		name   string
		filter repository.ArticleFilter
		want   []string
		total  int64
	}{
		{"minh thích", repository.ArticleFilter{FavoritedBy: "minh"}, []string{"a3", "a1"}, 2},
		{"jake thích", repository.ArticleFilter{FavoritedBy: "jake"}, []string{"a4", "a3"}, 2},
		{"minh thích + tag go", repository.ArticleFilter{FavoritedBy: "minh", Tag: "go"}, []string{"a1"}, 1},
		{"minh thích + tác giả anna", repository.ArticleFilter{FavoritedBy: "minh", Author: "anna"}, []string{"a3"}, 1},
		{"minh thích, trang 2", repository.ArticleFilter{FavoritedBy: "minh", Limit: 1, Offset: 1}, []string{"a1"}, 2},
		{"không ai thích", repository.ArticleFilter{FavoritedBy: "anna"}, []string{}, 0},
		{"user không tồn tại", repository.ArticleFilter{FavoritedBy: "nobody"}, []string{}, 0},
	}
	for _, tc := range cases {
		got, total := f.list(t, tc.filter)
		if !slices.Equal(got, tc.want) || total != tc.total {
			t.Errorf("%s: %v, total %d; muốn %v, total %d", tc.name, got, total, tc.want, tc.total)
		}
	}
}
