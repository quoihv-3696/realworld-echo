package services_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type fakeTagRepo struct {
	names []string
	err   error
}

func (f fakeTagRepo) ListNames(context.Context) ([]string, error) { return f.names, f.err }

type fakeArticleRepo struct {
	articles      []models.Article
	total         int64
	err           error
	gotFilter     repository.ArticleFilter
	authors       map[uint]models.User
	nextID        uint
	createErrs    []error
	gotCreateTags []string
	gotUpdateTags *[]string
	deleted       []uint
}

func (f *fakeArticleRepo) List(_ context.Context, filter repository.ArticleFilter) ([]models.Article, int64, error) {
	f.gotFilter = filter
	return f.articles, f.total, f.err
}

func (f *fakeArticleRepo) FindBySlug(_ context.Context, slug string) (*models.Article, error) {
	for _, a := range f.articles {
		if a.Slug == slug {
			if u, ok := f.authors[a.AuthorID]; ok {
				a.Author = u
			}
			return &a, nil
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return nil, repository.ErrNotFound
}

func (f *fakeArticleRepo) Create(_ context.Context, a *models.Article, tagNames []string) error {
	f.gotCreateTags = tagNames
	if len(f.createErrs) > 0 {
		err := f.createErrs[0]
		f.createErrs = f.createErrs[1:]
		if err != nil {
			return err
		}
	}
	if f.err != nil {
		return f.err
	}
	for _, other := range f.articles {
		if other.Slug == a.Slug {
			return &repository.DuplicateError{Field: "slug"}
		}
	}
	f.nextID++
	a.ID = f.nextID
	a.Tags = nil
	for _, n := range tagNames {
		a.Tags = append(a.Tags, models.Tag{Name: n})
	}
	f.articles = append(f.articles, *a)
	return nil
}

func (f *fakeArticleRepo) Update(_ context.Context, a *models.Article, tagNames *[]string) error {
	f.gotUpdateTags = tagNames
	if f.err != nil {
		return f.err
	}
	for i := range f.articles {
		if f.articles[i].ID == a.ID {
			if tagNames != nil {
				a.Tags = nil
				for _, n := range *tagNames {
					a.Tags = append(a.Tags, models.Tag{Name: n})
				}
			}
			f.articles[i] = *a
			return nil
		}
	}
	return repository.ErrNotFound
}

func (f *fakeArticleRepo) Delete(_ context.Context, id uint) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, id)
	for i := range f.articles {
		if f.articles[i].ID == id {
			f.articles = append(f.articles[:i], f.articles[i+1:]...)
			return nil
		}
	}
	return nil
}

type fakeFollowRepo struct {
	pairs      map[[2]uint]bool
	err        error
	amongCalls int
}

func (f *fakeFollowRepo) ensure() {
	if f.pairs == nil {
		f.pairs = map[[2]uint]bool{}
	}
}

func (f *fakeFollowRepo) Follow(_ context.Context, followerID, followingID uint) error {
	if f.err != nil {
		return f.err
	}
	f.ensure()
	f.pairs[[2]uint{followerID, followingID}] = true
	return nil
}

func (f *fakeFollowRepo) Unfollow(_ context.Context, followerID, followingID uint) error {
	if f.err != nil {
		return f.err
	}
	f.ensure()
	delete(f.pairs, [2]uint{followerID, followingID})
	return nil
}

func (f *fakeFollowRepo) IsFollowing(_ context.Context, followerID, followingID uint) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	return f.pairs[[2]uint{followerID, followingID}], nil
}

func (f *fakeFollowRepo) FollowingAmong(_ context.Context, followerID uint, userIDs []uint) (map[uint]bool, error) {
	f.amongCalls++
	if f.err != nil {
		return nil, f.err
	}
	out := map[uint]bool{}
	for _, id := range userIDs {
		if f.pairs[[2]uint{followerID, id}] {
			out[id] = true
		}
	}
	return out, nil
}

var hcm = time.FixedZone("ICT", 7*3600)

func sampleArticle() models.Article {
	bio := "bio"
	return models.Article{
		Slug: "s", Title: "T", Description: "D", Body: "B",
		Author:    models.User{Username: "jake", Bio: &bio},
		Tags:      []models.Tag{{Name: "go"}, {Name: "echo"}},
		CreatedAt: time.Date(2026, 1, 2, 10, 0, 0, 0, hcm),
		UpdatedAt: time.Date(2026, 1, 2, 11, 0, 0, 0, hcm),
	}
}

func TestTagService_ListNilBecomesEmpty(t *testing.T) {
	res, err := services.NewTagService(fakeTagRepo{}).List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.Tags == nil || len(res.Tags) != 0 {
		t.Errorf("Tags = %#v, muốn []string{} (nil sẽ thành null trong JSON)", res.Tags)
	}
}

func TestTagService_ListPassesError(t *testing.T) {
	boom := errors.New("db down")
	if _, err := services.NewTagService(fakeTagRepo{err: boom}).List(context.Background()); !errors.Is(err, boom) {
		t.Errorf("err = %v, muốn trả nguyên lỗi của repo", err)
	}
}

