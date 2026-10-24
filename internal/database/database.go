package database

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

func Open(dsn string, logLevel logger.LogLevel) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:         logger.Default.LogMode(logLevel),
		TranslateError: true,
	})
}

func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&models.User{}, &models.Tag{}, &models.Article{}, &models.Comment{}, &models.Follow{}, &models.Favorite{})
}
