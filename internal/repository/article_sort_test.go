package repository_test

import (
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestArticleRepository_Sort(t *testing.T) {
	f := newFilterFixture(t)
	var a2, a3 models.Article
	f.db.Where("slug = ?", "a2").First(&a2)
	f.db.Where("slug = ?", "a3").First(&a3)
	testutil.CreateFavorite(t, f.db, f.jake, a2)
	testutil.CreateFavorite(t, f.db, f.anna, a2)
	testutil.CreateFavorite(t, f.db, f.minh, a3)

	cases := []struct {
		sort string
		want []string
	}{
		{"", []string{"a4", "a3", "a2", "a1"}},
		{repository.SortNewest, []string{"a4", "a3", "a2", "a1"}},
		{repository.SortOldest, []string{"a1", "a2", "a3", "a4"}},
		{repository.SortPopular, []string{"a2", "a3", "a4", "a1"}},
		{"title ASC", []string{"a4", "a3", "a2", "a1"}},
	}
	for _, tc := range cases {
		got, total := f.list(t, repository.ArticleFilter{Sort: tc.sort})
		if !slices.Equal(got, tc.want) || total != 4 {
			t.Errorf("Sort %q: %v (total %d), muốn %v (total 4)", tc.sort, got, total, tc.want)
		}
	}
}

func TestArticleRepository_SortWithFilterAndPaging(t *testing.T) {
	f := newFilterFixture(t)
	var a1 models.Article
	f.db.Where("slug = ?", "a1").First(&a1)
	testutil.CreateFavorite(t, f.db, f.minh, a1)

	got, total := f.list(t, repository.ArticleFilter{Tag: "go", Sort: repository.SortPopular, Limit: 1})
	if !slices.Equal(got, []string{"a1"}) || total != 2 {
		t.Errorf("tag go, popular, limit 1: %v (total %d), muốn [a1] (total 2: sắp xếp không đổi số đếm)", got, total)
	}
	got, _ = f.list(t, repository.ArticleFilter{Tag: "go", Sort: repository.SortPopular, Limit: 1, Offset: 1})
	if !slices.Equal(got, []string{"a2"}) {
		t.Errorf("tag go, popular, trang 2: %v, muốn [a2]", got)
	}
}
