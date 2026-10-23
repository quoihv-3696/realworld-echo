package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type ArticleHandler struct {
	svc *services.ArticleService
}

func NewArticleHandler(svc *services.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

func (h *ArticleHandler) List(c *echo.Context) error {
	res, err := h.svc.List(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ArticleHandler) Get(c *echo.Context) error {
	res, err := h.svc.Get(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ArticleHandler) Create(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	var req dto.CreateArticleRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Create(c.Request().Context(), viewerID, req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, res)
}

func (h *ArticleHandler) Update(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	var req dto.UpdateArticleRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Update(c.Request().Context(), viewerID, c.Param("slug"), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ArticleHandler) Delete(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	if err := h.svc.Delete(c.Request().Context(), viewerID, c.Param("slug")); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
