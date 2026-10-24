package handlers

import (
	"errors"
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
	q, err := articleQuery(c)
	if err != nil {
		return err
	}
	viewerID, _ := auth.UserID(c)
	res, err := h.svc.List(c.Request().Context(), viewerID, q)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ArticleHandler) Get(c *echo.Context) error {
	viewerID, _ := auth.UserID(c)
	res, err := h.svc.Get(c.Request().Context(), viewerID, c.Param("slug"))
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
	a, ok := articleFromContext(c)
	if !ok {
		return errors.New("route PUT bài viết thiếu middleware RequireOwner")
	}
	var req dto.UpdateArticleRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Update(c.Request().Context(), a, req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ArticleHandler) Delete(c *echo.Context) error {
	a, ok := articleFromContext(c)
	if !ok {
		return errors.New("route DELETE bài viết thiếu middleware RequireOwner")
	}
	if err := h.svc.Delete(c.Request().Context(), a); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *ArticleHandler) Feed(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	q, err := articleQuery(c)
	if err != nil {
		return err
	}
	res, err := h.svc.Feed(c.Request().Context(), viewerID, q)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func articleQuery(c *echo.Context) (services.ArticleQuery, error) {
	limit, err := echo.QueryParamOr[int](c, "limit", services.DefaultLimit)
	if err != nil {
		return services.ArticleQuery{}, err
	}
	offset, err := echo.QueryParamOr[int](c, "offset", 0)
	if err != nil {
		return services.ArticleQuery{}, err
	}
	return services.ArticleQuery{
		Tag:    c.QueryParam("tag"),
		Author: c.QueryParam("author"),
		Limit:  limit,
		Offset: offset,
	}, nil
}
