package server_test

import (
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

type day4Fixture struct {
	e                *echo.Echo
	db               *gorm.DB
	jake, anna, minh models.User
}

func newDay4Fixture(t *testing.T) day4Fixture {
	t.Helper()
	e, db := newTestServer(t)
	f := day4Fixture{e: e, db: db,
		jake: testutil.CreateUser(t, db, "jake", nil),
		anna: testutil.CreateUser(t, db, "anna", nil),
		minh: testutil.CreateUser(t, db, "minh", nil),
	}
	at := func(h int) time.Time { return testutil.BaseTime.Add(time.Duration(h) * time.Hour) }
	testutil.CreateArticle(t, db, f.jake, "a1", at(1), "go", "echo")
	testutil.CreateArticle(t, db, f.jake, "a2", at(2), "go")
	testutil.CreateArticle(t, db, f.anna, "a3", at(3), "echo")
	return f
}

func slugsOf(t *testing.T, res testutil.Response) []string {
	t.Helper()
	var out []string
	for _, a := range articlesOf(t, res) {
		s, _ := a["slug"].(string)
		out = append(out, s)
	}
	return out
}

func followingByAuthor(t *testing.T, res testutil.Response) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	for _, a := range articlesOf(t, res) {
		author, _ := a["author"].(map[string]any)
		name, _ := author["username"].(string)
		out[name], _ = author["following"].(bool)
	}
	return out
}

func TestListArticles_FiltersAndPaging(t *testing.T) {
	f := newDay4Fixture(t)
	cases := []struct {
		path  string
		slugs []string
		count float64
	}{
		{"/api/articles?tag=go", []string{"a2", "a1"}, 2},
		{"/api/articles?author=anna", []string{"a3"}, 1},
		{"/api/articles?tag=echo&author=jake", []string{"a1"}, 1},
		{"/api/articles?tag=go&limit=1&offset=1", []string{"a1"}, 2},
		{"/api/articles?tag=khong-co", nil, 0},
	}
	for _, tc := range cases {
		res := testutil.Do(t, f.e, http.MethodGet, tc.path)
		if res.Status != http.StatusOK {
			t.Fatalf("%s: %d %s", tc.path, res.Status, res.Body)
		}
		if got := slugsOf(t, res); !slices.Equal(got, tc.slugs) || res.JSON["articlesCount"] != tc.count {
			t.Errorf("%s: slugs %v, articlesCount %v; muốn %v, %v", tc.path, got, res.JSON["articlesCount"], tc.slugs, tc.count)
		}
	}
}

func TestListArticles_InvalidPaging(t *testing.T) {
	f := newDay4Fixture(t)
	cases := []struct {
		path   string
		status int
		field  string
	}{
		{"/api/articles?limit=abc", http.StatusBadRequest, ""},
		{"/api/articles?offset=xyz", http.StatusBadRequest, ""},
		{"/api/articles?limit=0", http.StatusUnprocessableEntity, "limit"},
		{"/api/articles?limit=101", http.StatusUnprocessableEntity, "limit"},
		{"/api/articles?offset=-1", http.StatusUnprocessableEntity, "offset"},
	}
	for _, tc := range cases {
		res := testutil.Do(t, f.e, http.MethodGet, tc.path)
		if res.Status != tc.status || (tc.field != "" && errorOf(res, tc.field) == "") {
			t.Errorf("%s: %d %s, muốn %d (field %q)", tc.path, res.Status, res.Body, tc.status, tc.field)
		}
	}
}

