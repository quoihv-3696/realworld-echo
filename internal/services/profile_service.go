package services

import (
	"context"
	"errors"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

type ProfileService struct {
	users   repository.UserRepository
	follows repository.FollowRepository
}

func NewProfileService(users repository.UserRepository, follows repository.FollowRepository) *ProfileService {
	return &ProfileService{users: users, follows: follows}
}

func (s *ProfileService) Get(ctx context.Context, viewerID uint, username string) (dto.ProfileResponse, error) {
	u, err := s.findProfile(ctx, username)
	if err != nil {
		return dto.ProfileResponse{}, err
	}
	following := false
	if viewerID != 0 && viewerID != u.ID {
		following, err = s.follows.IsFollowing(ctx, viewerID, u.ID)
		if err != nil {
			return dto.ProfileResponse{}, err
		}
	}
	return profileResponse(*u, following), nil
}

func (s *ProfileService) Follow(ctx context.Context, viewerID uint, username string) (dto.ProfileResponse, error) {
	u, err := s.findProfile(ctx, username)
	if err != nil {
		return dto.ProfileResponse{}, err
	}
	if u.ID == viewerID {
		return dto.ProfileResponse{}, apperr.Invalid("profile", "cannot follow yourself")
	}
	if err := s.follows.Follow(ctx, viewerID, u.ID); err != nil {
		return dto.ProfileResponse{}, err
	}
	return profileResponse(*u, true), nil
}

func (s *ProfileService) Unfollow(ctx context.Context, viewerID uint, username string) (dto.ProfileResponse, error) {
	u, err := s.findProfile(ctx, username)
	if err != nil {
		return dto.ProfileResponse{}, err
	}
	if err := s.follows.Unfollow(ctx, viewerID, u.ID); err != nil {
		return dto.ProfileResponse{}, err
	}
	return profileResponse(*u, false), nil
}

func (s *ProfileService) findProfile(ctx context.Context, username string) (*models.User, error) {
	u, err := s.users.FindByUsername(ctx, username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperr.NotFound("profile")
	}
	return u, err
}

func profileResponse(u models.User, following bool) dto.ProfileResponse {
	p := toProfile(u)
	p.Following = following
	return dto.ProfileResponse{Profile: p}
}
