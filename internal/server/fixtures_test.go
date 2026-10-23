package server_test

import (
	"testing"

	"github.com/labstack/echo/v5"
	"gorm.io/gorm"

	"github.com/quoihv-3696/realworld-echo/internal/server"
	"github.com/quoihv-3696/realworld-echo/internal/testutil"
)

func newTestServer(t *testing.T) (*echo.Echo, *gorm.DB) {
	t.Helper()
	db := testutil.DB(t)
	e := server.New(db, testutil.JWT(t))
	e.Logger = testutil.Logger(t)
	return e, db
}
