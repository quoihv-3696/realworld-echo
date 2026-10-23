package auth_test

import (
	"strings"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/auth"
)

func TestHashPassword(t *testing.T) {
	h1, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !strings.HasPrefix(h1, "$2a$10$") {
		t.Errorf("hash = %q, muốn chuỗi bcrypt dạng $2a$10$... (bcrypt.GenerateFromPassword với bcrypt.DefaultCost)", h1)
	}
	h2, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if h1 == h2 {
		t.Error("băm cùng một mật khẩu hai lần ra cùng một hash: bcrypt tự sinh salt ngẫu nhiên, đừng tự viết hàm băm")
	}
}

func TestHashPassword_TooLongReturnsError(t *testing.T) {
	if _, err := auth.HashPassword(strings.Repeat("a", 73)); err == nil {
		t.Error("mật khẩu 73 byte phải trả lỗi (bcrypt chỉ nhận tối đa 72 byte); đừng bỏ qua err của GenerateFromPassword")
	}
}

func TestCheckPassword(t *testing.T) {
	hash, err := auth.HashPassword("password123")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !auth.CheckPassword(hash, "password123") {
		t.Error("CheckPassword(hash, đúng mật khẩu) = false, muốn true")
	}
	if auth.CheckPassword(hash, "password124") {
		t.Error("CheckPassword(hash, sai mật khẩu) = true, muốn false")
	}
	if auth.CheckPassword("khong-phai-hash", "password123") {
		t.Error("CheckPassword với hash hỏng = true, muốn false")
	}
}
