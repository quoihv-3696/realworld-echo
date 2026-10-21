package server

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/handlers"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

func New(db *gorm.DB) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = apperr.Handler
	e.Use(middleware.RequestLogger(), middleware.Recover())

	tags := handlers.NewTagHandler(services.NewTagService(repository.NewTagRepository(db)))
	articles := handlers.NewArticleHandler(services.NewArticleService(repository.NewArticleRepository(db)))

	api := e.Group("/api")
	api.GET("/tags", tags.List)
	api.GET("/articles", articles.List)
	api.GET("/articles/:slug", articles.Get)

	return e
}
