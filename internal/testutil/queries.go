package testutil

import (
	"sync/atomic"
	"testing"

	"gorm.io/gorm"
)

func QueryCounter(t *testing.T, db *gorm.DB) func() int64 {
	t.Helper()
	var n atomic.Int64
	inc := func(db *gorm.DB) {
		if !db.DryRun {
			n.Add(1)
		}
	}
	if err := db.Callback().Query().After("gorm:query").Register("testutil:count_query", inc); err != nil {
		t.Fatal(err)
	}
	if err := db.Callback().Row().After("gorm:row").Register("testutil:count_row", inc); err != nil {
		t.Fatal(err)
	}
	return func() int64 { return n.Swap(0) }
}
