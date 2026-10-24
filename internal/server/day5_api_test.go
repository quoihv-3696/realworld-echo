package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func favState(a map[string]any) string {
	return fmt.Sprintf("favorited=%v count=%v", a["favorited"], a["favoritesCount"])
}

func TestFavorites_Flow(t *testing.T) {
	f := newDay4Fixture(t)
	minh := testutil.WithToken(testutil.Token(t, f.minh.ID))
	anna := testutil.WithToken(testutil.Token(t, f.anna.ID))

	steps := []struct {
		name   string
		method string
		opts   []testutil.Option
		want   string
	}{
		{"minh favorite a1", http.MethodPost, []testutil.Option{minh}, "favorited=true count=1"},
		{"anna favorite a1", http.MethodPost, []testutil.Option{anna}, "favorited=true count=2"},
		{"minh favorite a1 lần 2", http.MethodPost, []testutil.Option{minh}, "favorited=true count=2"},
		{"minh unfavorite a1", http.MethodDelete, []testutil.Option{minh}, "favorited=false count=1"},
		{"minh unfavorite a1 lần 2", http.MethodDelete, []testutil.Option{minh}, "favorited=false count=1"},
	}
	for _, s := range steps {
		res := testutil.Do(t, f.e, s.method, "/api/articles/a1/favorite", s.opts...)
		if res.Status != http.StatusOK {
			t.Fatalf("%s: status %d, muốn 200. Body: %s", s.name, res.Status, res.Body)
		}
		a := articleOf(t, res)
		if got := favState(a); got != s.want {
			t.Errorf("%s: %s, muốn %s", s.name, got, s.want)
		}
		if a["slug"] != "a1" || a["body"] != "Body a1" {
			t.Errorf("%s: response phải là bài a1 đầy đủ (có body), có %v", s.name, a)
		}
	}

	if got := favState(articleOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles/a1"))); got != "favorited=false count=1" {
		t.Errorf("khách xem a1: %s, muốn favorited=false count=1", got)
	}
	if got := favState(articleOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles/a1", anna))); got != "favorited=true count=1" {
		t.Errorf("anna xem a1: %s, muốn favorited=true count=1", got)
	}
}

func TestFavorites_Errors(t *testing.T) {
	f := newDay4Fixture(t)
	minh := testutil.WithToken(testutil.Token(t, f.minh.ID))
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		res := testutil.Do(t, f.e, method, "/api/articles/a1/favorite")
		if res.Status != http.StatusUnauthorized {
			t.Errorf("%s favorite không token: status %d, muốn 401. Body: %s", method, res.Status, res.Body)
		}
		res = testutil.Do(t, f.e, method, "/api/articles/khong-co/favorite")
		if res.Status != http.StatusUnauthorized {
			t.Errorf("%s favorite slug không có, không token: status %d, muốn 401 (jwt.Required chạy trước). Body: %s", method, res.Status, res.Body)
		}
		res = testutil.Do(t, f.e, method, "/api/articles/khong-co/favorite", minh)
		if res.Status != http.StatusNotFound || errorOf(res, "article") != "not found" {
			t.Errorf("%s favorite slug không có: %d %s, muốn 404 {\"article\": [\"not found\"]}", method, res.Status, res.Body)
		}
	}
}

func TestListArticles_FavoritedFieldsAndFilter(t *testing.T) {
	f := newDay4Fixture(t)
	var a1, a3 models.Article
	f.db.Where("slug = ?", "a1").First(&a1)
	f.db.Where("slug = ?", "a3").First(&a3)
	testutil.CreateFavorite(t, f.db, f.minh, a1)
	testutil.CreateFavorite(t, f.db, f.anna, a1)
	testutil.CreateFavorite(t, f.db, f.minh, a3)
	minh := testutil.WithToken(testutil.Token(t, f.minh.ID))

	state := func(res testutil.Response) map[string]string {
		out := map[string]string{}
		for _, a := range articlesOf(t, res) {
			out[a["slug"].(string)] = favState(a)
		}
		return out
	}
	got := state(testutil.Do(t, f.e, http.MethodGet, "/api/articles", minh))
	want := map[string]string{"a1": "favorited=true count=2", "a2": "favorited=false count=0", "a3": "favorited=true count=1"}
	for slug, w := range want {
		if got[slug] != w {
			t.Errorf("minh xem danh sách, bài %s: %s, muốn %s", slug, got[slug], w)
		}
	}
	got = state(testutil.Do(t, f.e, http.MethodGet, "/api/articles"))
	if got["a1"] != "favorited=false count=2" {
		t.Errorf("khách xem a1: %s, muốn favorited=false count=2", got["a1"])
	}

	cases := []struct {
		query string
		want  []string
	}{
		{"favorited=minh", []string{"a3", "a1"}},
		{"favorited=anna", []string{"a1"}},
		{"favorited=jake", []string{}},
		{"favorited=nobody", []string{}},
		{"favorited=minh&tag=go", []string{"a1"}},
		{"favorited=minh&author=anna", []string{"a3"}},
		{"favorited=minh&limit=1&offset=1", []string{"a1"}},
	}
	for _, tc := range cases {
		res := testutil.Do(t, f.e, http.MethodGet, "/api/articles?"+tc.query)
		if res.Status != http.StatusOK {
			t.Fatalf("?%s: status %d. Body: %s", tc.query, res.Status, res.Body)
		}
		if got := slugsOf(t, res); !slices.Equal(got, tc.want) && !(len(got) == 0 && len(tc.want) == 0) {
			t.Errorf("?%s: %v, muốn %v", tc.query, got, tc.want)
		}
	}
	res := testutil.Do(t, f.e, http.MethodGet, "/api/articles?favorited=minh&limit=1")
	if res.JSON["articlesCount"] != float64(2) {
		t.Errorf("?favorited=minh&limit=1: articlesCount %v, muốn 2 (đếm trước khi cắt trang)", res.JSON["articlesCount"])
	}
}

