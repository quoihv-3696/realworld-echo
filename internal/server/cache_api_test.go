package server_test

import (
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/cache"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/server"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

type cachedFixture struct {
	e                *echo.Echo
	db               *gorm.DB
	mem              *testutil.MemoryCache
	jake, anna, minh models.User
	a1               models.Article
}

func newCachedFixture(t *testing.T) cachedFixture {
	t.Helper()
	db := testutil.DB(t)
	mem := testutil.NewMemoryCache()
	e := server.New(db, testutil.JWT(t), cache.New(mem, time.Minute, testutil.Logger(t)))
	e.Logger = testutil.Logger(t)
	f := cachedFixture{e: e, db: db, mem: mem,
		jake: testutil.CreateUser(t, db, "jake", nil),
		anna: testutil.CreateUser(t, db, "anna", nil),
		minh: testutil.CreateUser(t, db, "minh", nil),
	}
	f.a1 = testutil.CreateArticle(t, db, f.jake, "a1", testutil.BaseTime, "go")
	testutil.CreateArticle(t, db, f.anna, "a2", testutil.BaseTime.Add(time.Hour), "echo")
	return f
}

func (f cachedFixture) token(t *testing.T, u models.User) testutil.Option {
	return testutil.WithToken(testutil.Token(t, u.ID))
}

func TestCache_TagsServedFromCacheAndRefreshed(t *testing.T) {
	f := newCachedFixture(t)
	queries := testutil.QueryCounter(t, f.db)
	testutil.Do(t, f.e, http.MethodGet, "/api/tags")
	if n := queries(); n == 0 {
		t.Fatal("GET /api/tags lần đầu: 0 câu SQL, muốn đọc DB (cache còn trống)")
	}
	if got := tagsOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/tags")); !slices.Equal(got, []string{"echo", "go"}) {
		t.Fatalf("tags = %v", got)
	}
	if n := queries(); n != 0 {
		t.Errorf("GET /api/tags lần 2: %d câu SQL, muốn 0 (lấy từ cache)", n)
	}

	res := testutil.Do(t, f.e, http.MethodPost, "/api/articles", f.token(t, f.jake),
		testutil.WithJSON(`{"article":{"title":"Bài có tag mới","description":"d","body":"b","tagList":["tag-moi"]}}`))
	if res.Status != http.StatusCreated {
		t.Fatalf("tạo bài: %d %s", res.Status, res.Body)
	}
	if got := tagsOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/tags")); !slices.Contains(got, "tag-moi") {
		t.Errorf("sau khi tạo bài có tag mới, GET /api/tags = %v: thiếu tag-moi (quên Invalidate khi tạo bài?)", got)
	}
}

