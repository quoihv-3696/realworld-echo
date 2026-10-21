package server_test

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

var isoTime = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)

func articlesOf(t *testing.T, res testutil.Response) []map[string]any {
	t.Helper()
	raw, ok := res.JSON["articles"].([]any)
	if !ok {
		t.Fatalf("\"articles\" phải là mảng (rỗng thì [] chứ không phải null). Body: %s", res.Body)
	}
	out := make([]map[string]any, 0, len(raw))
	for _, v := range raw {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("mỗi phần tử của articles phải là object. Body: %s", res.Body)
		}
		out = append(out, m)
	}
	return out
}

func stringList(t *testing.T, v any, name string) []string {
	t.Helper()
	raw, ok := v.([]any)
	if !ok {
		t.Fatalf("%s phải là mảng chuỗi ([] khi rỗng, không phải null), nhận: %#v", name, v)
	}
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		out = append(out, fmt.Sprint(s))
	}
	return out
}

func checkArticleShape(t *testing.T, a map[string]any, withBody bool) {
	t.Helper()
	for _, key := range []string{"slug", "title", "description", "tagList", "createdAt", "updatedAt", "favorited", "favoritesCount", "author"} {
		if _, ok := a[key]; !ok {
			t.Errorf("article thiếu trường %q: %v", key, a)
		}
	}
	_, hasBody := a["body"]
	if withBody && !hasBody {
		t.Errorf("GET một bài viết phải có \"body\": %v", a)
	}
	if !withBody && hasBody {
		t.Errorf("danh sách bài viết KHÔNG được có \"body\" (đúng spec RealWorld): %v", a)
	}
	for _, key := range []string{"createdAt", "updatedAt"} {
		if s, _ := a[key].(string); !isoTime.MatchString(s) {
			t.Errorf("%s = %#v, muốn chuỗi ISO 8601 dạng 2026-01-02T03:04:05...", key, a[key])
		}
	}
	if a["favorited"] != false {
		t.Errorf("favorited = %#v, ngày 1 luôn false", a["favorited"])
	}
	if a["favoritesCount"] != float64(0) {
		t.Errorf("favoritesCount = %#v, ngày 1 luôn 0", a["favoritesCount"])
	}
	author, ok := a["author"].(map[string]any)
	if !ok {
		t.Fatalf("author phải là object: %v", a)
	}
	for _, key := range []string{"username", "bio", "image", "following"} {
		if _, ok := author[key]; !ok {
			t.Errorf("author thiếu trường %q (bio/image rỗng thì là null, vẫn phải có key): %v", key, author)
		}
	}
	if author["following"] != false {
		t.Errorf("author.following = %#v, ngày 1 luôn false", author["following"])
	}
}

