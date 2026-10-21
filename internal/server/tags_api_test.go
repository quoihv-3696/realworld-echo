package server_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func tagsOf(t *testing.T, res testutil.Response) []string {
	t.Helper()
	raw, ok := res.JSON["tags"].([]any)
	if !ok {
		t.Fatalf("\"tags\" phải là mảng (rỗng thì [] chứ không phải null). Body: %s", res.Body)
	}
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		s, ok := v.(string)
		if !ok {
			t.Fatalf("mỗi tag phải là chuỗi. Body: %s", res.Body)
		}
		out = append(out, s)
	}
	return out
}

func TestListTags_Empty(t *testing.T) {
	e, _ := newTestServer(t)
	res := testutil.Do(t, e, http.MethodGet, "/api/tags")
	if res.Status != http.StatusOK {
		t.Fatalf("GET /api/tags: status = %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	if got := tagsOf(t, res); len(got) != 0 {
		t.Errorf("tags = %v, muốn []", got)
	}
}

func TestListTags_SortedByName(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime, "go", "echo")
	testutil.CreateArticle(t, db, jake, "a2", testutil.BaseTime, "gorm", "go")

	res := testutil.Do(t, e, http.MethodGet, "/api/tags")
	if res.Status != http.StatusOK {
		t.Fatalf("GET /api/tags: status = %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	want := []string{"echo", "go", "gorm"}
	if got := tagsOf(t, res); !slices.Equal(got, want) {
		t.Errorf("tags = %v, muốn %v (mỗi tag một lần, sắp theo tên)", got, want)
	}
}
