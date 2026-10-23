package services_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/dto"
	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/services"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

type fakeUserRepo struct {
	users  map[uint]models.User
	nextID uint
	err    error
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{users: map[uint]models.User{}} }

func (f *fakeUserRepo) duplicate(u *models.User) error {
	for _, other := range f.users {
		if other.ID == u.ID {
			continue
		}
		if other.Username == u.Username {
			return &repository.DuplicateError{Field: "username"}
		}
		if other.Email == u.Email {
			return &repository.DuplicateError{Field: "email"}
		}
	}
	return nil
}

func (f *fakeUserRepo) Create(_ context.Context, u *models.User) error {
	if f.err != nil {
		return f.err
	}
	if err := f.duplicate(u); err != nil {
		return err
	}
	f.nextID++
	u.ID = f.nextID
	f.users[u.ID] = *u
	return nil
}

func (f *fakeUserRepo) FindByID(_ context.Context, id uint) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	u, ok := f.users[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &u, nil
}

func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, u := range f.users {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) FindByUsername(_ context.Context, username string) (*models.User, error) {
	if f.err != nil {
		return nil, f.err
	}
	for _, u := range f.users {
		if u.Username == username {
			return &u, nil
		}
	}
	return nil, repository.ErrNotFound
}

func (f *fakeUserRepo) Update(_ context.Context, u *models.User) error {
	if f.err != nil {
		return f.err
	}
	if _, ok := f.users[u.ID]; !ok {
		return repository.ErrNotFound
	}
	if err := f.duplicate(u); err != nil {
		return err
	}
	f.users[u.ID] = *u
	return nil
}

func newUserService(t *testing.T) (*services.UserService, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return services.NewUserService(repo, testutil.JWT(t)), repo
}

func register(t *testing.T, svc *services.UserService, username, password string) dto.User {
	t.Helper()
	res, err := svc.Register(context.Background(), dto.RegisterRequest{User: dto.NewUser{
		Username: username, Email: username + "@test.local", Password: password,
	}})
	if err != nil {
		t.Fatalf("Register %s: %v", username, err)
	}
	return res.User
}

func wantAppErr(t *testing.T, err error, status int, field, msg string) {
	t.Helper()
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, muốn *apperr.Error %d {%q: [%q]}", err, status, field, msg)
	}
	if appErr.Status != status || len(appErr.Fields[field]) == 0 || appErr.Fields[field][0] != msg {
		t.Fatalf("lỗi = %d %v, muốn %d {%q: [%q]}", appErr.Status, appErr.Fields, status, field, msg)
	}
}

func subjectOf(t *testing.T, token string) string {
	t.Helper()
	var claims jwt.RegisteredClaims
	if _, err := jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return []byte(testutil.JWTSecret), nil }); err != nil {
		t.Fatalf("token trong response không hợp lệ: %v", err)
	}
	return claims.Subject
}

func set(s string) dto.Optional[string] { return dto.Optional[string]{Set: true, Value: &s} }

var setNull = dto.Optional[string]{Set: true}

func TestUserService_RegisterBlankFields(t *testing.T) {
	svc, _ := newUserService(t)
	cases := []struct {
		in    dto.NewUser
		field string
	}{
		{dto.NewUser{Username: "", Email: "a@test.local", Password: "password123"}, "username"},
		{dto.NewUser{Username: "a", Email: "", Password: "password123"}, "email"},
		{dto.NewUser{Username: "a", Email: "a@test.local", Password: ""}, "password"},
	}
	for _, tc := range cases {
		_, err := svc.Register(context.Background(), dto.RegisterRequest{User: tc.in})
		wantAppErr(t, err, http.StatusUnprocessableEntity, tc.field, "can't be blank")
	}
}

func TestUserService_RegisterPasswordLength(t *testing.T) {
	svc, _ := newUserService(t)
	_, err := svc.Register(context.Background(), dto.RegisterRequest{User: dto.NewUser{Username: "a", Email: "a@test.local", Password: "short7c"}})
	wantAppErr(t, err, http.StatusUnprocessableEntity, "password", "is too short (minimum is 8 characters)")

	_, err = svc.Register(context.Background(), dto.RegisterRequest{User: dto.NewUser{Username: "b", Email: "b@test.local", Password: strings.Repeat("a", 73)}})
	wantAppErr(t, err, http.StatusUnprocessableEntity, "password", "is too long (maximum is 72 characters)")

	register(t, svc, "c", "bonjour1")
	register(t, svc, "d", strings.Repeat("a", 72))
}

