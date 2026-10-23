package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type ProfileHandler struct {
	svc *services.ProfileService
}

func NewProfileHandler(svc *services.ProfileService) *ProfileHandler {
	return &ProfileHandler{svc: svc}
}

func (h *ProfileHandler) Get(c *echo.Context) error {
	res, err := h.svc.Get(c.Request().Context(), c.Param("username"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}
