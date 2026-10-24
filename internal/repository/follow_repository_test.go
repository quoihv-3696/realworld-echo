package repository_test

import (
	"context"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestFollowRepository_FollowUnfollow(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	minh := testutil.CreateUser(t, db, "minh", nil)
	repo := repository.NewFollowRepository(db)
	ctx := context.Background()

	for i := 0; i < 2; i++ {
		if err := repo.Follow(ctx, minh.ID, jake.ID); err != nil {
			t.Fatalf("Follow lần %d: %v (follow lại người đã follow không được lỗi: ON CONFLICT DO NOTHING)", i+1, err)
		}
	}
	var n int64
	db.Model(&models.Follow{}).Count(&n)
	if n != 1 {
		t.Errorf("follow 2 lần ra %d dòng, muốn 1", n)
	}
	if ok, err := repo.IsFollowing(ctx, minh.ID, jake.ID); err != nil || !ok {
		t.Errorf("IsFollowing(minh, jake) = %v, %v; muốn true", ok, err)
	}
	if ok, err := repo.IsFollowing(ctx, jake.ID, minh.ID); err != nil || ok {
		t.Errorf("IsFollowing(jake, minh) = %v, %v; muốn false (follow có chiều)", ok, err)
	}
	if err := repo.Unfollow(ctx, minh.ID, jake.ID); err != nil {
		t.Fatalf("Unfollow: %v", err)
	}
	if ok, _ := repo.IsFollowing(ctx, minh.ID, jake.ID); ok {
		t.Error("sau Unfollow vẫn đang follow")
	}
	if err := repo.Unfollow(ctx, minh.ID, jake.ID); err != nil {
		t.Errorf("Unfollow khi chưa follow: %v, muốn không lỗi", err)
	}
}

func TestFollowRepository_FollowingAmong(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	anna := testutil.CreateUser(t, db, "anna", nil)
	minh := testutil.CreateUser(t, db, "minh", nil)
	testutil.CreateFollow(t, db, minh, jake)
	testutil.CreateFollow(t, db, anna, minh)
	repo := repository.NewFollowRepository(db)

	got, err := repo.FollowingAmong(context.Background(), minh.ID, []uint{jake.ID, anna.ID, 999})
	if err != nil {
		t.Fatalf("FollowingAmong: %v", err)
	}
	if len(got) != 1 || !got[jake.ID] {
		t.Errorf("FollowingAmong(minh, [jake anna 999]) = %v, muốn chỉ {jake: true}", got)
	}
	empty, err := repo.FollowingAmong(context.Background(), minh.ID, nil)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Errorf("FollowingAmong với danh sách rỗng = %v, %v; muốn map rỗng (khác nil), không lỗi", empty, err)
	}
}

func TestFollowRepository_UsesRequestContext(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	minh := testutil.CreateUser(t, db, "minh", nil)
	repo := repository.NewFollowRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Errorf("%s với context đã hủy phải trả lỗi (quên WithContext(ctx)?)", name)
		}
	}
	check("Follow", repo.Follow(ctx, minh.ID, jake.ID))
	check("Unfollow", repo.Unfollow(ctx, minh.ID, jake.ID))
	_, err := repo.IsFollowing(ctx, minh.ID, jake.ID)
	check("IsFollowing", err)
	_, err = repo.FollowingAmong(ctx, minh.ID, []uint{jake.ID})
	check("FollowingAmong", err)
}