func TestUserService_RegisterSuccess(t *testing.T) {
	svc, repo := newUserService(t)
	u := register(t, svc, "jake", "password123")

	if u.Username != "jake" || u.Email != "jake@test.local" || u.Bio != nil || u.Image != nil {
		t.Errorf("user = %+v, muốn jake, jake@test.local, bio nil, image nil", u)
	}
	if sub := subjectOf(t, u.Token); sub != "1" {
		t.Errorf("sub trong token = %q, muốn \"1\" (ID của user vừa tạo)", sub)
	}
	stored := repo.users[1]
	if stored.Password == "password123" || bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("password123")) != nil {
		t.Errorf("mật khẩu lưu vào repo = %q, muốn hash bcrypt của mật khẩu (auth.HashPassword)", stored.Password)
	}
}

func TestUserService_RegisterDuplicate(t *testing.T) {
	svc, _ := newUserService(t)
	register(t, svc, "jake", "password123")

	_, err := svc.Register(context.Background(), dto.RegisterRequest{User: dto.NewUser{Username: "jake", Email: "khac@test.local", Password: "password123"}})
	wantAppErr(t, err, http.StatusConflict, "username", "has already been taken")
	_, err = svc.Register(context.Background(), dto.RegisterRequest{User: dto.NewUser{Username: "khac", Email: "jake@test.local", Password: "password123"}})
	wantAppErr(t, err, http.StatusConflict, "email", "has already been taken")
}

func TestUserService_RegisterPassesOtherErrors(t *testing.T) {
	svc, repo := newUserService(t)
	boom := errors.New("db down")
	repo.err = boom
	_, err := svc.Register(context.Background(), dto.RegisterRequest{User: dto.NewUser{Username: "a", Email: "a@test.local", Password: "password123"}})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v, muốn trả nguyên lỗi khác DuplicateError", err)
	}
}

func TestUserService_Login(t *testing.T) {
	svc, _ := newUserService(t)
	register(t, svc, "jake", "password123")
	ctx := context.Background()

	res, err := svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "jake@test.local", Password: "password123"}})
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if res.User.Username != "jake" || subjectOf(t, res.User.Token) != "1" {
		t.Errorf("Login = %+v, muốn jake với token sub \"1\"", res.User)
	}

	_, err = svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "jake@test.local", Password: "wrongpassword"}})
	wantAppErr(t, err, http.StatusUnauthorized, "credentials", "invalid")
	_, err = svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "khong-co@test.local", Password: "password123"}})
	wantAppErr(t, err, http.StatusUnauthorized, "credentials", "invalid")
	_, err = svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "", Password: "password123"}})
	wantAppErr(t, err, http.StatusUnprocessableEntity, "email", "can't be blank")
	_, err = svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "jake@test.local", Password: ""}})
	wantAppErr(t, err, http.StatusUnprocessableEntity, "password", "can't be blank")
}

func TestUserService_Current(t *testing.T) {
	svc, _ := newUserService(t)
	register(t, svc, "jake", "password123")

	res, err := svc.Current(context.Background(), 1)
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if res.User.Username != "jake" || res.User.Token == "" {
		t.Errorf("Current = %+v, muốn jake kèm token", res.User)
	}
	_, err = svc.Current(context.Background(), 999)
	wantAppErr(t, err, http.StatusUnauthorized, "token", "is invalid")
}

