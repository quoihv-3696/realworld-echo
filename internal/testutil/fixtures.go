package testutil

import (
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

var BaseTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func CreateUser(t *testing.T, db *gorm.DB, username string, bio *string) models.User {
	t.Helper()
	u := models.User{Username: username, Email: username + "@test.local", Password: "x", Bio: bio}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("tạo user %s: %v", username, err)
	}
	return u
}

func CreateArticle(t *testing.T, db *gorm.DB, author models.User, slug string, createdAt time.Time, tags ...string) models.Article {
	t.Helper()
	a := models.Article{
		Slug:        slug,
		Title:       "Title " + slug,
		Description: "Desc " + slug,
		Body:        "Body " + slug,
		AuthorID:    author.ID,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	}
	for _, name := range tags {
		tag := models.Tag{Name: name}
		if err := db.Where(models.Tag{Name: name}).FirstOrCreate(&tag).Error; err != nil {
			t.Fatalf("tạo tag %s: %v", name, err)
		}
		a.Tags = append(a.Tags, tag)
	}
	if err := db.Create(&a).Error; err != nil {
		t.Fatalf("tạo article %s: %v", slug, err)
	}
	return a
}

func CreateUserWithPassword(t *testing.T, db *gorm.DB, username, password string) models.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	u := models.User{Username: username, Email: username + "@test.local", Password: string(hash)}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("tạo user %s: %v", username, err)
	}
	return u
}

func StrPtr(s string) *string { return &s }

func CreateComment(t *testing.T, db *gorm.DB, article models.Article, author models.User, body string) models.Comment {
	t.Helper()
	c := models.Comment{Body: body, ArticleID: article.ID, AuthorID: author.ID}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("tạo comment: %v", err)
	}
	return c
}