func TestListArticles_Empty(t *testing.T) {
	e, _ := newTestServer(t)
	res := testutil.Do(t, e, http.MethodGet, "/api/articles")
	if res.Status != http.StatusOK {
		t.Fatalf("GET /api/articles: status = %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	if got := articlesOf(t, res); len(got) != 0 {
		t.Errorf("articles có %d phần tử, muốn 0", len(got))
	}
	if res.JSON["articlesCount"] != float64(0) {
		t.Errorf("articlesCount = %#v, muốn 0", res.JSON["articlesCount"])
	}
}

func TestListArticles_NewestFirstWithoutBody(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", testutil.StrPtr("I work at statefarm"))
	anna := testutil.CreateUser(t, db, "anna", nil)
	testutil.CreateArticle(t, db, jake, "old", testutil.BaseTime, "go", "echo")
	testutil.CreateArticle(t, db, anna, "newest", testutil.BaseTime.Add(2*time.Hour))
	testutil.CreateArticle(t, db, jake, "middle", testutil.BaseTime.Add(time.Hour), "gorm")

	res := testutil.Do(t, e, http.MethodGet, "/api/articles")
	if res.Status != http.StatusOK {
		t.Fatalf("GET /api/articles: status = %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	if res.JSON["articlesCount"] != float64(3) {
		t.Errorf("articlesCount = %#v, muốn 3", res.JSON["articlesCount"])
	}
	list := articlesOf(t, res)
	var slugs []string
	for _, a := range list {
		checkArticleShape(t, a, false)
		slugs = append(slugs, fmt.Sprint(a["slug"]))
	}
	if want := []string{"newest", "middle", "old"}; !slices.Equal(slugs, want) {
		t.Fatalf("thứ tự slug = %v, muốn %v (mới nhất trước)", slugs, want)
	}

	old := list[2]
	if got := stringList(t, old["tagList"], "tagList"); !slices.Equal(got, []string{"echo", "go"}) {
		t.Errorf("tagList của \"old\" = %v, muốn [echo go] (sắp theo tên)", got)
	}
	if got := stringList(t, list[0]["tagList"], "tagList"); len(got) != 0 {
		t.Errorf("tagList của \"newest\" = %v, muốn []", got)
	}
	author := old["author"].(map[string]any)
	if author["username"] != "jake" || author["bio"] != "I work at statefarm" || author["image"] != nil {
		t.Errorf("author của \"old\" = %v, muốn username jake, bio \"I work at statefarm\", image null", author)
	}
	if bio := list[0]["author"].(map[string]any)["bio"]; bio != nil {
		t.Errorf("bio của anna = %#v, muốn null", bio)
	}
	if old["createdAt"] != "2026-01-02T03:04:05Z" {
		t.Errorf("createdAt của \"old\" = %#v, muốn \"2026-01-02T03:04:05Z\" (giờ UTC)", old["createdAt"])
	}
}

func TestListArticles_DefaultLimit20CountsAll(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	for i := range 25 {
		testutil.CreateArticle(t, db, jake, fmt.Sprintf("a%02d", i), testutil.BaseTime.Add(time.Duration(i)*time.Minute))
	}

	res := testutil.Do(t, e, http.MethodGet, "/api/articles")
	if res.Status != http.StatusOK {
		t.Fatalf("GET /api/articles: status = %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	list := articlesOf(t, res)
	if len(list) != 20 {
		t.Errorf("trả %d bài, muốn 20 (limit mặc định)", len(list))
	}
	if res.JSON["articlesCount"] != float64(25) {
		t.Errorf("articlesCount = %#v, muốn 25 (đếm TẤT CẢ bài, trước khi cắt trang)", res.JSON["articlesCount"])
	}
	if len(list) > 0 && list[0]["slug"] != "a24" {
		t.Errorf("bài đầu = %v, muốn a24 (mới nhất)", list[0]["slug"])
	}
}

func TestGetArticle(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	if err := db.Model(&jake).Update("image", "https://i.example/jake.png").Error; err != nil {
		t.Fatal(err)
	}
	testutil.CreateArticle(t, db, jake, "how-to-train-your-dragon", testutil.BaseTime, "training", "dragons")

	res := testutil.Do(t, e, http.MethodGet, "/api/articles/how-to-train-your-dragon")
	if res.Status != http.StatusOK {
		t.Fatalf("GET /api/articles/:slug: status = %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	a, ok := res.JSON["article"].(map[string]any)
	if !ok {
		t.Fatalf("body phải có dạng {\"article\": {...}}. Body: %s", res.Body)
	}
	checkArticleShape(t, a, true)
	if a["slug"] != "how-to-train-your-dragon" || a["title"] != "Title how-to-train-your-dragon" || a["body"] != "Body how-to-train-your-dragon" {
		t.Errorf("sai nội dung bài viết: %v", a)
	}
	if got := stringList(t, a["tagList"], "tagList"); !slices.Equal(got, []string{"dragons", "training"}) {
		t.Errorf("tagList = %v, muốn [dragons training]", got)
	}
	if img := a["author"].(map[string]any)["image"]; img != "https://i.example/jake.png" {
		t.Errorf("author.image = %#v, muốn \"https://i.example/jake.png\"", img)
	}
}

func TestDatabaseError_Returns500(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "dragon", testutil.BaseTime, "go")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	for _, path := range []string{"/api/tags", "/api/articles", "/api/articles/dragon"} {
		res := testutil.Do(t, e, http.MethodGet, path, testutil.WithContext(ctx))
		if res.Status != http.StatusInternalServerError {
			t.Errorf("GET %s khi DB lỗi (context đã hủy): status = %d, muốn 500. Body: %s\n"+
				"Gợi ý: repository có WithContext(ctx)? handler có tự viết status thay vì return err?", path, res.Status, res.Body)
		}
	}
}

func TestGetArticle_NotFound(t *testing.T) {
	e, _ := newTestServer(t)
	res := testutil.Do(t, e, http.MethodGet, "/api/articles/khong-ton-tai")
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, muốn 404. Body: %s", res.Status, res.Body)
	}
	errs, _ := res.JSON["errors"].(map[string]any)
	msgs, _ := errs["article"].([]any)
	if len(msgs) != 1 || msgs[0] != "not found" {
		t.Errorf("body = %s, muốn {\"errors\":{\"article\":[\"not found\"]}}", res.Body)
	}
}