func TestArticleService_ListMapsAndUsesDefaultLimit(t *testing.T) {
	repo := &fakeArticleRepo{articles: []models.Article{sampleArticle(), {Slug: "no-tags"}}, total: 42}
	res, err := services.NewArticleService(repo, &fakeFollowRepo{}, &fakeFavoriteRepo{}).List(context.Background(), 0, services.ArticleQuery{Limit: services.DefaultLimit})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if repo.gotFilter.Limit != services.DefaultLimit || repo.gotFilter.Offset != 0 {
		t.Errorf("filter = %+v, muốn Limit %d, Offset 0", repo.gotFilter, services.DefaultLimit)
	}
	if res.ArticlesCount != 42 || len(res.Articles) != 2 {
		t.Fatalf("ArticlesCount = %d, len = %d; muốn 42 và 2", res.ArticlesCount, len(res.Articles))
	}
	a := res.Articles[0]
	if a.Slug != "s" || a.Title != "T" || a.Description != "D" {
		t.Errorf("sai slug/title/description: %+v", a)
	}
	if !slices.Equal(a.TagList, []string{"echo", "go"}) {
		t.Errorf("TagList = %v, muốn [echo go] (sắp theo tên)", a.TagList)
	}
	if res.Articles[1].TagList == nil {
		t.Error("TagList của bài không tag là nil, muốn []string{}")
	}
	if a.CreatedAt.Location() != time.UTC || !a.CreatedAt.Equal(sampleArticle().CreatedAt) {
		t.Errorf("CreatedAt = %v, muốn cùng thời điểm nhưng ở UTC", a.CreatedAt)
	}
	if a.UpdatedAt.Location() != time.UTC || !a.UpdatedAt.Equal(sampleArticle().UpdatedAt) {
		t.Errorf("UpdatedAt = %v, muốn cùng thời điểm nhưng ở UTC", a.UpdatedAt)
	}
	if a.Author.Username != "jake" || a.Author.Bio == nil || *a.Author.Bio != "bio" || a.Author.Image != nil || a.Author.Following {
		t.Errorf("Author = %+v, muốn jake, bio \"bio\", image nil, following false", a.Author)
	}
	if res.Articles[1].Author.Bio != nil {
		t.Error("Bio rỗng phải là nil (JSON null)")
	}
}

func TestArticleService_ListEmpty(t *testing.T) {
	res, err := services.NewArticleService(&fakeArticleRepo{}, &fakeFollowRepo{}, &fakeFavoriteRepo{}).List(context.Background(), 0, services.ArticleQuery{Limit: services.DefaultLimit})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.Articles == nil {
		t.Error("Articles là nil, muốn []dto.ArticleListItem{} (nil sẽ thành null trong JSON)")
	}
}

func TestArticleService_Get(t *testing.T) {
	res, err := services.NewArticleService(&fakeArticleRepo{articles: []models.Article{sampleArticle()}}, &fakeFollowRepo{}, &fakeFavoriteRepo{}).Get(context.Background(), 0, "s")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if res.Article.Body != "B" || res.Article.Slug != "s" || !slices.Equal(res.Article.TagList, []string{"echo", "go"}) {
		t.Errorf("Article = %+v, muốn slug s, body B, tagList [echo go]", res.Article)
	}
}

func TestArticleService_GetNotFound(t *testing.T) {
	_, err := services.NewArticleService(&fakeArticleRepo{}, &fakeFollowRepo{}, &fakeFavoriteRepo{}).Get(context.Background(), 0, "khong-co")
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Status != http.StatusNotFound || appErr.Fields["article"][0] != "not found" {
		t.Fatalf("err = %v, muốn apperr.NotFound(\"article\")", err)
	}
}

func TestArticleService_GetPassesOtherErrors(t *testing.T) {
	boom := errors.New("db down")
	_, err := services.NewArticleService(&fakeArticleRepo{err: boom}, &fakeFollowRepo{}, &fakeFavoriteRepo{}).Get(context.Background(), 0, "x")
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, muốn trả nguyên lỗi khác ErrNotFound", err)
	}
}

type fakeFavoriteRepo struct {
	pairs      map[[2]uint]bool
	err        error
	countCalls int
	amongCalls int
}

func (f *fakeFavoriteRepo) Favorite(_ context.Context, userID, articleID uint) error {
	if f.err != nil {
		return f.err
	}
	if f.pairs == nil {
		f.pairs = map[[2]uint]bool{}
	}
	f.pairs[[2]uint{userID, articleID}] = true
	return nil
}

func (f *fakeFavoriteRepo) Unfavorite(_ context.Context, userID, articleID uint) error {
	if f.err != nil {
		return f.err
	}
	delete(f.pairs, [2]uint{userID, articleID})
	return nil
}

func (f *fakeFavoriteRepo) CountByArticle(_ context.Context, articleIDs []uint) (map[uint]int64, error) {
	f.countCalls++
	if f.err != nil {
		return nil, f.err
	}
	out := map[uint]int64{}
	for pair := range f.pairs {
		for _, id := range articleIDs {
			if pair[1] == id {
				out[id]++
			}
		}
	}
	return out, nil
}

func (f *fakeFavoriteRepo) FavoritedAmong(_ context.Context, userID uint, articleIDs []uint) (map[uint]bool, error) {
	f.amongCalls++
	if f.err != nil {
		return nil, f.err
	}
	out := map[uint]bool{}
	for _, id := range articleIDs {
		if f.pairs[[2]uint{userID, id}] {
			out[id] = true
		}
	}
	return out, nil
}
