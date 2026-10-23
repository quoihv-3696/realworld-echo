package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/models"
	"github.com/quoihv-3696/realworld-echo/internal/repository"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func TestUserRepository_CreateAndFind(t *testing.T) {
	db := testutil.DB(t)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	u := models.User{Username: "jake", Email: "jake@test.local", Password: "hash"}
	if err := repo.Create(ctx, &u); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID == 0 {
		t.Fatal("sau Create, u.ID vẫn là 0: truyền con trỏ u vào Create để GORM điền ID")
	}
	byEmail, err := repo.FindByEmail(ctx, "jake@test.local")
	if err != nil {
		t.Fatalf("FindByEmail: %v", err)
	}
	if byEmail.ID != u.ID || byEmail.Username != "jake" {
		t.Errorf("FindByEmail = %+v, muốn user jake id %d", byEmail, u.ID)
	}
	byID, err := repo.FindByID(ctx, u.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if byID.Email != "jake@test.local" {
		t.Errorf("FindByID = %+v, muốn email jake@test.local", byID)
	}
}

func TestUserRepository_NotFound(t *testing.T) {
	repo := repository.NewUserRepository(testutil.DB(t))
	if _, err := repo.FindByEmail(context.Background(), "khong-co@test.local"); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindByEmail: err = %v, muốn repository.ErrNotFound", err)
	}
	if _, err := repo.FindByID(context.Background(), 999); !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("FindByID: err = %v, muốn repository.ErrNotFound", err)
	}
}

func TestUserRepository_CreateDuplicate(t *testing.T) {
	db := testutil.DB(t)
	testutil.CreateUser(t, db, "jake", nil)
	repo := repository.NewUserRepository(db)

	cases := []struct{ username, email, field string }{
		{"jake", "khac@test.local", "username"},
		{"khac", "jake@test.local", "email"},
	}
	for _, tc := range cases {
		err := repo.Create(context.Background(), &models.User{Username: tc.username, Email: tc.email, Password: "x"})
		var dup *repository.DuplicateError
		if !errors.As(err, &dup) || dup.Field != tc.field {
			t.Errorf("trùng %s: err = %v, muốn &repository.DuplicateError{Field: %q} (errors.As với *pgconn.PgError, xem ConstraintName)", tc.field, err, tc.field)
		}
	}
}

func TestUserRepository_UpdateWritesNilAndDetectsDuplicate(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", testutil.StrPtr("bio cũ"))
	anna := testutil.CreateUser(t, db, "anna", nil)
	repo := repository.NewUserRepository(db)
	ctx := context.Background()

	jake.Bio = nil
	jake.Username = "jake2"
	if err := repo.Update(ctx, &jake); err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := repo.FindByID(ctx, jake.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Username != "jake2" {
		t.Errorf("username = %q, muốn \"jake2\"", got.Username)
	}
	if got.Bio != nil {
		t.Errorf("bio = %q, muốn NULL: Updates(struct) bỏ qua giá trị nil, hãy dùng Save", *got.Bio)
	}

	jake.Email = anna.Email
	err = repo.Update(ctx, &jake)
	var dup *repository.DuplicateError
	if !errors.As(err, &dup) || dup.Field != "email" {
		t.Errorf("đổi email trùng: err = %v, muốn &repository.DuplicateError{Field: \"email\"}", err)
	}
}

func TestUserRepository_UsesRequestContext(t *testing.T) {
	db := testutil.DB(t)
	jake := testutil.CreateUser(t, db, "jake", nil)
	repo := repository.NewUserRepository(db)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	check := func(name string, err error) {
		t.Helper()
		if err == nil || errors.Is(err, repository.ErrNotFound) {
			t.Errorf("%s với context đã hủy: err = %v, muốn lỗi context (quên r.db.WithContext(ctx)?)", name, err)
		}
	}
	check("Create", repo.Create(ctx, &models.User{Username: "x", Email: "x@test.local", Password: "x"}))
	_, err := repo.FindByID(ctx, jake.ID)
	check("FindByID", err)
	_, err = repo.FindByEmail(ctx, jake.Email)
	check("FindByEmail", err)
	check("Update", repo.Update(ctx, &jake))
}
