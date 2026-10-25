package server_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestListArticles_Sort(t *testing.T) {
	f := newDay4Fixture(t)
	var a1 models.Article
	f.db.Where("slug = ?", "a1").First(&a1)
	testutil.CreateFavorite(t, f.db, f.minh, a1)

	cases := []struct {
		query string
		want  []string
	}{
		{"", []string{"a3", "a2", "a1"}},
		{"?sort=newest", []string{"a3", "a2", "a1"}},
		{"?sort=oldest", []string{"a1", "a2", "a3"}},
		{"?sort=popular", []string{"a1", "a3", "a2"}},
		{"?sort=oldest&tag=go&limit=1&offset=1", []string{"a2"}},
	}
	for _, tc := range cases {
		res := testutil.Do(t, f.e, http.MethodGet, "/api/articles"+tc.query)
		if got := slugsOf(t, res); !slices.Equal(got, tc.want) {
			t.Errorf("GET /api/articles%s: %v, muốn %v", tc.query, got, tc.want)
		}
	}
	res := testutil.Do(t, f.e, http.MethodGet, "/api/articles?sort=title")
	if res.Status != http.StatusUnprocessableEntity || errorOf(res, "sort") != "must be one of newest, oldest, popular" {
		t.Errorf("sort=title: %d %s, muốn 422 sort", res.Status, res.Body)
	}
}

func TestCache_SortIsPartOfKey(t *testing.T) {
	f := newCachedFixture(t)
	newest := slugsOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles"))
	oldest := slugsOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles?sort=oldest"))
	if !slices.Equal(newest, []string{"a2", "a1"}) || !slices.Equal(oldest, []string{"a1", "a2"}) {
		t.Errorf("khách: mặc định %v, oldest %v; muốn [a2 a1] và [a1 a2] (sort phải nằm trong key cache)", newest, oldest)
	}
	queries := testutil.QueryCounter(t, f.db)
	testutil.Do(t, f.e, http.MethodGet, "/api/articles?sort=newest")
	if n := queries(); n != 0 {
		t.Errorf("?sort=newest sau khi đã cache mặc định: %d câu SQL, muốn 0 (newest và không truyền sort là cùng một key)", n)
	}
}
