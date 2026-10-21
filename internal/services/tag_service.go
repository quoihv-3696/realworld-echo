package services

import (
	"context"

	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

type TagService struct {
	repo repository.TagRepository
}

func NewTagService(repo repository.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) List(ctx context.Context) (dto.TagsResponse, error) {
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
