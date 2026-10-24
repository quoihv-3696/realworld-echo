package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type ProfileHandler struct {
	svc *services.ProfileService
}

func NewProfileHandler(svc *services.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

func (h *ProfileHandler) Get(c *echo.Context) error {
	viewerID, _ := auth.UserID(c)
	res, err := h.svc.Get(c.Request().Context(), viewerID, c.Param("username"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ProfileHandler) Follow(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	res, err := h.svc.Follow(c.Request().Context(), viewerID, c.Param("username"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *ProfileHandler) Unfollow(c *echo.Context) error {
	viewerID, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	res, err := h.svc.Unfollow(c.Request().Context(), viewerID, c.Param("username"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}
