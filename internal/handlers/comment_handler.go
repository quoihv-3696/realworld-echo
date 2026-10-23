package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type CommentHandler struct {
	svc *services.CommentService
}

func NewCommentHandler(svc *services.CommentService) *CommentHandler {
	return &CommentHandler{svc: svc}
}

func (h *CommentHandler) List(c *echo.Context) error {
	res, err := h.svc.List(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *CommentHandler) Create(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	var req dto.CreateCommentRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Create(c.Request().Context(), viewerID, c.Param("slug"), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, res)
}

func (h *CommentHandler) Delete(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	id, err := echo.PathParam[uint](c, "id")
	if err != nil {
		return err
	}
	if err := h.svc.Delete(c.Request().Context(), viewerID, c.Param("slug"), id); err != nil {
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
