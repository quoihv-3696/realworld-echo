package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

type CommentRepository interface {
	Create(ctx context.Context, c *models.Comment) error
	ListByArticle(ctx context.Context, articleID uint) ([]models.Comment, error)
	FindByID(ctx context.Context, id uint) (*models.Comment, error)
	Delete(ctx context.Context, id uint) error
}

type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

func (r *commentRepository) Create(ctx context.Context, c *models.Comment) error {
	db := r.db.WithContext(ctx)
	if err := db.Omit("Author").Create(c).Error; err != nil {
		return err
	}
	return db.Preload("Author").First(c, c.ID).Error
}

func (r *commentRepository) ListByArticle(ctx context.Context, articleID uint) ([]models.Comment, error) {
	var comments []models.Comment
	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("article_id = ?", articleID).
		Order("created_at, id").
		Find(&comments).Error
	return comments, err
}

func (r *commentRepository) FindByID(ctx context.Context, id uint) (*models.Comment, error) {
	var c models.Comment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *commentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Comment{}, id).Error
}
