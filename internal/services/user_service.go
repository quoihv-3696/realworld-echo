package services

import (
	"context"
	"errors"
	"unicode/utf8"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/cache"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

type UserService struct {
	repo  repository.UserRepository
	jwt   *auth.JWT
	cache *cache.Store
}

func NewUserService(repo repository.UserRepository, jwt *auth.JWT, cache *cache.Store) *UserService {
	return &UserService{repo: repo, jwt: jwt, cache: cache}
}

func (s *UserService) Register(ctx context.Context, req dto.RegisterRequest) (dto.UserResponse, error) {
	in := req.User
	if in.Username == "" {
		return dto.UserResponse{}, apperr.Blank("username")
	}
	if in.Email == "" {
		return dto.UserResponse{}, apperr.Blank("email")
	}
	if err := validatePassword(in.Password); err != nil {
		return dto.UserResponse{}, err
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return dto.UserResponse{}, err
	}
	u := models.User{Username: in.Username, Email: in.Email, Password: hash}
	if err := s.repo.Create(ctx, &u); err != nil {
		return dto.UserResponse{}, takenOr(err)
	}
	return s.respond(u)
}

func (s *UserService) Login(ctx context.Context, req dto.LoginRequest) (dto.UserResponse, error) {
	in := req.User
	if in.Email == "" {
		return dto.UserResponse{}, apperr.Blank("email")
	}
	if in.Password == "" {
		return dto.UserResponse{}, apperr.Blank("password")
	}
	u, err := s.repo.FindByEmail(ctx, in.Email)
	if errors.Is(err, repository.ErrNotFound) {
		return dto.UserResponse{}, apperr.InvalidCredentials()
	}
	if err != nil {
		return dto.UserResponse{}, err
	}
	if !auth.CheckPassword(u.Password, in.Password) {
		return dto.UserResponse{}, apperr.InvalidCredentials()
	}
	return s.respond(*u)
}

func (s *UserService) Current(ctx context.Context, userID uint) (dto.UserResponse, error) {
	u, err := s.findCurrent(ctx, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	return s.respond(*u)
}

func (s *UserService) Update(ctx context.Context, userID uint, req dto.UpdateUserRequest) (dto.UserResponse, error) {
	u, err := s.findCurrent(ctx, userID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	in := req.User
	if in.Username.Set {
		if isBlank(in.Username) {
			return dto.UserResponse{}, apperr.Blank("username")
		}
		u.Username = *in.Username.Value
	}
	if in.Email.Set {
		if isBlank(in.Email) {
			return dto.UserResponse{}, apperr.Blank("email")
		}
		u.Email = *in.Email.Value
	}
	if in.Password.Set {
		if in.Password.Value == nil {
			return dto.UserResponse{}, apperr.Blank("password")
		}
		if err := validatePassword(*in.Password.Value); err != nil {
			return dto.UserResponse{}, err
		}
		hash, err := auth.HashPassword(*in.Password.Value)
		if err != nil {
			return dto.UserResponse{}, err
		}
		u.Password = hash
	}
	if in.Bio.Set {
		u.Bio = nullIfEmpty(in.Bio)
	}
	if in.Image.Set {
		u.Image = nullIfEmpty(in.Image)
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return dto.UserResponse{}, takenOr(err)
	}
	s.cache.Invalidate(ctx)
	return s.respond(*u)
}

func (s *UserService) findCurrent(ctx context.Context, userID uint) (*models.User, error) {
	u, err := s.repo.FindByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperr.TokenInvalid()
	}
	return u, err
}

func (s *UserService) respond(u models.User) (dto.UserResponse, error) {
	token, err := s.jwt.Sign(u.ID)
	if err != nil {
		return dto.UserResponse{}, err
	}
	return dto.UserResponse{User: toUser(u, token)}, nil
}

func validatePassword(p string) error {
	switch {
	case p == "":
		return apperr.Blank("password")
	case utf8.RuneCountInString(p) < auth.MinPasswordLength:
		return apperr.TooShort("password", auth.MinPasswordLength)
	case len(p) > auth.MaxPasswordBytes:
		return apperr.TooLong("password", auth.MaxPasswordBytes)
	}
	return nil
}

func takenOr(err error) error {
	var dup *repository.DuplicateError
	if errors.As(err, &dup) {
		return apperr.Taken(dup.Field)
	}
	return err
}

func isBlank(o dto.Optional[string]) bool {
	return o.Value == nil || *o.Value == ""
}

func nullIfEmpty(o dto.Optional[string]) *string {
	if isBlank(o) {
		return nil
	}
	return o.Value
}
