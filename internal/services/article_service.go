package services

import (
	"context"
	"errors"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

const DefaultLimit = 20

type ArticleService struct {
	repo repository.ArticleRepository
}

func NewArticleService(repo repository.ArticleRepository) *ArticleService {
	return &ArticleService{repo: repo}
}

func (s *ArticleService) List(ctx context.Context) (dto.MultipleArticlesResponse, error) {
	articles, total, err := s.repo.List(ctx, repository.ArticleFilter{Limit: DefaultLimit})
	if err != nil {
		return dto.MultipleArticlesResponse{}, err
	}

	items := make([]dto.ArticleListItem, 0, len(articles))
	for _, a := range articles {
		items = append(items, toArticleListItem(a))
	}

	return dto.MultipleArticlesResponse{
		Articles:      items,
		ArticlesCount: total,
	}, nil

}

func (s *ArticleService) Get(ctx context.Context, slug string) (dto.SingleArticleResponse, error) {
	a, err := s.repo.FindBySlug(ctx, slug)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.SingleArticleResponse{}, apperr.NotFound("article")
	}
	if err != nil {
		return dto.SingleArticleResponse{}, err
	}
	return dto.SingleArticleResponse{Article: toArticle(*a)}, nil
}
