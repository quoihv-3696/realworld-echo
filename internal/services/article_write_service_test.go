package services_test

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"slices"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
)

const jakeID = 7

func newArticle(title string, tags ...string) dto.CreateArticleRequest {
	return dto.CreateArticleRequest{Article: dto.NewArticle{Title: title, Description: "Mô tả", Body: "Nội dung", TagList: tags}}
}

func setTags(v ...string) dto.Optional[[]string] {
	if v == nil {
		v = []string{}
	}
	return dto.Optional[[]string]{Set: true, Value: &v}
}

func ownedArticleRepo() *fakeArticleRepo {
	return &fakeArticleRepo{
		articles: []models.Article{{ID: 1, Slug: "s", Title: "T", Description: "D", Body: "B", AuthorID: jakeID, Tags: []models.Tag{{Name: "go"}}}},
		authors:  map[uint]models.User{jakeID: {ID: jakeID, Username: "jake"}},
		nextID:   1,
	}
}

func TestArticleService_CreateBlankFields(t *testing.T) {
	svc := services.NewArticleService(&fakeArticleRepo{})
	cases := map[string]dto.NewArticle{
		"title":       {Title: "", Description: "d", Body: "b"},
		"description": {Title: "t", Description: "", Body: "b"},
		"body":        {Title: "t", Description: "d", Body: ""},
	}
	for field, in := range cases {
		_, err := svc.Create(context.Background(), jakeID, dto.CreateArticleRequest{Article: in})
		wantAppErr(t, err, http.StatusUnprocessableEntity, field, "can't be blank")
	}
}

func TestArticleService_CreateSlugTagsAuthor(t *testing.T) {
	repo := &fakeArticleRepo{authors: map[uint]models.User{jakeID: {ID: jakeID, Username: "jake"}}}
	res, err := services.NewArticleService(repo).Create(context.Background(), jakeID, newArticle("Học Go trong 30 ngày", " go ", "echo", "go", ""))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	a := res.Article
	if a.Slug != "hoc-go-trong-30-ngay" {
		t.Errorf("slug = %q, muốn \"hoc-go-trong-30-ngay\" (slug.Make(title))", a.Slug)
	}
	if !slices.Equal(repo.gotCreateTags, []string{"go", "echo"}) {
		t.Errorf("tag truyền cho repo = %q, muốn [go echo]: bỏ khoảng trắng, bỏ rỗng, bỏ trùng (normalizeTags)", repo.gotCreateTags)
	}
	if !slices.Equal(a.TagList, []string{"echo", "go"}) {
		t.Errorf("tagList = %v, muốn [echo go]", a.TagList)
	}
	if a.Author.Username != "jake" {
		t.Errorf("author.username = %q, muốn \"jake\": sau Create, đọc lại bài bằng FindBySlug để có author", a.Author.Username)
	}
	if len(repo.articles) != 1 || repo.articles[0].AuthorID != jakeID {
		t.Errorf("bài lưu xuống repo = %+v, muốn AuthorID %d (lấy từ token, không lấy từ body)", repo.articles, jakeID)
	}
}

