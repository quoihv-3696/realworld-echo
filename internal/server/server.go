package server

import (
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/handlers"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

func New(db *gorm.DB, jwt *auth.JWT) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = apperr.Handler
	e.Use(middleware.RequestLogger(), middleware.Recover())

	userRepo := repository.NewUserRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	tags := handlers.NewTagHandler(services.NewTagService(repository.NewTagRepository(db)))
	articles := handlers.NewArticleHandler(services.NewArticleService(articleRepo))
	comments := handlers.NewCommentHandler(services.NewCommentService(articleRepo, commentRepo))
	users := handlers.NewUserHandler(services.NewUserService(userRepo, jwt))
	profiles := handlers.NewProfileHandler(services.NewProfileService(userRepo))

	api := e.Group("/api")
	api.GET("/tags", tags.List)

	api.GET("/articles", articles.List)
	api.POST("/articles", articles.Create, jwt.Required())
	api.GET("/articles/:slug", articles.Get)
	api.PUT("/articles/:slug", articles.Update, jwt.Required())
	api.DELETE("/articles/:slug", articles.Delete, jwt.Required())

	api.GET("/articles/:slug/comments", comments.List, jwt.Optional())
	api.POST("/articles/:slug/comments", comments.Create, jwt.Required())
	api.DELETE("/articles/:slug/comments/:id", comments.Delete, jwt.Required())

	api.POST("/users", users.Register)
	api.POST("/users/login", users.Login)
	api.GET("/user", users.Current, jwt.Required())
	api.PUT("/user", users.Update, jwt.Required())

	api.GET("/profiles/:username", profiles.Get, jwt.Optional())

	return e
}
