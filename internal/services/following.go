package services

import (
	"context"

	"github.com/quoihv-3696/realworld-echo/internal/repository"
)

func followingAmong(ctx context.Context, follows repository.FollowRepository, viewerID uint, userIDs []uint) (map[uint]bool, error) {
	if viewerID == 0 || len(userIDs) == 0 {
		return map[uint]bool{}, nil
	}
	return follows.FollowingAmong(ctx, viewerID, userIDs)
}
