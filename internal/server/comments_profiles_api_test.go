package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func commentsOf(t *testing.T, res testutil.Response) []any {
	t.Helper()
	list, ok := res.JSON["comments"].([]any)
	if !ok {
		t.Fatalf("\"comments\" phải là mảng ([] khi rỗng, không phải null). Status %d, body: %s", res.Status, res.Body)
	}
	return list
}

func TestComments_Flow(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	token := testutil.Token(t, jake.ID)

	res := testutil.Do(t, e, http.MethodPost, "/api/articles/a/comments", testutil.WithToken(token), testutil.WithJSON(`{"comment":{"body":"Bình luận đầu tiên"}}`))
	if res.Status != http.StatusCreated {
		t.Fatalf("POST comment: %d %s, muốn 201", res.Status, res.Body)
	}
	c, _ := res.JSON["comment"].(map[string]any)
	id, isNumber := c["id"].(float64)
	author, _ := c["author"].(map[string]any)
	if !isNumber || c["body"] != "Bình luận đầu tiên" || author["username"] != "jake" {
		t.Fatalf("comment = %v, muốn id là số, body đúng, author jake", c)
	}
	for _, k := range []string{"createdAt", "updatedAt"} {
		if s, _ := c[k].(string); !isoTime.MatchString(s) {
			t.Errorf("%s = %v, muốn dạng 2006-01-02T15:04:05", k, c[k])
		}
	}

	if list := commentsOf(t, testutil.Do(t, e, http.MethodGet, "/api/articles/a/comments")); len(list) != 1 {
		t.Errorf("GET comments (không token): %d comment, muốn 1", len(list))
	}
	res = testutil.Do(t, e, http.MethodDelete, fmt.Sprintf("/api/articles/a/comments/%d", int(id)), testutil.WithToken(token))
	if res.Status != http.StatusNoContent || res.Body != "" {
		t.Fatalf("DELETE comment: %d %q, muốn 204 không có body", res.Status, res.Body)
	}
	if list := commentsOf(t, testutil.Do(t, e, http.MethodGet, "/api/articles/a/comments")); len(list) != 0 {
		t.Errorf("sau khi xóa còn %d comment", len(list))
	}
}

func TestComments_Errors(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	a := testutil.CreateArticle(t, db, jake, "a", testutil.BaseTime)
	annas := testutil.CreateComment(t, db, a, anna, "của anna")
	token := testutil.Token(t, jake.ID)

	cases := []struct {
		name, method, path, body string
		withToken                bool
		status                   int
		field, msg               string
	}{
		{"POST không token", http.MethodPost, "/api/articles/a/comments", `{"comment":{"body":"x"}}`, false, 401, "token", "is missing"},
		{"POST body rỗng", http.MethodPost, "/api/articles/a/comments", `{"comment":{"body":""}}`, true, 422, "body", "can't be blank"},
		{"POST bài không có", http.MethodPost, "/api/articles/khong-co/comments", `{"comment":{"body":"x"}}`, true, 404, "article", "not found"},
		{"GET bài không có", http.MethodGet, "/api/articles/khong-co/comments", "", false, 404, "article", "not found"},
		{"DELETE bài không có", http.MethodDelete, "/api/articles/khong-co/comments/1", "", true, 404, "article", "not found"},
		{"DELETE comment không có", http.MethodDelete, "/api/articles/a/comments/99999", "", true, 404, "comment", "not found"},
		{"DELETE comment của người khác", http.MethodDelete, fmt.Sprintf("/api/articles/a/comments/%d", annas.ID), "", true, 403, "comment", "forbidden"},
		{"DELETE id không phải số", http.MethodDelete, "/api/articles/a/comments/abc", "", true, 400, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var opts []testutil.Option
			if tc.withToken {
				opts = append(opts, testutil.WithToken(token))
			}
			if tc.body != "" {
				opts = append(opts, testutil.WithJSON(tc.body))
			}
			res := testutil.Do(t, e, tc.method, tc.path, opts...)
			if res.Status != tc.status || (tc.field != "" && errorOf(res, tc.field) != tc.msg) {
				t.Errorf("%d %s, muốn %d {%q: [%q]}", res.Status, res.Body, tc.status, tc.field, tc.msg)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", testutil.StrPtr("Gopher"))

	for name, opts := range map[string][]testutil.Option{
		"không token": nil,
		"có token":    {testutil.WithToken(testutil.Token(t, jake.ID))},
	} {
		res := testutil.Do(t, e, http.MethodGet, "/api/profiles/jake", opts...)
		p, _ := res.JSON["profile"].(map[string]any)
		if res.Status != http.StatusOK || p["username"] != "jake" || p["bio"] != "Gopher" || p["image"] != nil || p["following"] != false {
			t.Errorf("%s: %d %s, muốn 200 profile jake, bio Gopher, image null, following false", name, res.Status, res.Body)
		}
	}
	res := testutil.Do(t, e, http.MethodGet, "/api/profiles/khong-co")
	if res.Status != http.StatusNotFound || errorOf(res, "profile") != "not found" {
		t.Errorf("user không có: %d %s, muốn 404 profile not found", res.Status, res.Body)
	}
}
