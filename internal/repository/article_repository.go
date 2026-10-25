package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/quoihv-3696/realworld-echo/internal/models"
)

type ArticleFilter struct {
	Limit       int
	Offset      int
	Tag         string
	Author      string
	FavoritedBy string
	FollowedBy  uint
	Sort        string
}

const (
	SortNewest  = "newest"
	SortOldest  = "oldest"
	SortPopular = "popular"
)

var articleOrders = map[string]string{
	SortNewest:  "articles.created_at DESC, articles.id DESC",
	SortOldest:  "articles.created_at ASC, articles.id ASC",
	SortPopular: "(SELECT COUNT(*) FROM favorites WHERE favorites.article_id = articles.id) DESC, articles.created_at DESC, articles.id DESC",
}

func orderFor(sort string) string {
	if o, ok := articleOrders[sort]; ok {
		return o
	}
	return articleOrders[SortNewest]
}

type ArticleRepository interface {
	List(ctx context.Context, f ArticleFilter) ([]models.Article, int64, error)
	FindBySlug(ctx context.Context, slug string) (*models.Article, error)
	Create(ctx context.Context, a *models.Article, tagNames []string) error
	Update(ctx context.Context, a *models.Article, tagNames *[]string) error
	Delete(ctx context.Context, id uint) error
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
	if err := db.Model(&models.Article{}).Scopes(r.filter(f)).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var articles []models.Article
	err := db.Scopes(r.filter(f)).
		Preload("Author").
		Preload("Tags").
		Order(orderFor(f.Sort)).
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

var articleUniqueIndexes = map[string]string{
	"idx_articles_slug": "slug",
}

func (r *articleRepository) Create(ctx context.Context, a *models.Article, tagNames []string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tags, err := upsertTags(tx, tagNames)
		if err != nil {
			return err
		}
		a.Tags = tags
		return tx.Omit("Author", "Tags.*").Create(a).Error
	})
	return translateDuplicate(err, articleUniqueIndexes)
}

func (r *articleRepository) Update(ctx context.Context, a *models.Article, tagNames *[]string) error {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit(clause.Associations).Save(a).Error; err != nil {
			return err
		}
		if tagNames == nil {
			return nil
		}
		tags, err := upsertTags(tx, *tagNames)
		if err != nil {
			return err
		}
		a.Tags = tags
		return tx.Model(a).Association("Tags").Replace(tags)
	})
	return translateDuplicate(err, articleUniqueIndexes)
}

func (r *articleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Article{}, id).Error
}

func upsertTags(tx *gorm.DB, names []string) ([]models.Tag, error) {
	tags := make([]models.Tag, 0, len(names))
	if len(names) == 0 {
		return tags, nil
	}
	for _, n := range names {
		tags = append(tags, models.Tag{Name: n})
	}

	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&tags).Error; err != nil {
		return nil, err
	}

	var out []models.Tag
	if err := tx.Where("name IN ?", names).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *articleRepository) filter(f ArticleFilter) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if f.Tag != "" {
			db = db.Where("articles.id IN (?)", r.db.Table("article_tags").
				Select("article_tags.article_id").
				Joins("JOIN tags ON tags.id = article_tags.tag_id").
				Where("tags.name = ?", f.Tag))
		}
		if f.Author != "" {
			db = db.Where("articles.author_id IN (?)", r.db.Model(&models.User{}).
				Select("id").
				Where("username = ?", f.Author))
		}
		if f.FavoritedBy != "" {
			db = db.Where("articles.id IN (?)", r.db.Model(&models.Favorite{}).
				Select("favorites.article_id").
				Joins("JOIN users ON users.id = favorites.user_id").
				Where("users.username = ?", f.FavoritedBy))
		}
		if f.FollowedBy != 0 {
			db = db.Where("articles.author_id IN (?)", r.db.Model(&models.Follow{}).
				Select("following_id").
				Where("follower_id = ?", f.FollowedBy))
		}
		return db
	}
}
