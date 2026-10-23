package repository_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func countRows(t *testing.T, db *gorm.DB, table string) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func newArticle(author models.User, slug string) models.Article {
	return models.Article{Slug: slug, Title: "Title " + slug, Description: "Desc", Body: "Body", AuthorID: author.ID}
}

func TestArticleRepository_CreateReusesExistingTags(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "old", testutil.BaseTime, "go")
	repo := repository.NewArticleRepository(db)
	ctx := context.Background()

	a := newArticle(jake, "new")
	if err := repo.Create(ctx, &a, []string{"go", "echo"}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("sau Create, a.ID vẫn là 0: truyền con trỏ a vào Create")
	}
	got, err := repo.FindBySlug(ctx, "new")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if names := tagNames(*got); !slices.Equal(names, []string{"echo", "go"}) {
		t.Errorf("tag của bài = %v, muốn [echo go]", names)
	}
	if got.Author.Username != "jake" {
		t.Errorf("Author.Username = %q, muốn \"jake\"", got.Author.Username)
	}
	if n := countRows(t, db, "tags"); n != 2 {
		t.Errorf("bảng tags có %d dòng, muốn 2: tag \"go\" đã có phải được dùng lại (upsert rồi đọc lại theo tên)", n)
	}
}

func TestArticleRepository_CreateWithoutTags(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	repo := repository.NewArticleRepository(db)

	a := newArticle(jake, "no-tags")
	if err := repo.Create(context.Background(), &a, nil); err != nil {
		t.Fatalf("Create không có tag: %v", err)
	}
	got, err := repo.FindBySlug(context.Background(), "no-tags")
	if err != nil {
		t.Fatalf("FindBySlug: %v", err)
	}
	if len(got.Tags) != 0 {
		t.Errorf("Tags = %v, muốn rỗng", tagNames(*got))
	}
}

func TestArticleRepository_CreateDuplicateSlugRollsBack(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "dup", testutil.BaseTime)
	repo := repository.NewArticleRepository(db)

	a := newArticle(jake, "dup")
	err := repo.Create(context.Background(), &a, []string{"chi-trong-giao-dich-hong"})
	var dup *repository.DuplicateError
	if !errors.As(err, &dup) || dup.Field != "slug" {
		t.Fatalf("slug trùng: err = %v, muốn &repository.DuplicateError{Field: \"slug\"} (bảng tra idx_articles_slug)", err)
	}
	var n int64
	db.Model(&models.Tag{}).Where("name = ?", "chi-trong-giao-dich-hong").Count(&n)
	if n != 0 {
		t.Error("tag tạo trong lần Create thất bại vẫn còn: tạo tag và bài phải nằm trong CÙNG một db.Transaction")
	}
}

func TestArticleRepository_Update(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime, "go", "echo")
	repo := repository.NewArticleRepository(db)
	ctx := context.Background()
	reload := func() *models.Article {
		t.Helper()
		got, err := repo.FindBySlug(ctx, "a")
		if err != nil {
			t.Fatalf("FindBySlug: %v", err)
		}
		return got
	}

	a := reload()
	a.Body = "nội dung mới"
	if err := repo.Update(ctx, a, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got := reload()
	if got.Body != "nội dung mới" {
		t.Errorf("Body = %q, muốn \"nội dung mới\"", got.Body)
	}
	if names := tagNames(*got); !slices.Equal(names, []string{"echo", "go"}) {
		t.Errorf("tagNames == nil mà tag bị đổi thành %v, muốn giữ [echo go]", names)
	}
	if !got.CreatedAt.Equal(testutil.BaseTime) {
		t.Errorf("CreatedAt = %v, muốn giữ nguyên %v", got.CreatedAt, testutil.BaseTime)
	}
	if !got.UpdatedAt.After(testutil.BaseTime) {
		t.Errorf("UpdatedAt = %v, muốn mới hơn %v", got.UpdatedAt, testutil.BaseTime)
	}

	if err := repo.Update(ctx, got, &[]string{"gorm", "go"}); err != nil {
		t.Fatalf("Update tag: %v", err)
	}
	if names := tagNames(*reload()); !slices.Equal(names, []string{"go", "gorm"}) {
		t.Errorf("sau khi thay tag = %v, muốn [go gorm] (Association(\"Tags\").Replace)", names)
	}

	if err := repo.Update(ctx, reload(), &[]string{}); err != nil {
		t.Fatalf("Update tag rỗng: %v", err)
	}
	if names := tagNames(*reload()); len(names) != 0 {
		t.Errorf("tagList [] mà còn tag %v, muốn xóa hết", names)
	}
	if n := countRows(t, db, "tags"); n != 3 {
		t.Errorf("bảng tags có %d dòng, muốn 3: gỡ tag khỏi bài không xóa tag", n)
	}
}

func TestArticleRepository_DeleteCascades(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	a := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime, "go", "echo")
	testutil.CreateComment(t, db, a, jake, "hay")
	testutil.CreateComment(t, db, a, jake, "tuyệt")
	repo := repository.NewArticleRepository(db)

	if err := repo.Delete(context.Background(), a.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindBySlug(context.Background(), "a"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindBySlug sau khi xóa: err = %v, muốn ErrNotFound", err)
	}
	if n := countRows(t, db, "comments"); n != 0 {
		t.Errorf("còn %d comment của bài đã xóa", n)
	}
	if n := countRows(t, db, "article_tags"); n != 0 {
		t.Errorf("còn %d dòng article_tags của bài đã xóa", n)
	}
	if n := countRows(t, db, "tags"); n != 2 {
		t.Errorf("bảng tags còn %d dòng, muốn 2: xóa bài không xóa tag", n)
	}
}

func TestArticleRepository_WritesUseRequestContext(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	existing := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	repo := repository.NewArticleRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	a := newArticle(jake, "b")
	if err := repo.Create(ctx, &a, []string{"go"}); err == nil {
		t.Error("Create với context đã hủy phải trả lỗi (quên WithContext(ctx)?)")
	}
	if err := repo.Update(ctx, &existing, nil); err == nil {
		t.Error("Update với context đã hủy phải trả lỗi")
	}
	if err := repo.Delete(ctx, existing.ID); err == nil {
		t.Error("Delete với context đã hủy phải trả lỗi")
	}
}
