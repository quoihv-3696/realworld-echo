package apperr

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

type body struct {
	Errors map[string][]string `json:"errors"`
}

func Handler(c *echo.Context, err error) {
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}

	var appErr *Error
	if errors.As(err, &appErr) {
		_ = c.JSON(appErr.Status, body{Errors: appErr.Fields})
		return
	}

	status := echo.StatusCode(err)
	if status == 0 {
		status = http.StatusInternalServerError
		c.Logger().Error("lỗi không mong đợi", "error", err)
	}

	_ = c.JSON(status, body{Errors: map[string][]string{"body": {http.StatusText(status)}}})
}
