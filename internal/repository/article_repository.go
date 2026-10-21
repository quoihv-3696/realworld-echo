package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

type ArticleFilter struct {
	Limit  int
	Offset int
}

type ArticleRepository interface {
	List(ctx context.Context, f ArticleFilter) ([]models.Article, int64, error)
	FindBySlug(ctx context.Context, slug string) (*models.Article, error)
}

type articleRepository struct {
	db *gorm.DB
}

func NewArticleRepository(db *gorm.DB) ArticleRepository {
	return &articleRepository{db: db}
}

func (r *articleRepository) List(ctx context.Context, f ArticleFilter) ([]models.Article, int64, error) {
	db := r.db.WithContext(ctx)

	var total int64
	if err := db.Model(&models.Article{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var articles []models.Article
	err := db.Preload("Author").
		Preload("Tags").
		Order("created_at DESC, id DESC").
		Limit(f.Limit).
		Offset(f.Offset).
		Find(&articles).Error
	if err != nil {
		return nil, 0, err
	}
	return articles, total, nil
}

func (r *articleRepository) FindBySlug(ctx context.Context, slug string) (*models.Article, error) {
	var article models.Article
	err := r.db.WithContext(ctx).
		Preload("Author").
		Preload("Tags").
		Where("slug = ?", slug).
		First(&article).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &article, nil
}
