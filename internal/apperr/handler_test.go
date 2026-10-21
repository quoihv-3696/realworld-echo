package apperr_test

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func newEcho(t *testing.T, err error) *echo.Echo {
	t.Helper()
	e := echo.New()
	e.Logger = testutil.Logger(t)
	e.HTTPErrorHandler = apperr.Handler
	e.GET("/boom", func(c *echo.Context) error { return err })
	return e
}

func firstError(t *testing.T, res testutil.Response, field string) string {
	t.Helper()
	errs, ok := res.JSON["errors"].(map[string]any)
	if !ok {
		t.Fatalf("body phải có dạng {\"errors\": {...}}, nhận: %s\n(viết apperr.Handler trong internal/apperr/handler.go, guide ngày 1 bước 5)", res.Body)
	}
	msgs, ok := errs[field].([]any)
	if !ok || len(msgs) == 0 {
		t.Fatalf("errors.%s phải là mảng chuỗi, nhận: %s", field, res.Body)
	}
	return fmt.Sprint(msgs[0])
}

func TestHandler_AppError(t *testing.T) {
	res := testutil.Do(t, newEcho(t, apperr.NotFound("article")), http.MethodGet, "/boom")
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, muốn 404. Body: %s", res.Status, res.Body)
	}
	if got := firstError(t, res, "article"); got != "not found" {
		t.Errorf("errors.article[0] = %q, muốn \"not found\"", got)
	}
}

func TestHandler_WrappedAppError(t *testing.T) {
	err := fmt.Errorf("lấy bài viết: %w", apperr.Forbidden("article"))
	res := testutil.Do(t, newEcho(t, err), http.MethodGet, "/boom")
	if res.Status != http.StatusForbidden {
		t.Fatalf("status = %d, muốn 403 (gợi ý: errors.As, không ép kiểu trực tiếp). Body: %s", res.Status, res.Body)
	}
	if got := firstError(t, res, "article"); got != "forbidden" {
		t.Errorf("errors.article[0] = %q, muốn \"forbidden\"", got)
	}
}

func TestHandler_EchoSentinelKeepsStatus(t *testing.T) {
	res := testutil.Do(t, newEcho(t, echo.ErrNotFound), http.MethodGet, "/boom")
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, muốn 404. echo.ErrNotFound KHÔNG phải *echo.HTTPError ở v5, dùng echo.StatusCode(err). Body: %s", res.Status, res.Body)
	}
	firstError(t, res, "body")
}

func TestHandler_EchoHTTPError(t *testing.T) {
	res := testutil.Do(t, newEcho(t, echo.NewHTTPError(http.StatusBadRequest, "bad json")), http.MethodGet, "/boom")
	if res.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, muốn 400. Body: %s", res.Status, res.Body)
	}
	firstError(t, res, "body")
}

func TestHandler_UnknownErrorHidesDetails(t *testing.T) {
	res := testutil.Do(t, newEcho(t, errors.New("pq: password=secret")), http.MethodGet, "/boom")
	if res.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, muốn 500. Body: %s", res.Status, res.Body)
	}
	if got := firstError(t, res, "body"); got != http.StatusText(http.StatusInternalServerError) {
		t.Errorf("errors.body[0] = %q, muốn %q (không lộ lỗi nội bộ cho client)", got, http.StatusText(http.StatusInternalServerError))
	}
}

func TestHandler_UnknownRoute(t *testing.T) {
	res := testutil.Do(t, newEcho(t, nil), http.MethodGet, "/khong-co-route")
	if res.Status != http.StatusNotFound {
		t.Fatalf("status = %d, muốn 404. Body: %s", res.Status, res.Body)
	}
	firstError(t, res, "body")
}

func TestHandler_CommittedResponseUntouched(t *testing.T) {
	e := echo.New()
	e.Logger = testutil.Logger(t)
	e.HTTPErrorHandler = apperr.Handler
	e.GET("/done", func(c *echo.Context) error {
		if err := c.JSON(http.StatusCreated, map[string]string{"ok": "yes"}); err != nil {
			return err
		}
		return errors.New("lỗi sau khi đã trả response")
	})
	res := testutil.Do(t, e, http.MethodGet, "/done")
	if res.Status != http.StatusCreated || res.JSON["errors"] != nil {
		t.Fatalf("response đã gửi thì Handler không được ghi thêm. status=%d body=%s", res.Status, res.Body)
	}
}
