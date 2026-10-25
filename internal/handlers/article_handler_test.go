package handlers_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/handlers"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func newContext(method, target, body string) (*echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	return echo.New().NewContext(req, rec), rec
}

func TestFeed_WithoutUser_ReturnsTokenMissing(t *testing.T) {
	h := handlers.NewArticleHandler(nil)
	c, rec := newContext(http.MethodGet, "/api/articles/feed", "")

	err := h.Feed(c)

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Status != http.StatusUnauthorized || appErr.Fields["token"][0] != "is missing" {
		t.Fatalf("err = %v, muốn apperr 401 {token: [is missing]}", err)
	}
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Errorf("gọi thẳng handler thì lỗi chưa được ghi ra response (error handler không chạy): code %d, body %q", rec.Code, rec.Body.String())
	}
}

func TestList_InvalidLimit_FailsBeforeService(t *testing.T) {
	h := handlers.NewArticleHandler(nil)
	c, _ := newContext(http.MethodGet, "/api/articles?limit=abc", "")

	err := h.List(c)

	if err == nil || echo.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("?limit=abc: err = %v (status %d), muốn lỗi 400 từ QueryParamOr, trước khi chạm service", err, echo.StatusCode(err))
	}
}

func TestUpdate_WithoutRequireOwner_IsServerError(t *testing.T) {
	h := handlers.NewArticleHandler(nil)
	c, rec := newContext(http.MethodPut, "/api/articles/a1", `{"article":{"title":"x"}}`)
	c.SetPathValues(echo.PathValues{{Name: "slug", Value: "a1"}})
	auth.SetUserID(c, 1)

	err := h.Update(c)

	var appErr *apperr.Error
	if err == nil || errors.As(err, &appErr) {
		t.Fatalf("err = %v, muốn lỗi thường (lỗi lập trình: route quên RequireOwner), không phải apperr", err)
	}
	apperr.Handler(c, err)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("qua error handler: %d, muốn 500", rec.Code)
	}
}

func TestRequireOwner_OnlyOwnerReachesHandler(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	testutil.CreateArticle(t, db, jake, "a1", testutil.BaseTime)
	svc := services.NewArticleService(repository.NewArticleRepository(db), repository.NewFollowRepository(db), repository.NewFavoriteRepository(db), nil)
	h := handlers.NewArticleHandler(svc)

	c, rec := newContext(http.MethodPut, "/api/articles/a1", `{"article":{"title":"Tiêu đề mới"}}`)
	c.SetPathValues(echo.PathValues{{Name: "slug", Value: "a1"}})
	auth.SetUserID(c, jake.ID)
	if err := h.RequireOwner(h.Update)(c); err != nil {
		t.Fatalf("tác giả sửa bài: %v", err)
	}
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Tiêu đề mới") {
		t.Errorf("tác giả sửa bài: %d %s, muốn 200 kèm tiêu đề mới", rec.Code, rec.Body.String())
	}

	mustNotRun := func(c *echo.Context) error {
		t.Error("handler không được chạy khi RequireOwner từ chối")
		return nil
	}
	cases := []struct {
		name   string
		slug   string
		user   uint
		status int
	}{
		{"người khác", "a1", anna.ID, http.StatusForbidden},
		{"bài không có", "khong-co", jake.ID, http.StatusNotFound},
	}
	for _, tc := range cases {
		c, _ := newContext(http.MethodPut, "/api/articles/"+tc.slug, `{"article":{"title":"x"}}`)
		c.SetPathValues(echo.PathValues{{Name: "slug", Value: tc.slug}})
		auth.SetUserID(c, tc.user)
		err := h.RequireOwner(mustNotRun)(c)
		if echo.StatusCode(err) != tc.status {
			t.Errorf("%s: err = %v (status %d), muốn %d", tc.name, err, echo.StatusCode(err), tc.status)
		}
	}
}
