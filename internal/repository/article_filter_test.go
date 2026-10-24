package repository_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

type filterFixture struct {
	db               *gorm.DB
	repo             repository.ArticleRepository
	jake, anna, minh models.User
}

func newFilterFixture(t *testing.T) filterFixture {
	t.Helper()
	db := testutil.DB(t)
	f := filterFixture{
		db:   db,
		repo: repository.NewArticleRepository(db),
		jake: testutil.CreateUser(t, db, "jake", nil),
		anna: testutil.CreateUser(t, db, "anna", nil),
		minh: testutil.CreateUser(t, db, "minh", nil),
	}
	at := func(h int) time.Time { return testutil.BaseTime.Add(time.Duration(h) * time.Hour) }
	testutil.CreateArticle(t, db, f.jake, "a1", at(1), "go", "echo")
	testutil.CreateArticle(t, db, f.jake, "a2", at(2), "go")
	testutil.CreateArticle(t, db, f.anna, "a3", at(3), "echo")
	testutil.CreateArticle(t, db, f.anna, "a4", at(4))
	return f
}

func (f filterFixture) list(t *testing.T, filter repository.ArticleFilter) ([]string, int64) {
	t.Helper()
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	got, total, err := f.repo.List(context.Background(), filter)
	if err != nil {
		t.Fatalf("List(%+v): %v", filter, err)
	}
	slugs := make([]string, 0, len(got))
	for _, a := range got {
		slugs = append(slugs, a.Slug)
	}
	return slugs, total
}

func TestArticleRepository_FilterByTag(t *testing.T) {
	f := newFilterFixture(t)
	cases := []struct {
		tag   string
		slugs []string
	}{
		{"go", []string{"a2", "a1"}},
		{"echo", []string{"a3", "a1"}},
		{"khong-co", []string{}},
	}
	for _, tc := range cases {
		slugs, total := f.list(t, repository.ArticleFilter{Tag: tc.tag})
		if !slices.Equal(slugs, tc.slugs) || total != int64(len(tc.slugs)) {
			t.Errorf("tag %q: slugs %v, total %d; muốn %v, total %d (áp cùng bộ lọc cho Count và Find)", tc.tag, slugs, total, tc.slugs, len(tc.slugs))
		}
	}
}

func TestArticleRepository_FilterByAuthorAndCombined(t *testing.T) {
	f := newFilterFixture(t)
	if slugs, total := f.list(t, repository.ArticleFilter{Author: "jake"}); !slices.Equal(slugs, []string{"a2", "a1"}) || total != 2 {
		t.Errorf("author jake: %v, total %d; muốn [a2 a1], 2", slugs, total)
	}
	if slugs, total := f.list(t, repository.ArticleFilter{Author: "jake", Tag: "echo"}); !slices.Equal(slugs, []string{"a1"}) || total != 1 {
		t.Errorf("author jake + tag echo: %v, total %d; muốn [a1], 1 (các điều kiện kết hợp bằng AND)", slugs, total)
	}
	if slugs, total := f.list(t, repository.ArticleFilter{Author: "khong-co"}); len(slugs) != 0 || total != 0 {
		t.Errorf("author không có: %v, total %d; muốn rỗng", slugs, total)
	}
}

func TestArticleRepository_FilterCountsBeforePaging(t *testing.T) {
	f := newFilterFixture(t)
	slugs, total := f.list(t, repository.ArticleFilter{Tag: "go", Limit: 1, Offset: 1})
	if !slices.Equal(slugs, []string{"a1"}) || total != 2 {
		t.Errorf("tag go, limit 1, offset 1: %v, total %d; muốn [a1], total 2 (đếm trước khi cắt trang)", slugs, total)
	}
}

func TestArticleRepository_FilterKeepsAllTags(t *testing.T) {
	f := newFilterFixture(t)
	got, _, err := f.repo.List(context.Background(), repository.ArticleFilter{Tag: "go", Limit: 20})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, a := range got {
		if a.Slug == "a1" && !slices.Equal(tagNames(a), []string{"echo", "go"}) {
			t.Errorf("a1 lọc theo go chỉ còn tag %v, muốn [echo go]: lọc không được làm hụt tag: vẫn Preload(\"Tags\") như ngày 1", tagNames(a))
		}
		if a.Author.Username == "" {
			t.Error("lọc xong mất Author: vẫn Preload(\"Author\")")
		}
	}
}

func TestArticleRepository_FilterFollowedBy(t *testing.T) {
	f := newFilterFixture(t)
	if slugs, total := f.list(t, repository.ArticleFilter{FollowedBy: f.minh.ID}); len(slugs) != 0 || total != 0 {
		t.Errorf("minh chưa follow ai: %v, total %d; muốn rỗng", slugs, total)
	}
	testutil.CreateFollow(t, f.db, f.minh, f.anna)
	if slugs, total := f.list(t, repository.ArticleFilter{FollowedBy: f.minh.ID}); !slices.Equal(slugs, []string{"a4", "a3"}) || total != 2 {
		t.Errorf("minh follow anna: %v, total %d; muốn [a4 a3], 2", slugs, total)
	}
}
