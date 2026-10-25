package services

import (
	"context"

	"github.com/quoihv-3696/realworld-echo/internal/cache"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

type TagService struct {
	repo  repository.TagRepository
	cache *cache.Store
}

func NewTagService(repo repository.TagRepository, cache *cache.Store) *TagService {
	return &TagService{repo: repo, cache: cache}
}

func (s *TagService) List(ctx context.Context) (dto.TagsResponse, error) {
	return s.cache.Remember(ctx, "tags", func() (dto.TagsResponse, error) {
		return s.list(ctx)
	})
}

func (s *TagService) list(ctx context.Context) (dto.TagsResponse, error) {
	tags, err := s.repo.ListNames(ctx)
	if err != nil {
		return dto.TagsResponse{}, err
	}
	if tags == nil {
		tags = []string{}
	}
	return dto.TagsResponse{
		Tags: tags,
	}, nil
}
