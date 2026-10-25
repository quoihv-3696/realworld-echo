package handlers

import (
	"bytes"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/api"
)

const publicServer = "https://api.realworld.show/api"

var localSpec = bytes.Replace(api.Spec, []byte(publicServer), []byte("/api"), 1)

func OpenAPISpec(c *echo.Context) error {
	return c.Blob(http.StatusOK, "application/yaml", localSpec)
}

func SwaggerUI(c *echo.Context) error {
	return c.HTMLBlob(http.StatusOK, api.SwaggerPage)
}