func TestFeed_FillsFavorites(t *testing.T) {
	f := newDay4Fixture(t)
	var a2 models.Article
	f.db.Where("slug = ?", "a2").First(&a2)
	testutil.CreateFollow(t, f.db, f.minh, f.jake)
	testutil.CreateFavorite(t, f.db, f.minh, a2)
	testutil.CreateFavorite(t, f.db, f.anna, a2)

	res := testutil.Do(t, f.e, http.MethodGet, "/api/articles/feed?favorited=anna", testutil.WithToken(testutil.Token(t, f.minh.ID)))
	if got := slugsOf(t, res); !slices.Equal(got, []string{"a2", "a1"}) {
		t.Fatalf("feed của minh: %v, muốn [a2 a1] (feed bỏ qua ?favorited=)", got)
	}
	if got := favState(articlesOf(t, res)[0]); got != "favorited=true count=2" {
		t.Errorf("feed, bài a2: %s, muốn favorited=true count=2", got)
	}
}

func TestFavorites_DeleteFavoritedArticle(t *testing.T) {
	f := newDay4Fixture(t)
	var a1 models.Article
	f.db.Where("slug = ?", "a1").First(&a1)
	testutil.CreateFavorite(t, f.db, f.minh, a1)
	res := testutil.Do(t, f.e, http.MethodDelete, "/api/articles/a1", testutil.WithToken(testutil.Token(t, f.jake.ID)))
	if res.Status != http.StatusNoContent {
		t.Fatalf("xóa bài đã có người thích: status %d, muốn 204. Body: %s", res.Status, res.Body)
	}
	var n int64
	f.db.Model(&models.Favorite{}).Count(&n)
	if n != 0 {
		t.Errorf("xóa bài xong còn %d dòng favorites, muốn 0 (ON DELETE CASCADE)", n)
	}
}

func TestListArticles_QueryCountConstant(t *testing.T) {
	counts := map[int][2]int64{}
	for _, size := range []int{2, 20} {
		t.Run(fmt.Sprintf("%d bài", size), func(t *testing.T) {
			e, db := newTestServer(t)
			jake := testutil.CreateUser(t, db, "jake", nil)
			minh := testutil.CreateUser(t, db, "minh", nil)
			testutil.CreateFollow(t, db, minh, jake)
			for i := range size {
				a := testutil.CreateArticle(t, db, jake, fmt.Sprintf("bai-%d", i), testutil.BaseTime, "go", fmt.Sprintf("tag-%d", i))
				testutil.CreateFavorite(t, db, minh, a)
			}
			queries := testutil.QueryCounter(t, db)
			viewer := testutil.Do(t, e, http.MethodGet, "/api/articles?limit=20", testutil.WithToken(testutil.Token(t, minh.ID)))
			nViewer := queries()
			guest := testutil.Do(t, e, http.MethodGet, "/api/articles?limit=20")
			nGuest := queries()
			if len(articlesOf(t, viewer)) != size || len(articlesOf(t, guest)) != size {
				t.Fatalf("danh sách trả về sai số bài, muốn %d", size)
			}
			counts[size] = [2]int64{nViewer, nGuest}
		})
	}
	t.Logf("số câu SQL (người xem, khách): 2 bài = %v, 20 bài = %v", counts[2], counts[20])
	if counts[2] != counts[20] {
		t.Errorf("số câu SQL (người xem, khách): 2 bài = %v, 20 bài = %v; muốn bằng nhau (có câu chạy trong vòng lặp: N+1)", counts[2], counts[20])
	}
	if counts[20][0] > 8 || counts[20][1] > 6 {
		t.Errorf("số câu SQL (người xem, khách) = %v, muốn tối đa (8, 6): đếm, trang, 3 preload, đếm favorite, follow, favorited", counts[20])
	}
}
