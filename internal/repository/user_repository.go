package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/quoihv-3696/realworld-echo/internal/models"
)

type UserRepository interface {
	Create(ctx context.Context, u *models.User) error
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, u *models.User) error
	FindByUsername(ctx context.Context, username string) (*models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

var userUniqueIndexes = map[string]string{
	"idx_users_username": "username",
	"idx_users_email":    "email",
}

func (r *userRepository) Create(ctx context.Context, u *models.User) error {
	return translateUserError(r.db.WithContext(ctx).Create(u).Error)
}

func (r *userRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	return r.findOne(ctx, "id = ?", id)
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	return r.findOne(ctx, "email = ?", email)
}

func (r *userRepository) findOne(ctx context.Context, query string, arg any) (*models.User, error) {
	var u models.User
	err := r.db.WithContext(ctx).Where(query, arg).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) Update(ctx context.Context, u *models.User) error {
	return translateUserError(r.db.WithContext(ctx).Save(u).Error)
}

func translateUserError(err error) error {
	return translateDuplicate(err, userUniqueIndexes)
}

func translateDuplicate(err error, indexes map[string]string) error {
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if field, ok := indexes[pgErr.ConstraintName]; ok {
			return &DuplicateError{Field: field}
		}
	}
	return err
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	return r.findOne(ctx, "username = ?", username)
}
