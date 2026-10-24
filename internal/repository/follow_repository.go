package repository

import (
	"context"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type FollowRepository interface {
	Follow(ctx context.Context, followerID, followingID uint) error
	Unfollow(ctx context.Context, followerID, followingID uint) error
	IsFollowing(ctx context.Context, followerID, followingID uint) (bool, error)
	FollowingAmong(ctx context.Context, followerID uint, userIDs []uint) (map[uint]bool, error)
}

type followRepository struct {
	db *gorm.DB
}

func NewFollowRepository(db *gorm.DB) FollowRepository {
	return &followRepository{db: db}
}

func (r *followRepository) Follow(ctx context.Context, followerID, followingID uint) error {
	f := models.Follow{FollowerID: followerID, FollowingID: followingID}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Omit("Follower", "Following").
		Create(&f).Error
}

func (r *followRepository) Unfollow(ctx context.Context, followerID, followingID uint) error {
	return r.db.WithContext(ctx).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Delete(&models.Follow{}).Error
}

func (r *followRepository) IsFollowing(ctx context.Context, followerID, followingID uint) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&models.Follow{}).
		Where("follower_id = ? AND following_id = ?", followerID, followingID).
		Count(&n).Error
	return n > 0, err
}

func (r *followRepository) FollowingAmong(ctx context.Context, followerID uint, userIDs []uint) (map[uint]bool, error) {
	out := make(map[uint]bool, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	var ids []uint
	err := r.db.WithContext(ctx).
		Model(&models.Follow{}).
		Where("follower_id = ? AND following_id IN ?", followerID, userIDs).
		Pluck("following_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}
