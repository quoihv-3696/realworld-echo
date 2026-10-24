package repository

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

type FavoriteRepository interface {
	Favorite(ctx context.Context, userID, articleID uint) error
	Unfavorite(ctx context.Context, userID, articleID uint) error
	CountByArticle(ctx context.Context, articleIDs []uint) (map[uint]int64, error)
	FavoritedAmong(ctx context.Context, userID uint, articleIDs []uint) (map[uint]bool, error)
}

type favoriteRepository struct {
	db *gorm.DB
}

func NewFavoriteRepository(db *gorm.DB) FavoriteRepository {
	return &favoriteRepository{db: db}
}

func (r *favoriteRepository) Favorite(ctx context.Context, userID, articleID uint) error {
	f := models.Favorite{UserID: userID, ArticleID: articleID}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Omit("User", "Article").
		Create(&f).Error
}

func (r *favoriteRepository) Unfavorite(ctx context.Context, userID, articleID uint) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND article_id = ?", userID, articleID).
		Delete(&models.Favorite{}).Error
}

func (r *favoriteRepository) CountByArticle(ctx context.Context, articleIDs []uint) (map[uint]int64, error) {
	out := make(map[uint]int64, len(articleIDs))
	if len(articleIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ArticleID uint
		Total     int64
	}
	err := r.db.WithContext(ctx).
		Model(&models.Favorite{}).
		Select("article_id, COUNT(*) AS total").
		Where("article_id IN ?", articleIDs).
		Group("article_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ArticleID] = row.Total
	}
	return out, nil
}

func (r *favoriteRepository) FavoritedAmong(ctx context.Context, userID uint, articleIDs []uint) (map[uint]bool, error) {
	out := make(map[uint]bool, len(articleIDs))
	if len(articleIDs) == 0 {
		return out, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).
		Model(&models.Favorite{}).
		Where("user_id = ? AND article_id IN ?", userID, articleIDs).
		Pluck("article_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
