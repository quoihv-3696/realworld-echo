package services

import (
	"context"
	"errors"
	"strings"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/slug"
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
	a, err := findArticle(ctx, s.repo, slug)
	if err != nil {
		return dto.SingleArticleResponse{}, err
	}
	return dto.SingleArticleResponse{Article: toArticle(*a)}, nil
}

const maxSlugAttempts = 3

func (s *ArticleService) Create(ctx context.Context, authorID uint, req dto.CreateArticleRequest) (dto.SingleArticleResponse, error) {
	in := req.Article
	switch {
	case in.Title == "":
		return dto.SingleArticleResponse{}, apperr.Blank("title")
	case in.Description == "":
		return dto.SingleArticleResponse{}, apperr.Blank("description")
	case in.Body == "":
		return dto.SingleArticleResponse{}, apperr.Blank("body")
	}

	a := models.Article{Title: in.Title, Description: in.Description, Body: in.Body, AuthorID: authorID}
	tags := normalizeTags(in.TagList)
	base := slug.Make(in.Title)
	a.Slug = base
	for attempt := 1; ; attempt++ {
		err := s.repo.Create(ctx, &a, tags)
		var dup *repository.DuplicateError
		if errors.As(err, &dup) && dup.Field == "slug" && attempt < maxSlugAttempts {
			a.Slug = slug.WithSuffix(base)
			continue
		}
		if err != nil {
			return dto.SingleArticleResponse{}, err
		}
		break
	}
	return s.Get(ctx, a.Slug)
}

func (s *ArticleService) Update(ctx context.Context, viewerID uint, articleSlug string, req dto.UpdateArticleRequest) (dto.SingleArticleResponse, error) {
	a, err := findOwnArticle(ctx, s.repo, viewerID, articleSlug)
	if err != nil {
		return dto.SingleArticleResponse{}, err
	}
	in := req.Article
	if in.Title.Set {
		if isBlank(in.Title) {
			return dto.SingleArticleResponse{}, apperr.Blank("title")
		}
		a.Title = *in.Title.Value
	}
	if in.Description.Set {
		if isBlank(in.Description) {
			return dto.SingleArticleResponse{}, apperr.Blank("description")
		}
		a.Description = *in.Description.Value
	}
	if in.Body.Set {
		if isBlank(in.Body) {
			return dto.SingleArticleResponse{}, apperr.Blank("body")
		}
		a.Body = *in.Body.Value
	}

	var tags *[]string
	if in.TagList.Set {
		if in.TagList.Value == nil {
			return dto.SingleArticleResponse{}, apperr.NotNull("tagList")
		}
		normalized := normalizeTags(*in.TagList.Value)
		tags = &normalized
	}
	if err := s.repo.Update(ctx, a, tags); err != nil {
		return dto.SingleArticleResponse{}, err
	}
	return s.Get(ctx, a.Slug)
}

func (s *ArticleService) Delete(ctx context.Context, viewerID uint, articleSlug string) error {
	a, err := findOwnArticle(ctx, s.repo, viewerID, articleSlug)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, a.ID)
}

func findArticle(ctx context.Context, repo repository.ArticleRepository, articleSlug string) (*models.Article, error) {
	a, err := repo.FindBySlug(ctx, articleSlug)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperr.NotFound("article")
	}
	return a, err
}

func findOwnArticle(ctx context.Context, repo repository.ArticleRepository, viewerID uint, articleSlug string) (*models.Article, error) {
	a, err := findArticle(ctx, repo, articleSlug)
	if err != nil {
		return nil, err
	}
	if a.AuthorID != viewerID {
		return nil, apperr.Forbidden("article")
	}
	return a, nil
}

func normalizeTags(in []string) []string {
	out := make([]string, 0, len(in))
	seen := make(map[string]bool, len(in))
	for _, t := range in {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}
