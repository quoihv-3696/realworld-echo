package handlers

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type UserHandler struct {
	svc *services.UserService
}

func NewUserHandler(svc *services.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Register(c *echo.Context) error {
	var req dto.RegisterRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, res)
}

func (h *UserHandler) Login(c *echo.Context) error {
	var req dto.LoginRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *UserHandler) Current(c *echo.Context) error {
	id, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	res, err := h.svc.Current(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}

func (h *UserHandler) Update(c *echo.Context) error {
	id, ok := auth.UserID(c)
	if !ok {
		return apperr.TokenMissing()
	}
	var req dto.UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	res, err := h.svc.Update(c.Request().Context(), id, req)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, res)
}