func TestListArticles_FollowingForViewer(t *testing.T) {
	f := newDay4Fixture(t)
	testutil.CreateFollow(t, f.db, f.minh, f.jake)

	got := followingByAuthor(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles", testutil.WithToken(testutil.Token(t, f.minh.ID))))
	if !got["jake"] || got["anna"] {
		t.Errorf("minh xem danh sách: following = %v, muốn jake true, anna false (route dùng jwt.Optional)", got)
	}
	got = followingByAuthor(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles"))
	if got["jake"] || got["anna"] {
		t.Errorf("khách xem danh sách: following = %v, muốn false hết", got)
	}
	res := testutil.Do(t, f.e, http.MethodGet, "/api/articles/a1", testutil.WithToken(testutil.Token(t, f.minh.ID)))
	if author, _ := articleOf(t, res)["author"].(map[string]any); author["following"] != true {
		t.Errorf("minh xem bài a1 của jake: author = %v, muốn following true", author)
	}
}

func TestFeed(t *testing.T) {
	f := newDay4Fixture(t)
	minhToken := testutil.Token(t, f.minh.ID)

	if res := testutil.Do(t, f.e, http.MethodGet, "/api/articles/feed"); res.Status != http.StatusUnauthorized || errorOf(res, "token") != "is missing" {
		t.Errorf("feed không token: %d %s, muốn 401 token is missing", res.Status, res.Body)
	}
	res := testutil.Do(t, f.e, http.MethodGet, "/api/articles/feed", testutil.WithToken(minhToken))
	if res.Status != http.StatusOK || len(articlesOf(t, res)) != 0 || res.JSON["articlesCount"] != float64(0) {
		t.Errorf("feed khi chưa follow ai: %d %s, muốn 200 với 0 bài", res.Status, res.Body)
	}

	testutil.CreateFollow(t, f.db, f.minh, f.jake)
	res = testutil.Do(t, f.e, http.MethodGet, "/api/articles/feed", testutil.WithToken(minhToken))
	if got := slugsOf(t, res); !slices.Equal(got, []string{"a2", "a1"}) || res.JSON["articlesCount"] != float64(2) {
		t.Errorf("feed sau khi follow jake: %v, articlesCount %v; muốn [a2 a1], 2", got, res.JSON["articlesCount"])
	}
	if got := followingByAuthor(t, res); !got["jake"] {
		t.Errorf("bài trong feed có author.following = %v, muốn true", got)
	}
	res = testutil.Do(t, f.e, http.MethodGet, "/api/articles/feed?limit=1&offset=1", testutil.WithToken(minhToken))
	if got := slugsOf(t, res); !slices.Equal(got, []string{"a1"}) || res.JSON["articlesCount"] != float64(2) {
		t.Errorf("feed limit 1 offset 1: %v, articlesCount %v; muốn [a1], 2", got, res.JSON["articlesCount"])
	}
}

func TestProfiles_FollowFlow(t *testing.T) {
	f := newDay4Fixture(t)
	minhToken := testutil.Token(t, f.minh.ID)
	profile := func(res testutil.Response) map[string]any {
		t.Helper()
		p, ok := res.JSON["profile"].(map[string]any)
		if !ok {
			t.Fatalf("body phải có dạng {\"profile\": {...}}. Status %d, body %s", res.Status, res.Body)
		}
		return p
	}

	res := testutil.Do(t, f.e, http.MethodPost, "/api/profiles/jake/follow", testutil.WithToken(minhToken))
	if res.Status != http.StatusOK || profile(res)["following"] != true || profile(res)["username"] != "jake" {
		t.Fatalf("follow: %d %s, muốn 200 profile jake following true", res.Status, res.Body)
	}
	if p := profile(testutil.Do(t, f.e, http.MethodGet, "/api/profiles/jake", testutil.WithToken(minhToken))); p["following"] != true {
		t.Errorf("GET profile jake (minh): following = %v, muốn true", p["following"])
	}
	if p := profile(testutil.Do(t, f.e, http.MethodGet, "/api/profiles/jake")); p["following"] != false {
		t.Errorf("GET profile jake (khách): following = %v, muốn false", p["following"])
	}
	res = testutil.Do(t, f.e, http.MethodDelete, "/api/profiles/jake/follow", testutil.WithToken(minhToken))
	if res.Status != http.StatusOK || profile(res)["following"] != false {
		t.Errorf("unfollow: %d %s, muốn 200 following false", res.Status, res.Body)
	}

	if res := testutil.Do(t, f.e, http.MethodPost, "/api/profiles/jake/follow"); res.Status != http.StatusUnauthorized {
		t.Errorf("follow không token: %d, muốn 401", res.Status)
	}
	if res := testutil.Do(t, f.e, http.MethodPost, "/api/profiles/khong-co/follow", testutil.WithToken(minhToken)); res.Status != http.StatusNotFound || errorOf(res, "profile") != "not found" {
		t.Errorf("follow user không có: %d %s, muốn 404 profile not found", res.Status, res.Body)
	}
	if res := testutil.Do(t, f.e, http.MethodPost, "/api/profiles/minh/follow", testutil.WithToken(minhToken)); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("tự follow mình: %d %s, muốn 422", res.Status, res.Body)
	}
}

func TestOwnerMiddleware_RunsBeforeBind(t *testing.T) {
	f := newDay4Fixture(t)
	annaToken := testutil.Token(t, f.anna.ID)
	res := testutil.Do(t, f.e, http.MethodPut, "/api/articles/a1", testutil.WithToken(annaToken), testutil.WithJSON(`{"article":`))
	if res.Status != http.StatusForbidden || errorOf(res, "article") != "forbidden" {
		t.Errorf("anna PUT bài jake với JSON hỏng: %d %s, muốn 403 (middleware RequireOwner chạy trước c.Bind)", res.Status, res.Body)
	}
	res = testutil.Do(t, f.e, http.MethodDelete, "/api/articles/khong-co", testutil.WithToken(annaToken))
	if res.Status != http.StatusNotFound || errorOf(res, "article") != "not found" {
		t.Errorf("DELETE slug không có: %d %s, muốn 404 article not found", res.Status, res.Body)
	}
}