func TestUserService_UpdateNullableFields(t *testing.T) {
	svc, repo := newUserService(t)
	register(t, svc, "jake", "password123")
	ctx := context.Background()
	update := func(u dto.UpdateUser) dto.User {
		t.Helper()
		res, err := svc.Update(ctx, 1, dto.UpdateUserRequest{User: u})
		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		return res.User
	}
	bioOf := func(u dto.User) string {
		if u.Bio == nil {
			return "<nil>"
		}
		return *u.Bio
	}

	if got := update(dto.UpdateUser{Bio: set("Updated bio")}); bioOf(got) != "Updated bio" {
		t.Errorf("bio = %s, muốn \"Updated bio\"", bioOf(got))
	}
	if got := update(dto.UpdateUser{Image: set("https://example.com/a.jpg")}); bioOf(got) != "Updated bio" {
		t.Errorf("bio = %s sau khi chỉ gửi image, muốn giữ \"Updated bio\" (field không gửi thì không đổi)", bioOf(got))
	}
	if got := update(dto.UpdateUser{Bio: set("")}); got.Bio != nil {
		t.Errorf("bio \"\" → %s, muốn nil (chuỗi rỗng chuẩn hóa thành null)", bioOf(got))
	}
	update(dto.UpdateUser{Bio: set("tạm")})
	if got := update(dto.UpdateUser{Bio: setNull}); got.Bio != nil {
		t.Errorf("bio null → %s, muốn nil", bioOf(got))
	}
	if got := update(dto.UpdateUser{Image: setNull}); got.Image != nil {
		t.Errorf("image null → %q, muốn nil", *got.Image)
	}
	if repo.users[1].Bio != nil || repo.users[1].Image != nil {
		t.Error("giá trị trả về đúng nhưng chưa lưu xuống repo: gọi s.repo.Update")
	}
}

func TestUserService_UpdateUsernameEmail(t *testing.T) {
	svc, repo := newUserService(t)
	register(t, svc, "jake", "password123")
	register(t, svc, "anna", "password123")
	ctx := context.Background()

	res, err := svc.Update(ctx, 1, dto.UpdateUserRequest{User: dto.UpdateUser{Username: set("jake2"), Email: set("jake2@test.local")}})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if res.User.Username != "jake2" || res.User.Email != "jake2@test.local" || repo.users[1].Username != "jake2" {
		t.Errorf("Update = %+v, muốn jake2 / jake2@test.local và đã lưu", res.User)
	}

	for _, field := range []string{"username", "email"} {
		for _, v := range []dto.Optional[string]{set(""), setNull} {
			u := dto.UpdateUser{}
			if field == "username" {
				u.Username = v
			} else {
				u.Email = v
			}
			_, err := svc.Update(ctx, 1, dto.UpdateUserRequest{User: u})
			wantAppErr(t, err, http.StatusUnprocessableEntity, field, "can't be blank")
		}
	}

	_, err = svc.Update(ctx, 1, dto.UpdateUserRequest{User: dto.UpdateUser{Username: set("anna")}})
	wantAppErr(t, err, http.StatusConflict, "username", "has already been taken")
}

func TestUserService_UpdatePassword(t *testing.T) {
	svc, _ := newUserService(t)
	register(t, svc, "jake", "password123")
	ctx := context.Background()

	for _, v := range []dto.Optional[string]{set(""), setNull} {
		_, err := svc.Update(ctx, 1, dto.UpdateUserRequest{User: dto.UpdateUser{Password: v}})
		wantAppErr(t, err, http.StatusUnprocessableEntity, "password", "can't be blank")
	}
	_, err := svc.Update(ctx, 1, dto.UpdateUserRequest{User: dto.UpdateUser{Password: set("short7c")}})
	wantAppErr(t, err, http.StatusUnprocessableEntity, "password", "is too short (minimum is 8 characters)")

	if _, err := svc.Update(ctx, 1, dto.UpdateUserRequest{User: dto.UpdateUser{Password: set("bonjour1")}}); err != nil {
		t.Fatalf("đổi mật khẩu 8 ký tự: %v", err)
	}
	if _, err := svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "jake@test.local", Password: "bonjour1"}}); err != nil {
		t.Errorf("đăng nhập bằng mật khẩu mới: %v (nhớ băm mật khẩu mới trước khi lưu)", err)
	}
	if _, err := svc.Login(ctx, dto.LoginRequest{User: dto.LoginUser{Email: "jake@test.local", Password: "password123"}}); err == nil {
		t.Error("mật khẩu cũ vẫn đăng nhập được sau khi đổi")
	}
}

func TestUserService_UpdateUnknownUser(t *testing.T) {
	svc, _ := newUserService(t)
	_, err := svc.Update(context.Background(), 999, dto.UpdateUserRequest{User: dto.UpdateUser{Bio: set("x")}})
	wantAppErr(t, err, http.StatusUnauthorized, "token", "is invalid")
}