func TestCache_GuestListFreshAfterEachWrite(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, f cachedFixture)
		write func(t *testing.T, f cachedFixture) testutil.Response
		check func(t *testing.T, list []map[string]any)
	}{
		{"tạo bài", nil,
			func(t *testing.T, f cachedFixture) testutil.Response {
				return testutil.Do(t, f.e, http.MethodPost, "/api/articles", f.token(t, f.jake),
					testutil.WithJSON(`{"article":{"title":"Bài mới","description":"d","body":"b"}}`))
			},
			func(t *testing.T, list []map[string]any) {
				if len(list) != 3 {
					t.Errorf("có %d bài, muốn 3 (bài vừa tạo phải hiện)", len(list))
				}
			}},
		{"sửa bài", nil,
			func(t *testing.T, f cachedFixture) testutil.Response {
				return testutil.Do(t, f.e, http.MethodPut, "/api/articles/a1", f.token(t, f.jake),
					testutil.WithJSON(`{"article":{"title":"Tiêu đề mới"}}`))
			},
			func(t *testing.T, list []map[string]any) {
				if titleOf(list, "a1") != "Tiêu đề mới" {
					t.Errorf("tiêu đề a1 = %q, muốn tiêu đề mới", titleOf(list, "a1"))
				}
			}},
		{"xóa bài", nil,
			func(t *testing.T, f cachedFixture) testutil.Response {
				return testutil.Do(t, f.e, http.MethodDelete, "/api/articles/a1", f.token(t, f.jake))
			},
			func(t *testing.T, list []map[string]any) {
				if titleOf(list, "a1") != "" {
					t.Error("bài a1 đã xóa vẫn còn trong danh sách")
				}
			}},
		{"thích bài", nil,
			func(t *testing.T, f cachedFixture) testutil.Response {
				return testutil.Do(t, f.e, http.MethodPost, "/api/articles/a1/favorite", f.token(t, f.minh))
			},
			func(t *testing.T, list []map[string]any) {
				if countOf(list, "a1") != float64(1) {
					t.Errorf("favoritesCount của a1 = %v, muốn 1", countOf(list, "a1"))
				}
			}},
		{"bỏ thích",
			func(t *testing.T, f cachedFixture) { testutil.CreateFavorite(t, f.db, f.minh, f.a1) },
			func(t *testing.T, f cachedFixture) testutil.Response {
				return testutil.Do(t, f.e, http.MethodDelete, "/api/articles/a1/favorite", f.token(t, f.minh))
			},
			func(t *testing.T, list []map[string]any) {
				if countOf(list, "a1") != float64(0) {
					t.Errorf("favoritesCount của a1 = %v, muốn 0", countOf(list, "a1"))
				}
			}},
		{"sửa bio tác giả", nil,
			func(t *testing.T, f cachedFixture) testutil.Response {
				return testutil.Do(t, f.e, http.MethodPut, "/api/user", f.token(t, f.jake),
					testutil.WithJSON(`{"user":{"bio":"bio mới"}}`))
			},
			func(t *testing.T, list []map[string]any) {
				for _, a := range list {
					author, _ := a["author"].(map[string]any)
					if a["slug"] == "a1" && author["bio"] != "bio mới" {
						t.Errorf("bio tác giả của a1 = %v, muốn \"bio mới\" (UserService.Update phải Invalidate)", author["bio"])
					}
				}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCachedFixture(t)
			if tc.setup != nil {
				tc.setup(t, f)
			}
			queries := testutil.QueryCounter(t, f.db)
			testutil.Do(t, f.e, http.MethodGet, "/api/articles")
			queries()
			testutil.Do(t, f.e, http.MethodGet, "/api/articles")
			if n := queries(); n != 0 {
				t.Fatalf("khách GET /api/articles lần 2: %d câu SQL, muốn 0 (chưa cache danh sách cho khách?)", n)
			}
			if res := tc.write(t, f); res.Status >= 300 {
				t.Fatalf("%s: %d %s", tc.name, res.Status, res.Body)
			}
			tc.check(t, articlesOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles")))
		})
	}
}

func titleOf(list []map[string]any, slug string) string {
	for _, a := range list {
		if a["slug"] == slug {
			s, _ := a["title"].(string)
			return s
		}
	}
	return ""
}

func countOf(list []map[string]any, slug string) any {
	for _, a := range list {
		if a["slug"] == slug {
			return a["favoritesCount"]
		}
	}
	return nil
}

func TestCache_ViewerRequestsNotCached(t *testing.T) {
	f := newCachedFixture(t)
	testutil.CreateFollow(t, f.db, f.minh, f.jake)
	queries := testutil.QueryCounter(t, f.db)
	for _, path := range []string{"/api/articles", "/api/articles/feed"} {
		for i := 1; i <= 2; i++ {
			testutil.Do(t, f.e, http.MethodGet, path, f.token(t, f.minh))
			if n := queries(); n == 0 {
				t.Errorf("minh GET %s lần %d: 0 câu SQL, muốn đọc DB (favorited/following theo người xem: không cache)", path, i)
			}
		}
	}
}

func TestCache_BackendDownStillServes(t *testing.T) {
	f := newCachedFixture(t)
	f.mem.Fail(errors.New("redis: connection refused"))
	if got := tagsOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/tags")); !slices.Equal(got, []string{"echo", "go"}) {
		t.Errorf("Redis sập: tags = %v, muốn đọc thẳng DB", got)
	}
	if got := slugsOf(t, testutil.Do(t, f.e, http.MethodGet, "/api/articles")); !slices.Equal(got, []string{"a2", "a1"}) {
		t.Errorf("Redis sập: danh sách = %v, muốn đọc thẳng DB", got)
	}
	res := testutil.Do(t, f.e, http.MethodPost, "/api/articles/a1/favorite", f.token(t, f.minh))
	if res.Status != http.StatusOK {
		t.Errorf("Redis sập: favorite = %d %s, muốn 200 (Invalidate lỗi chỉ ghi log)", res.Status, res.Body)
	}
}
