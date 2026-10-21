package repository

import (
	"context"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"gorm.io/gorm"
)

type TagRepository interface {
	ListNames(ctx context.Context) ([]string, error)
}

type tagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

func (r *tagRepository) ListNames(ctx context.Context) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).
		Model(&models.Tag{}).
		Order("name").
		Pluck("name", &names).
		Error
	if err != nil {
		return nil, err
	}
	return names, nil
}
