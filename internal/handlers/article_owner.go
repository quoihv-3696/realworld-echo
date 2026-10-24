package handlers

import (
	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/models"
)

const articleKey = "article"

func (h *ArticleHandler) RequireOwner(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		viewerID, ok := auth.UserID(c)
		if !ok {
			return apperr.TokenMissing()
		}
		a, err := h.svc.Owned(c.Request().Context(), viewerID, c.Param("slug"))
		if err != nil {
			return err
		}
		c.Set(articleKey, a)
		return next(c)
	}
}

func articleFromContext(c *echo.Context) (*models.Article, bool) {
	a, err := echo.ContextGet[*models.Article](c, articleKey)
	return a, err == nil
}
