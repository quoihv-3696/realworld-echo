package repository_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestCommentRepository_CreateLoadsAuthor(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	a := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	repo := repository.NewCommentRepository(db)

	c := models.Comment{Body: "hay quá", ArticleID: a.ID, AuthorID: anna.ID}
	if err := repo.Create(context.Background(), &c); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if c.ID == 0 || c.CreatedAt.IsZero() {
		t.Errorf("sau Create: ID = %d, CreatedAt = %v; muốn có cả hai", c.ID, c.CreatedAt)
	}
	if c.Author.Username != "anna" {
		t.Errorf("Author.Username = %q, muốn \"anna\": Create xong phải nạp Author (Preload) để trả response", c.Author.Username)
	}
}

func TestCommentRepository_ListByArticle(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	a := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	b := testutil.CreateArticle(t, db, jake, "b", testutil.BaseTime)
	for _, c := range []models.Comment{
		{Body: "thứ hai", ArticleID: a.ID, AuthorID: jake.ID, CreatedAt: testutil.BaseTime.Add(2 * time.Hour)},
		{Body: "thứ nhất", ArticleID: a.ID, AuthorID: anna.ID, CreatedAt: testutil.BaseTime.Add(time.Hour)},
		{Body: "bài khác", ArticleID: b.ID, AuthorID: jake.ID, CreatedAt: testutil.BaseTime},
	} {
		if err := db.Create(&c).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := repository.NewCommentRepository(db)

	got, err := repo.ListByArticle(context.Background(), a.ID)
	if err != nil {
		t.Fatalf("ListByArticle: %v", err)
	}
	var bodies []string
	for _, c := range got {
		bodies = append(bodies, c.Body)
	}
	if !slices.Equal(bodies, []string{"thứ nhất", "thứ hai"}) {
		t.Fatalf("comments = %v, muốn [thứ nhất thứ hai]: chỉ comment của bài này, cũ trước (Order created_at)", bodies)
	}
	if got[0].Author.Username != "anna" {
		t.Errorf("Author.Username = %q, muốn \"anna\" (quên Preload(\"Author\")?)", got[0].Author.Username)
	}
}

func TestCommentRepository_FindAndDelete(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	a := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	c := testutil.CreateComment(t, db, a, jake, "x")
	repo := repository.NewCommentRepository(db)
	ctx := context.Background()

	got, err := repo.FindByID(ctx, c.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.ArticleID != a.ID || got.AuthorID != jake.ID {
		t.Errorf("FindByID = %+v, muốn ArticleID %d, AuthorID %d", got, a.ID, jake.ID)
	}
	if _, err := repo.FindByID(ctx, 99999); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindByID(99999): err = %v, muốn repository.ErrNotFound", err)
	}
	if err := repo.Delete(ctx, c.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.FindByID(ctx, c.ID); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("sau Delete: err = %v, muốn repository.ErrNotFound", err)
	}
}

func TestCommentRepository_UsesRequestContext(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	a := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	c := testutil.CreateComment(t, db, a, jake, "x")
	repo := repository.NewCommentRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := func(name string, err error) {
		t.Helper()
		if err == nil || errors.Is(err, repository.ErrNotFound) {
			t.Errorf("%s với context đã hủy: err = %v, muốn lỗi context (quên WithContext(ctx)?)", name, err)
		}
	}
	check("Create", repo.Create(ctx, &models.Comment{Body: "y", ArticleID: a.ID, AuthorID: jake.ID}))
	_, err := repo.ListByArticle(ctx, a.ID)
	check("ListByArticle", err)
	_, err = repo.FindByID(ctx, c.ID)
	check("FindByID", err)
	check("Delete", repo.Delete(ctx, c.ID))
}

func TestUserRepository_FindByUsername(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	repo := repository.NewUserRepository(db)

	got, err := repo.FindByUsername(context.Background(), "jake")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if got.ID != jake.ID {
		t.Errorf("FindByUsername = id %d, muốn %d", got.ID, jake.ID)
	}
	if _, err := repo.FindByUsername(context.Background(), "khong-co"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindByUsername không có: err = %v, muốn repository.ErrNotFound", err)
	}
}
