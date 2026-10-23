package server_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

const hocGoJSON = `{"article":{"title":"Học Go trong 30 ngày","description":"Lộ trình","body":"Bắt đầu","tagList":["go","beginner"]}}`

func articleOf(t *testing.T, res testutil.Response) map[string]any {
	t.Helper()
	a, ok := res.JSON["article"].(map[string]any)
	if !ok {
		t.Fatalf("body phải có dạng {\"article\": {...}}. Status %d, body: %s", res.Status, res.Body)
	}
	return a
}

func TestCreateArticle(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	token := testutil.Token(t, jake.ID)

	res := testutil.Do(t, e, http.MethodPost, "/api/articles", testutil.WithToken(token), testutil.WithJSON(hocGoJSON))
	if res.Status != http.StatusCreated {
		t.Fatalf("status = %d, muốn 201. Body: %s", res.Status, res.Body)
	}
	a := articleOf(t, res)
	checkArticleShape(t, a, true)
	if a["slug"] != "hoc-go-trong-30-ngay" || a["body"] != "Bắt đầu" {
		t.Errorf("slug = %v, body = %v; muốn hoc-go-trong-30-ngay, Bắt đầu", a["slug"], a["body"])
	}
	if tags := stringList(t, a["tagList"], "tagList"); !slices.Equal(tags, []string{"beginner", "go"}) {
		t.Errorf("tagList = %v, muốn [beginner go]", tags)
	}
	if author, _ := a["author"].(map[string]any); author["username"] != "jake" {
		t.Errorf("author = %v, muốn jake (người giữ token)", a["author"])
	}

	got := testutil.Do(t, e, http.MethodGet, "/api/articles/hoc-go-trong-30-ngay")
	if got.Status != http.StatusOK || articleOf(t, got)["title"] != "Học Go trong 30 ngày" {
		t.Errorf("GET bài vừa tạo: %d %s", got.Status, got.Body)
	}
}

func TestCreateArticle_Errors(t *testing.T) {
	e, db := newTestServer(t)
	token := testutil.Token(t, testutil.CreateUser(t, db, "jake", nil).ID)

	if res := testutil.Do(t, e, http.MethodPost, "/api/articles", testutil.WithJSON(hocGoJSON)); res.Status != http.StatusUnauthorized || errorOf(res, "token") != "is missing" {
		t.Errorf("không token: %d %s, muốn 401 token is missing", res.Status, res.Body)
	}
	res := testutil.Do(t, e, http.MethodPost, "/api/articles", testutil.WithToken(token), testutil.WithJSON(`{"article":{"title":"","description":"d","body":"b"}}`))
	if res.Status != http.StatusUnprocessableEntity || errorOf(res, "title") != "can't be blank" {
		t.Errorf("title rỗng: %d %s, muốn 422 title can't be blank", res.Status, res.Body)
	}
	first := testutil.Do(t, e, http.MethodPost, "/api/articles", testutil.WithToken(token), testutil.WithJSON(hocGoJSON))
	second := testutil.Do(t, e, http.MethodPost, "/api/articles", testutil.WithToken(token), testutil.WithJSON(hocGoJSON))
	if first.Status != http.StatusCreated || second.Status != http.StatusCreated {
		t.Fatalf("hai bài trùng tiêu đề: %d, %d; muốn 201 cả hai", first.Status, second.Status)
	}
	if articleOf(t, first)["slug"] == articleOf(t, second)["slug"] {
		t.Errorf("hai bài trùng tiêu đề có cùng slug %v", articleOf(t, first)["slug"])
	}
}

func TestUpdateArticle(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	testutil.CreateArticle(t, db, jake, "bai-cua-jake", testutil.BaseTime, "go", "echo")
	jakeToken, annaToken := testutil.Token(t, jake.ID), testutil.Token(t, anna.ID)
	put := func(token, slug, body string) testutil.Response {
		t.Helper()
		return testutil.Do(t, e, http.MethodPut, "/api/articles/"+slug, testutil.WithToken(token), testutil.WithJSON(body))
	}

	res := put(jakeToken, "bai-cua-jake", `{"article":{"body":"Nội dung mới"}}`)
	if res.Status != http.StatusOK {
		t.Fatalf("PUT: %d %s", res.Status, res.Body)
	}
	a := articleOf(t, res)
	if a["body"] != "Nội dung mới" || a["slug"] != "bai-cua-jake" || len(stringList(t, a["tagList"], "tagList")) != 2 {
		t.Errorf("sau PUT body = %v, slug = %v, tagList = %v; muốn nội dung mới, giữ slug và 2 tag", a["body"], a["slug"], a["tagList"])
	}
	if a["createdAt"] != "2026-01-02T03:04:05Z" {
		t.Errorf("createdAt = %v, muốn giữ nguyên 2026-01-02T03:04:05Z", a["createdAt"])
	}
	if a["updatedAt"] == "2026-01-02T03:04:05Z" {
		t.Error("updatedAt không đổi sau khi sửa bài")
	}
	if res := put(jakeToken, "bai-cua-jake", `{"article":{"tagList":[]}}`); len(stringList(t, articleOf(t, res)["tagList"], "tagList")) != 0 {
		t.Errorf("tagList []: %s, muốn xóa hết tag", res.Body)
	}
	if res := put(jakeToken, "bai-cua-jake", `{"article":{"tagList":null}}`); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("tagList null: %d %s, muốn 422", res.Status, res.Body)
	}
	if res := put(annaToken, "bai-cua-jake", `{"article":{"body":"hijacked"}}`); res.Status != http.StatusForbidden || errorOf(res, "article") != "forbidden" {
		t.Errorf("người khác sửa: %d %s, muốn 403 article forbidden", res.Status, res.Body)
	}
	if res := put(jakeToken, "khong-co", `{"article":{"body":"x"}}`); res.Status != http.StatusNotFound || errorOf(res, "article") != "not found" {
		t.Errorf("slug không có: %d %s, muốn 404 article not found", res.Status, res.Body)
	}
}

func TestDeleteArticle(t *testing.T) {
	e, db := newTestServer(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	a := testutil.CreateArticle(t, db, jake, "bai-cua-jake", testutil.BaseTime, "go")
	testutil.CreateComment(t, db, a, anna, "hay")

	res := testutil.Do(t, e, http.MethodDelete, "/api/articles/bai-cua-jake", testutil.WithToken(testutil.Token(t, anna.ID)))
	if res.Status != http.StatusForbidden || errorOf(res, "article") != "forbidden" {
		t.Errorf("người khác xóa: %d %s, muốn 403 article forbidden", res.Status, res.Body)
	}
	res = testutil.Do(t, e, http.MethodDelete, "/api/articles/bai-cua-jake", testutil.WithToken(testutil.Token(t, jake.ID)))
	if res.Status != http.StatusNoContent || res.Body != "" {
		t.Fatalf("tác giả xóa: %d %q, muốn 204 không có body (c.NoContent)", res.Status, res.Body)
	}
	if res := testutil.Do(t, e, http.MethodGet, "/api/articles/bai-cua-jake"); res.Status != http.StatusNotFound {
		t.Errorf("GET sau khi xóa: %d, muốn 404", res.Status)
	}
	var n int64
	db.Model(&models.Comment{}).Count(&n)
	if n != 0 {
		t.Errorf("còn %d comment của bài đã xóa", n)
	}
}
