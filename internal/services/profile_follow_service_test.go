package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

func profileFixture() (*services.ProfileService, *fakeFollowRepo) {
	users := newFakeUserRepo()
	users.users[jakeID] = models.User{ID: jakeID, Username: "jake", Email: "jake@test.local"}
	users.users[minhID] = models.User{ID: minhID, Username: "minh", Email: "minh@test.local"}
	follows := &fakeFollowRepo{}
	return services.NewProfileService(users, follows), follows
}

func TestProfileService_GetFollowingByViewer(t *testing.T) {
	svc, follows := profileFixture()
	follows.pairs = map[[2]uint]bool{{minhID, jakeID}: true}
	ctx := context.Background()

	cases := []struct {
		name   string
		viewer uint
		want   bool
	}{
		{"minh (đang follow jake)", minhID, true},
		{"khách", 0, false},
		{"chính jake", jakeID, false},
	}
	for _, tc := range cases {
		res, err := svc.Get(ctx, tc.viewer, "jake")
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if res.Profile.Following != tc.want {
			t.Errorf("%s xem profile jake: following = %v, muốn %v", tc.name, res.Profile.Following, tc.want)
		}
	}
}

func TestProfileService_FollowUnfollow(t *testing.T) {
	svc, follows := profileFixture()
	ctx := context.Background()

	res, err := svc.Follow(ctx, minhID, "jake")
	if err != nil {
		t.Fatalf("Follow: %v", err)
	}
	if !res.Profile.Following || res.Profile.Username != "jake" || !follows.pairs[[2]uint{minhID, jakeID}] {
		t.Errorf("Follow: profile %+v, đã lưu %v; muốn jake following true và đã lưu (minh → jake)", res.Profile, follows.pairs)
	}
	res, err = svc.Unfollow(ctx, minhID, "jake")
	if err != nil {
		t.Fatalf("Unfollow: %v", err)
	}
	if res.Profile.Following || follows.pairs[[2]uint{minhID, jakeID}] {
		t.Errorf("Unfollow: following %v, còn lưu %v; muốn false và đã xóa", res.Profile.Following, follows.pairs)
	}
}

func TestProfileService_FollowErrors(t *testing.T) {
	svc, follows := profileFixture()
	ctx := context.Background()

	_, err := svc.Follow(ctx, minhID, "khong-co")
	wantAppErr(t, err, http.StatusNotFound, "profile", "not found")
	_, err = svc.Unfollow(ctx, minhID, "khong-co")
	wantAppErr(t, err, http.StatusNotFound, "profile", "not found")
	_, err = svc.Follow(ctx, jakeID, "jake")
	wantAppErr(t, err, http.StatusUnprocessableEntity, "profile", "cannot follow yourself")

	follows.err = errors.New("db down")
	if _, err := svc.Follow(ctx, minhID, "jake"); !errors.Is(err, follows.err) {
		t.Errorf("Follow: err = %v, muốn trả nguyên lỗi repo", err)
	}
}

func TestCommentService_ListFollowing(t *testing.T) {
	articles := &fakeArticleRepo{articles: []models.Article{{ID: 1, Slug: "a", AuthorID: jakeID}}}
	comments := &fakeCommentRepo{
		authors: map[uint]models.User{jakeID: {ID: jakeID, Username: "jake"}, annaID: {ID: annaID, Username: "anna"}},
		comments: []models.Comment{
			{ID: 1, Body: "của jake", ArticleID: 1, AuthorID: jakeID},
			{ID: 2, Body: "của anna", ArticleID: 1, AuthorID: annaID},
		},
	}
	follows := minhFollowsJake()
	res, err := services.NewCommentService(articles, comments, follows).List(context.Background(), minhID, "a")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	got := map[string]bool{}
	for _, c := range res.Comments {
		got[c.Author.Username] = c.Author.Following
	}
	if !got["jake"] || got["anna"] {
		t.Errorf("following của người bình luận = %v, muốn jake true, anna false", got)
	}
	if follows.amongCalls != 1 {
		t.Errorf("gọi FollowingAmong %d lần, muốn 1", follows.amongCalls)
	}
}