func TestArticleService_CreateWithoutTags(t *testing.T) {
	repo := &fakeArticleRepo{}
	res, err := services.NewArticleService(repo).Create(context.Background(), jakeID, newArticle("Không tag"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if repo.gotCreateTags == nil || res.Article.TagList == nil || len(res.Article.TagList) != 0 {
		t.Errorf("tag gửi repo = %#v, tagList = %#v; muốn slice rỗng (không nil) ở cả hai", repo.gotCreateTags, res.Article.TagList)
	}
}

func TestArticleService_CreateDuplicateTitleGetsSuffix(t *testing.T) {
	repo := &fakeArticleRepo{articles: []models.Article{{ID: 1, Slug: "dup-title", AuthorID: jakeID}}, nextID: 1}
	res, err := services.NewArticleService(repo).Create(context.Background(), jakeID, newArticle("Dup Title"))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if !regexp.MustCompile(`^dup-title-[a-z0-9]{6}$`).MatchString(res.Article.Slug) {
		t.Errorf("slug = %q, muốn dup-title-xxxxxx: gặp DuplicateError{Field: \"slug\"} thì thử lại với slug.WithSuffix(base)", res.Article.Slug)
	}
}

func TestArticleService_CreateStopsRetrying(t *testing.T) {
	dup := &repository.DuplicateError{Field: "slug"}
	repo := &fakeArticleRepo{createErrs: []error{dup, dup, dup, dup}}
	_, err := services.NewArticleService(repo).Create(context.Background(), jakeID, newArticle("Kẹt"))
	if err == nil {
		t.Fatal("slug trùng mãi mà Create vẫn thành công")
	}
	if len(repo.createErrs) != 1 {
		t.Errorf("đã gọi repo.Create %d lần, muốn đúng 3 (maxSlugAttempts)", 4-len(repo.createErrs))
	}
}

func TestArticleService_CreatePassesOtherErrors(t *testing.T) {
	boom := errors.New("db down")
	repo := &fakeArticleRepo{createErrs: []error{boom}}
	_, err := services.NewArticleService(repo).Create(context.Background(), jakeID, newArticle("Lỗi"))
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, muốn trả nguyên lỗi khác DuplicateError (không thử lại)", err)
	}
	if len(repo.articles) != 0 {
		t.Error("lỗi DB mà vẫn có bài")
	}
}

func TestArticleService_UpdateFields(t *testing.T) {
	repo := ownedArticleRepo()
	svc := services.NewArticleService(repo)
	ctx := context.Background()
	update := func(u dto.UpdateArticle) dto.Article {
		t.Helper()
		res, err := svc.Update(ctx, jakeID, "s", dto.UpdateArticleRequest{Article: u})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		return res.Article
	}

	a := update(dto.UpdateArticle{Body: set("Nội dung mới")})
	if a.Body != "Nội dung mới" || a.Title != "T" || a.Description != "D" {
		t.Errorf("chỉ gửi body mà ra %+v: field không gửi phải giữ nguyên", a.ArticleListItem)
	}
	if repo.gotUpdateTags != nil {
		t.Errorf("không gửi tagList mà repo nhận %v, muốn nil (giữ tag)", *repo.gotUpdateTags)
	}
	if a = update(dto.UpdateArticle{Title: set("Tiêu đề mới")}); a.Slug != "s" || a.Title != "Tiêu đề mới" {
		t.Errorf("đổi title: slug = %q, title = %q; muốn slug giữ \"s\" (link cũ không gãy)", a.Slug, a.Title)
	}
	if a = update(dto.UpdateArticle{TagList: setTags("b", "a", "b", " ")}); repo.gotUpdateTags == nil || !slices.Equal(*repo.gotUpdateTags, []string{"b", "a"}) {
		t.Errorf("tagList [b a b \" \"] → repo nhận %v, muốn [b a]", repo.gotUpdateTags)
	}
	if a = update(dto.UpdateArticle{TagList: setTags()}); repo.gotUpdateTags == nil || len(*repo.gotUpdateTags) != 0 || len(a.TagList) != 0 {
		t.Errorf("tagList [] → repo nhận %v, tagList %v; muốn slice rỗng (xóa hết tag)", repo.gotUpdateTags, a.TagList)
	}
}

func TestArticleService_UpdateRejectsBlankAndNull(t *testing.T) {
	svc := services.NewArticleService(ownedArticleRepo())
	cases := []struct {
		in         dto.UpdateArticle
		field, msg string
	}{
		{dto.UpdateArticle{Title: set("")}, "title", "can't be blank"},
		{dto.UpdateArticle{Description: setNull}, "description", "can't be blank"},
		{dto.UpdateArticle{Body: setNull}, "body", "can't be blank"},
		{dto.UpdateArticle{TagList: dto.Optional[[]string]{Set: true}}, "tagList", "can't be null"},
	}
	for _, tc := range cases {
		_, err := svc.Update(context.Background(), jakeID, "s", dto.UpdateArticleRequest{Article: tc.in})
		wantAppErr(t, err, http.StatusUnprocessableEntity, tc.field, tc.msg)
	}
}

func TestArticleService_OwnershipAndNotFound(t *testing.T) {
	repo := ownedArticleRepo()
	svc := services.NewArticleService(repo)
	ctx := context.Background()
	body := dto.UpdateArticleRequest{Article: dto.UpdateArticle{Body: set("hijacked")}}

	_, err := svc.Update(ctx, 8, "s", body)
	wantAppErr(t, err, http.StatusForbidden, "article", "forbidden")
	_, err = svc.Update(ctx, jakeID, "khong-co", body)
	wantAppErr(t, err, http.StatusNotFound, "article", "not found")
	wantAppErr(t, svc.Delete(ctx, 8, "s"), http.StatusForbidden, "article", "forbidden")
	wantAppErr(t, svc.Delete(ctx, jakeID, "khong-co"), http.StatusNotFound, "article", "not found")
	if len(repo.deleted) != 0 {
		t.Fatalf("người khác xóa được bài: repo.Delete(%v)", repo.deleted)
	}
	if err := svc.Delete(ctx, jakeID, "s"); err != nil {
		t.Fatalf("tác giả xóa bài: %v", err)
	}
	if !slices.Equal(repo.deleted, []uint{1}) {
		t.Errorf("repo.Delete nhận %v, muốn [1] (ID của bài)", repo.deleted)
	}
}
