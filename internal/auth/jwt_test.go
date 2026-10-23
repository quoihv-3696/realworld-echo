package auth_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v5"

	"github.com/quoihv-3696/realworld-echo/internal/apperr"
	"github.com/quoihv-3696/realworld-echo/internal/auth"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func newJWT(t *testing.T, ttl time.Duration) *auth.JWT {
	t.Helper()
	j, err := auth.NewJWT(testutil.JWTSecret, ttl)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func signRaw(t *testing.T, method jwt.SigningMethod, secret, sub string) string {
	t.Helper()
	claims := jwt.RegisteredClaims{Subject: sub, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	s, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNewJWT_RejectsShortSecret(t *testing.T) {
	if _, err := auth.NewJWT("ngan-qua", time.Hour); err == nil {
		t.Error("secret 8 byte phải bị từ chối")
	}
}

func TestSign_Claims(t *testing.T) {
	token, err := newJWT(t, time.Hour).Sign(42)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	var claims jwt.RegisteredClaims
	_, err = jwt.ParseWithClaims(token, &claims, func(*jwt.Token) (any, error) { return []byte(testutil.JWTSecret), nil },
		jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		t.Fatalf("token không đọc lại được bằng secret: %v (ký bằng jwt.SigningMethodHS256 và j.secret?)", err)
	}
	if claims.Subject != "42" {
		t.Errorf("sub = %q, muốn \"42\" (strconv.FormatUint(uint64(userID), 10))", claims.Subject)
	}
	if claims.IssuedAt == nil {
		t.Error("thiếu iat (IssuedAt)")
	}
	if claims.ExpiresAt == nil {
		t.Fatal("thiếu exp (ExpiresAt): token không bao giờ hết hạn là lỗ hổng")
	}
	if d := time.Until(claims.ExpiresAt.Time); d < 59*time.Minute || d > 61*time.Minute {
		t.Errorf("exp cách hiện tại %v, muốn ≈ ttl (1h): ExpiresAt = now + j.ttl", d.Round(time.Second))
	}
}

func authServer(j *auth.JWT) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = apperr.Handler
	who := func(c *echo.Context) error {
		id, ok := auth.UserID(c)
		return c.JSON(http.StatusOK, map[string]any{"id": id, "ok": ok})
	}
	e.GET("/required", who, j.Required())
	e.GET("/optional", who, j.Optional())
	return e
}

func withAuthHeader(v string) testutil.Option {
	return func(r *http.Request) *http.Request {
		r.Header.Set("Authorization", v)
		return r
	}
}

func TestMiddleware(t *testing.T) {
	j := newJWT(t, time.Hour)
	good, err := j.Sign(42)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	expired, err := newJWT(t, -time.Minute).Sign(42)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	e := authServer(j)

	cases := []struct {
		name, path, header string
		status             int
		tokenErr           string
		id                 float64
	}{
		{"bắt buộc, không có header", "/required", "", 401, "is missing", 0},
		{"bắt buộc, scheme Bearer", "/required", "Bearer " + good, 401, "is missing", 0},
		{"bắt buộc, Token hợp lệ", "/required", "Token " + good, 200, "", 42},
		{"bắt buộc, Token rác", "/required", "Token abc.def.ghi", 401, "is invalid", 0},
		{"bắt buộc, Token hết hạn", "/required", "Token " + expired, 401, "is invalid", 0},
		{"bắt buộc, sai secret", "/required", "Token " + signRaw(t, jwt.SigningMethodHS256, "secret-khac-secret-khac-secret-khac", "42"), 401, "is invalid", 0},
		{"bắt buộc, ký HS512", "/required", "Token " + signRaw(t, jwt.SigningMethodHS512, testutil.JWTSecret, "42"), 401, "is invalid", 0},
		{"bắt buộc, sub không phải số", "/required", "Token " + signRaw(t, jwt.SigningMethodHS256, testutil.JWTSecret, "abc"), 401, "is invalid", 0},
		{"tùy chọn, không có header", "/optional", "", 200, "", 0},
		{"tùy chọn, Token hợp lệ", "/optional", "Token " + good, 200, "", 42},
		{"tùy chọn, Token rác", "/optional", "Token abc.def.ghi", 401, "is invalid", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var opts []testutil.Option
			if tc.header != "" {
				opts = append(opts, withAuthHeader(tc.header))
			}
			res := testutil.Do(t, e, http.MethodGet, tc.path, opts...)
			if res.Status != tc.status {
				t.Fatalf("status = %d, muốn %d. Body: %s", res.Status, tc.status, res.Body)
			}
			if tc.tokenErr != "" {
				errs, _ := res.JSON["errors"].(map[string]any)
				msgs, _ := errs["token"].([]any)
				if len(msgs) == 0 || msgs[0] != tc.tokenErr {
					t.Errorf("body = %s, muốn {\"errors\":{\"token\":[%q]}}", res.Body, tc.tokenErr)
				}
				return
			}
			if res.JSON["id"] != tc.id || res.JSON["ok"] != (tc.id != 0) {
				t.Errorf("auth.UserID trả %s, muốn id %v, ok %v", res.Body, tc.id, tc.id != 0)
			}
		})
	}
}
