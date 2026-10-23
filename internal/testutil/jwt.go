package testutil

import (
	"testing"
	"time"

	"github.com/quoihv-3696/realworld-echo/internal/auth"
)

const JWTSecret = "test-secret-for-go-tests-32-bytes-long"

func JWT(t *testing.T) *auth.JWT {
	t.Helper()
	j, err := auth.NewJWT(JWTSecret, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return j
}
