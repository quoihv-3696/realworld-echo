package services

import (
	"context"
	"errors"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

type ProfileService struct {
	users repository.UserRepository
}

func NewProfileService(users repository.UserRepository) *ProfileService {
	return &ProfileService{users: users}
}

func (s *ProfileService) Get(ctx context.Context, username string) (dto.ProfileResponse, error) {
	u, err := s.users.FindByUsername(ctx, username)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.ProfileResponse{}, apperr.NotFound("profile")
	}
	if err != nil {
		return dto.ProfileResponse{}, err
	}
	return dto.ProfileResponse{Profile: toProfile(*u)}, nil
}
