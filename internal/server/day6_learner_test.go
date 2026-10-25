package server_test

import (
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestFeed_QueryCountConstant(t *testing.T) {
	counts := map[int]int64{}
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
			res := testutil.Do(t, e, http.MethodGet, "/api/articles/feed?limit=20", testutil.WithToken(testutil.Token(t, minh.ID)))
			n := queries()

			if got := len(articlesOf(t, res)); got != size {
				t.Fatalf("feed có %d bài, muốn %d", got, size)
			}
			counts[size] = n
		})
	}
	t.Logf("số câu SQL của feed: 2 bài = %d, 20 bài = %d", counts[2], counts[20])
	if counts[2] != counts[20] {
		t.Errorf("feed: 2 bài = %d câu, 20 bài = %d câu; muốn bằng nhau (có câu chạy trong vòng lặp: N+1)", counts[2], counts[20])
	}
}

func TestFullFlow_RegisterToFeed(t *testing.T) {
	e, _ := newTestServer(t)

	register := func(name string) testutil.Option {
		t.Helper()
		res := testutil.Do(t, e, http.MethodPost, "/api/users",
			testutil.WithJSON(fmt.Sprintf(`{"user":{"username":%q,"email":"%s@x.io","password":"password123"}}`, name, name)))
		if res.Status != http.StatusCreated {
			t.Fatalf("đăng ký %s: %d %s", name, res.Status, res.Body)
		}
		return testutil.WithToken(userOf(t, res)["token"].(string))
	}
	jake, minh := register("jake"), register("minh")

	res := testutil.Do(t, e, http.MethodPost, "/api/articles", jake,
		testutil.WithJSON(`{"article":{"title":"Học Echo","description":"d","body":"b","tagList":["go"]}}`))
	if res.Status != http.StatusCreated {
		t.Fatalf("jake đăng bài: %d %s", res.Status, res.Body)
	}
	slug := articleOf(t, res)["slug"].(string)

	steps := []struct {
		name, method, path, body string
		want                     int
	}{
		{"minh bình luận", http.MethodPost, "/api/articles/" + slug + "/comments", `{"comment":{"body":"Hay quá"}}`, http.StatusCreated},
		{"minh thích bài", http.MethodPost, "/api/articles/" + slug + "/favorite", "", http.StatusOK},
		{"minh follow jake", http.MethodPost, "/api/profiles/jake/follow", "", http.StatusOK},
	}
	for _, s := range steps {
		opts := []testutil.Option{minh}
		if s.body != "" {
			opts = append(opts, testutil.WithJSON(s.body))
		}
		if res := testutil.Do(t, e, s.method, s.path, opts...); res.Status != s.want {
			t.Fatalf("%s: %d, muốn %d. Body: %s", s.name, res.Status, s.want, res.Body)
		}
	}

	feed := articlesOf(t, testutil.Do(t, e, http.MethodGet, "/api/articles/feed", minh))
	if len(feed) != 1 || feed[0]["slug"] != slug {
		t.Fatalf("feed của minh = %v, muốn đúng 1 bài %s", feed, slug)
	}
	if feed[0]["favorited"] != true || feed[0]["favoritesCount"] != float64(1) {
		t.Errorf("feed: favorited = %v, favoritesCount = %v; muốn true, 1", feed[0]["favorited"], feed[0]["favoritesCount"])
	}
	if author, _ := feed[0]["author"].(map[string]any); author["following"] != true {
		t.Errorf("feed: author.following = %v, muốn true", author["following"])
	}

	if comments := commentsOf(t, testutil.Do(t, e, http.MethodGet, "/api/articles/"+slug+"/comments")); len(comments) != 1 {
		t.Errorf("bình luận = %v, muốn 1", comments)
	}
	if got := slugsOf(t, testutil.Do(t, e, http.MethodGet, "/api/articles?favorited=minh")); !slices.Equal(got, []string{slug}) {
		t.Errorf("?favorited=minh = %v, muốn [%s]", got, slug)
	}

	testutil.Do(t, e, http.MethodDelete, "/api/profiles/jake/follow", minh)
	if feed := articlesOf(t, testutil.Do(t, e, http.MethodGet, "/api/articles/feed", minh)); len(feed) != 0 {
		t.Errorf("sau khi unfollow, feed còn %d bài, muốn 0", len(feed))
	}
}
