package services_test

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

type fakeCommentRepo struct {
	comments []models.Comment
	authors  map[uint]models.User
	nextID   uint
	err      error
	deleted  []uint
}

func (f *fakeCommentRepo) withAuthor(c models.Comment) models.Comment {
	if u, ok := f.authors[c.AuthorID]; ok {
		c.Author = u
	}
	return c
}

func (f *fakeCommentRepo) Create(_ context.Context, c *models.Comment) error {
	if f.err != nil {
		return f.err
	}
	f.nextID++
	c.ID = f.nextID
	c.CreatedAt = time.Date(2026, 10, 1, 9, 0, 0, 0, hcm)
	c.UpdatedAt = c.CreatedAt
	*c = f.withAuthor(*c)
	f.comments = append(f.comments, *c)
	return nil
}

func (f *fakeCommentRepo) ListByArticle(_ context.Context, articleID uint) ([]models.Comment, error) {
	if f.err != nil {
		return nil, f.err
	}
	var out []models.Comment
	for _, c := range f.comments {
		if c.ArticleID == articleID {
			out = append(out, f.withAuthor(c))
		}
	}
	return out, nil
}

func (f *fakeCommentRepo) FindByID(_ context.Context, id uint) (*models.Comment, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, c := range f.comments {
		if c.ID == id {
			return &c, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeCommentRepo) Delete(_ context.Context, id uint) error {
	if f.err != nil {
		return f.err
	}
	f.deleted = append(f.deleted, id)
	return nil
}

func commentFixture() (*services.CommentService, *fakeCommentRepo) {
	articles := &fakeArticleRepo{articles: []models.Article{{ID: 1, Slug: "a", AuthorID: jakeID}, {ID: 2, Slug: "b", AuthorID: jakeID}}}
	comments := &fakeCommentRepo{
		authors: map[uint]models.User{jakeID: {ID: jakeID, Username: "jake"}, 8: {ID: 8, Username: "anna"}},
		comments: []models.Comment{
			{ID: 10, Body: "của jake trên a", ArticleID: 1, AuthorID: jakeID},
			{ID: 11, Body: "của anna trên a", ArticleID: 1, AuthorID: 8},
			{ID: 12, Body: "trên b", ArticleID: 2, AuthorID: jakeID},
		},
		nextID: 12,
	}
	return services.NewCommentService(articles, comments), comments
}

func TestCommentService_List(t *testing.T) {
	svc, _ := commentFixture()
	res, err := svc.List(context.Background(), "a")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var bodies []string
	for _, c := range res.Comments {
		bodies = append(bodies, c.Body)
	}
	if !slices.Equal(bodies, []string{"của jake trên a", "của anna trên a"}) || res.Comments[1].Author.Username != "anna" {
		t.Errorf("comments = %+v, muốn 2 comment của bài a kèm author", res.Comments)
	}
	_, err = svc.List(context.Background(), "khong-co")
	wantAppErr(t, err, http.StatusNotFound, "article", "not found")
}

func TestCommentService_ListEmptyIsNotNil(t *testing.T) {
	articles := &fakeArticleRepo{articles: []models.Article{{ID: 1, Slug: "a"}}}
	res, err := services.NewCommentService(articles, &fakeCommentRepo{}).List(context.Background(), "a")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if res.Comments == nil {
		t.Error("Comments = nil, muốn []dto.Comment{} (nil thành null trong JSON)")
	}
}

func TestCommentService_Create(t *testing.T) {
	svc, repo := commentFixture()
	ctx := context.Background()

	_, err := svc.Create(ctx, jakeID, "a", dto.CreateCommentRequest{Comment: dto.NewComment{Body: ""}})
	wantAppErr(t, err, http.StatusUnprocessableEntity, "body", "can't be blank")
	_, err = svc.Create(ctx, jakeID, "khong-co", dto.CreateCommentRequest{Comment: dto.NewComment{Body: "orphan"}})
	wantAppErr(t, err, http.StatusNotFound, "article", "not found")

	res, err := svc.Create(ctx, 8, "b", dto.CreateCommentRequest{Comment: dto.NewComment{Body: "hay"}})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	c := res.Comment
	if c.ID == 0 || c.Body != "hay" || c.Author.Username != "anna" {
		t.Errorf("comment = %+v, muốn id khác 0, body \"hay\", author anna", c)
	}
	if c.CreatedAt.Location() != time.UTC {
		t.Errorf("createdAt = %v, muốn giờ UTC", c.CreatedAt)
	}
	stored := repo.comments[len(repo.comments)-1]
	if stored.ArticleID != 2 || stored.AuthorID != 8 {
		t.Errorf("comment lưu xuống repo: ArticleID %d, AuthorID %d; muốn 2 và 8", stored.ArticleID, stored.AuthorID)
	}
}

func TestCommentService_Delete(t *testing.T) {
	svc, repo := commentFixture()
	ctx := context.Background()

	wantAppErr(t, svc.Delete(ctx, jakeID, "khong-co", 10), http.StatusNotFound, "article", "not found")
	wantAppErr(t, svc.Delete(ctx, jakeID, "a", 99999), http.StatusNotFound, "comment", "not found")
	wantAppErr(t, svc.Delete(ctx, jakeID, "a", 12), http.StatusNotFound, "comment", "not found")
	wantAppErr(t, svc.Delete(ctx, jakeID, "a", 11), http.StatusForbidden, "comment", "forbidden")
	if len(repo.deleted) != 0 {
		t.Fatalf("xóa được comment không phải của mình hoặc sai bài: %v", repo.deleted)
	}
	if err := svc.Delete(ctx, jakeID, "a", 10); err != nil {
		t.Fatalf("xóa comment của mình: %v", err)
	}
	if !slices.Equal(repo.deleted, []uint{10}) {
		t.Errorf("repo.Delete nhận %v, muốn [10]", repo.deleted)
	}
}

func TestCommentService_PassesOtherErrors(t *testing.T) {
	boom := errors.New("db down")
	articles := &fakeArticleRepo{articles: []models.Article{{ID: 1, Slug: "a"}}}
	svc := services.NewCommentService(articles, &fakeCommentRepo{err: boom})
	if _, err := svc.List(context.Background(), "a"); !errors.Is(err, boom) {
		t.Errorf("List: err = %v, muốn trả nguyên lỗi repo", err)
	}
	if err := svc.Delete(context.Background(), jakeID, "a", 1); !errors.Is(err, boom) {
		t.Errorf("Delete: err = %v, muốn trả nguyên lỗi repo (không biến thành 404)", err)
	}
}

func TestProfileService_Get(t *testing.T) {
	repo := newFakeUserRepo()
	bio := "Gopher"
	repo.users[1] = models.User{ID: 1, Username: "jake", Email: "jake@test.local", Bio: &bio}
	svc := services.NewProfileService(repo)

	res, err := svc.Get(context.Background(), "jake")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	p := res.Profile
	if p.Username != "jake" || p.Bio == nil || *p.Bio != "Gopher" || p.Image != nil || p.Following {
		t.Errorf("profile = %+v, muốn jake, bio Gopher, image nil, following false", p)
	}
	_, err = svc.Get(context.Background(), "khong-co")
	wantAppErr(t, err, http.StatusNotFound, "profile", "not found")

	repo.err = errors.New("db down")
	if _, err := svc.Get(context.Background(), "jake"); !errors.Is(err, repo.err) {
		t.Errorf("err = %v, muốn trả nguyên lỗi repo", err)
	}
}
