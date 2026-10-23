package server_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

const jakeJSON = `{"user":{"username":"jake","email":"jake@test.local","password":"password123"}}`

func userOf(t *testing.T, res testutil.Response) map[string]any {
	t.Helper()
	u, ok := res.JSON["user"].(map[string]any)
	if !ok {
		t.Fatalf("body phải có dạng {\"user\": {...}}. Status %d, body: %s", res.Status, res.Body)
	}
	return u
}

func errorOf(res testutil.Response, field string) string {
	errs, _ := res.JSON["errors"].(map[string]any)
	msgs, _ := errs[field].([]any)
	if len(msgs) == 0 {
		return ""
	}
	s, _ := msgs[0].(string)
	return s
}

func registerJake(t *testing.T, e *echo.Echo) string {
	t.Helper()
	res := testutil.Do(t, e, http.MethodPost, "/api/users", testutil.WithJSON(jakeJSON))
	if res.Status != http.StatusCreated {
		t.Fatalf("POST /api/users: status %d, muốn 201. Body: %s", res.Status, res.Body)
	}
	token, _ := userOf(t, res)["token"].(string)
	return token
}

func TestRegister_CreatedWithSpecShape(t *testing.T) {
	e, _ := newTestServer(t)
	res := testutil.Do(t, e, http.MethodPost, "/api/users", testutil.WithJSON(jakeJSON))
	if res.Status != http.StatusCreated {
		t.Fatalf("status = %d, muốn 201 Created. Body: %s", res.Status, res.Body)
	}
	u := userOf(t, res)
	keys := make([]string, 0, len(u))
	for k := range u {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if want := []string{"bio", "email", "image", "token", "username"}; !slices.Equal(keys, want) {
		t.Errorf("các key của user = %v, muốn đúng %v (không bao giờ trả password)", keys, want)
	}
	if u["username"] != "jake" || u["email"] != "jake@test.local" || u["bio"] != nil || u["image"] != nil {
		t.Errorf("user = %v, muốn jake, jake@test.local, bio null, image null", u)
	}
	if tok, _ := u["token"].(string); tok == "" {
		t.Error("token rỗng")
	}
}

func TestRegister_Errors(t *testing.T) {
	e, _ := newTestServer(t)
	registerJake(t, e)

	res := testutil.Do(t, e, http.MethodPost, "/api/users", testutil.WithJSON(`{"user":{"username":"","email":"a@test.local","password":"password123"}}`))
	if res.Status != http.StatusUnprocessableEntity || errorOf(res, "username") != "can't be blank" {
		t.Errorf("username rỗng: %d %s, muốn 422 {\"errors\":{\"username\":[\"can't be blank\"]}}", res.Status, res.Body)
	}
	res = testutil.Do(t, e, http.MethodPost, "/api/users", testutil.WithJSON(`{"user":{"username":"khac","email":"jake@test.local","password":"password123"}}`))
	if res.Status != http.StatusConflict || errorOf(res, "email") != "has already been taken" {
		t.Errorf("email trùng: %d %s, muốn 409 {\"errors\":{\"email\":[\"has already been taken\"]}}", res.Status, res.Body)
	}
	res = testutil.Do(t, e, http.MethodPost, "/api/users", testutil.WithJSON(`{"user":`))
	if res.Status != http.StatusBadRequest || errorOf(res, "body") != "Bad Request" {
		t.Errorf("JSON hỏng: %d %s, muốn 400 {\"errors\":{\"body\":[\"Bad Request\"]}} (trả nguyên lỗi của c.Bind)", res.Status, res.Body)
	}
}

func TestLoginAndCurrentUser(t *testing.T) {
	e, _ := newTestServer(t)
	registerJake(t, e)

	res := testutil.Do(t, e, http.MethodPost, "/api/users/login", testutil.WithJSON(`{"user":{"email":"jake@test.local","password":"password123"}}`))
	if res.Status != http.StatusOK {
		t.Fatalf("login: status %d, muốn 200. Body: %s", res.Status, res.Body)
	}
	token, _ := userOf(t, res)["token"].(string)

	res = testutil.Do(t, e, http.MethodGet, "/api/user", testutil.WithToken(token))
	if res.Status != http.StatusOK || userOf(t, res)["username"] != "jake" {
		t.Fatalf("GET /api/user: %d %s, muốn 200 với user jake", res.Status, res.Body)
	}

	res = testutil.Do(t, e, http.MethodPost, "/api/users/login", testutil.WithJSON(`{"user":{"email":"jake@test.local","password":"wrongpassword"}}`))
	if res.Status != http.StatusUnauthorized || errorOf(res, "credentials") != "invalid" {
		t.Errorf("sai mật khẩu: %d %s, muốn 401 {\"errors\":{\"credentials\":[\"invalid\"]}}", res.Status, res.Body)
	}
}

func TestCurrentUser_Unauthorized(t *testing.T) {
	e, _ := newTestServer(t)
	res := testutil.Do(t, e, http.MethodGet, "/api/user")
	if res.Status != http.StatusUnauthorized || errorOf(res, "token") != "is missing" {
		t.Errorf("không có token: %d %s, muốn 401 {\"errors\":{\"token\":[\"is missing\"]}}", res.Status, res.Body)
	}
	res = testutil.Do(t, e, http.MethodPut, "/api/user", testutil.WithToken("abc.def.ghi"), testutil.WithJSON(`{"user":{"bio":"x"}}`))
	if res.Status != http.StatusUnauthorized || errorOf(res, "token") != "is invalid" {
		t.Errorf("token rác: %d %s, muốn 401 {\"errors\":{\"token\":[\"is invalid\"]}}", res.Status, res.Body)
	}
}

func TestUpdateUser(t *testing.T) {
	e, _ := newTestServer(t)
	token := registerJake(t, e)
	put := func(body string) testutil.Response {
		t.Helper()
		return testutil.Do(t, e, http.MethodPut, "/api/user", testutil.WithToken(token), testutil.WithJSON(body))
	}

	if res := put(`{"user":{"bio":"Updated bio"}}`); res.Status != http.StatusOK || userOf(t, res)["bio"] != "Updated bio" {
		t.Fatalf("đặt bio: %d %s", res.Status, res.Body)
	}
	if res := put(`{"user":{"image":"https://example.com/a.jpg"}}`); userOf(t, res)["bio"] != "Updated bio" {
		t.Errorf("chỉ gửi image mà bio bị đổi: %s (field không gửi thì giữ nguyên)", res.Body)
	}
	if res := put(`{"user":{"bio":null}}`); userOf(t, res)["bio"] != nil {
		t.Errorf("bio null: %s, muốn bio null", res.Body)
	}
	if res := put(`{"user":{"image":""}}`); userOf(t, res)["image"] != nil {
		t.Errorf("image \"\": %s, muốn image null", res.Body)
	}
	res := testutil.Do(t, e, http.MethodGet, "/api/user", testutil.WithToken(token))
	if u := userOf(t, res); u["bio"] != nil || u["image"] != nil {
		t.Errorf("GET sau khi cập nhật: %s, muốn bio và image đã lưu là null", res.Body)
	}
	if res := put(`{"user":{"username":null}}`); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("username null: %d %s, muốn 422", res.Status, res.Body)
	}
	if res := put(`{"user":{"password":"short7c"}}`); res.Status != http.StatusUnprocessableEntity {
		t.Errorf("mật khẩu 7 ký tự: %d %s, muốn 422", res.Status, res.Body)
	}
}
