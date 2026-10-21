package repository_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func tagNames(a models.Article) []string {
	var names []string
	for _, t := range a.Tags {
		names = append(names, t.Name)
	}
	slices.Sort(names)
	return names
}

func TestTagRepository_ListNames(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime, "go", "echo")
	testutil.CreateArticle(t, db, jake, "a2", testutil.BaseTime, "gorm", "go")

	got, err := repository.NewTagRepository(db).ListNames(context.Background())
	if err != nil {
		t.Fatalf("ListNames: %v", err)
	}
	if want := []string{"echo", "go", "gorm"}; !slices.Equal(got, want) {
		t.Errorf("ListNames = %v, muốn %v (sắp theo tên)", got, want)
	}
}

func TestArticleRepository_List(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	for i := range 5 {
		testutil.CreateArticle(t, db, jake, fmt.Sprintf("a%d", i), testutil.BaseTime.Add(time.Duration(i)*time.Hour), "go")
	}
	repo := repository.NewArticleRepository(db)

	got, total, err := repo.List(context.Background(), repository.ArticleFilter{Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, muốn 5 (đếm mọi bài, không bị Limit/Offset ảnh hưởng)", total)
	}
	var slugs []string
	for _, a := range got {
		slugs = append(slugs, a.Slug)
	}
	if want := []string{"a3", "a2"}; !slices.Equal(slugs, want) {
		t.Fatalf("slug = %v, muốn %v (mới nhất trước, bỏ qua 1, lấy 2)", slugs, want)
	}
	if got[0].Author.Username != "jake" {
		t.Errorf("Author.Username = %q, muốn \"jake\" (quên Preload(\"Author\")?)", got[0].Author.Username)
	}
	if names := tagNames(got[0]); !slices.Equal(names, []string{"go"}) {
		t.Errorf("Tags = %v, muốn [go] (quên Preload(\"Tags\")?)", names)
	}
}

func TestArticleRepository_FindBySlug(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "dragon", testutil.BaseTime, "training", "dragons")
	repo := repository.NewArticleRepository(db)

	a, err := repo.FindBySlug(context.Background(), "dragon")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if a.Slug != "dragon" || a.Author.Username != "jake" {
		t.Errorf("nhận slug %q, tác giả %q; muốn \"dragon\", \"jake\" (quên Preload(\"Author\")?)", a.Slug, a.Author.Username)
	}
	if names := tagNames(*a); !slices.Equal(names, []string{"dragons", "training"}) {
		t.Errorf("Tags = %v, muốn [dragons training] (quên Preload(\"Tags\")?)", names)
	}
}

func TestArticleRepository_FindBySlugNotFound(t *testing.T) {
	db := testutil.DB(t)
	_, err := repository.NewArticleRepository(db).FindBySlug(context.Background(), "khong-co")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("err = %v, muốn repository.ErrNotFound (đổi gorm.ErrRecordNotFound tại repository)", err)
	}
}

func TestArticleRepository_ListSameCreatedAtNewestIDFirst(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	for i, slug := range []string{"first", "second", "third"} {
		testutil.CreateArticle(t, db, jake, slug, testutil.BaseTime.Add(time.Duration(i)*time.Hour))
	}
	for _, slug := range []string{"third", "first", "second"} {
		if err := db.Model(&models.Article{}).Where("slug = ?", slug).Update("created_at", testutil.BaseTime).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := repository.NewArticleRepository(db)

	var slugs []string
	for offset := range 3 {
		got, _, err := repo.List(context.Background(), repository.ArticleFilter{Limit: 1, Offset: offset})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		for _, a := range got {
			slugs = append(slugs, a.Slug)
		}
	}
	if want := []string{"third", "second", "first"}; !slices.Equal(slugs, want) {
		t.Errorf("phân trang từng bài = %v, muốn %v (cùng created_at thì id lớn trước: ORDER BY created_at DESC, id DESC)", slugs, want)
	}
}

func TestRepositories_UseRequestContext(t *testing.T) {
	db := testutil.DB(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := repository.NewTagRepository(db).ListNames(ctx); err == nil {
		t.Error("ListNames với context đã hủy phải trả lỗi (quên r.db.WithContext(ctx)?)")
	}
	articles := repository.NewArticleRepository(db)
	if _, _, err := articles.List(ctx, repository.ArticleFilter{Limit: 20}); err == nil {
		t.Error("List với context đã hủy phải trả lỗi (quên r.db.WithContext(ctx)?)")
	}
	if _, err := articles.FindBySlug(ctx, "x"); err == nil || errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindBySlug với context đã hủy: err = %v, muốn lỗi context (quên WithContext? hay đổi MỌI lỗi thành ErrNotFound?)", err)
	}
}
