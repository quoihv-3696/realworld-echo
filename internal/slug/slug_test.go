package slug_test

import (
	"regexp"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/slug"
)

func TestMake(t *testing.T) {
	cases := map[string]string{
		"How to train your dragon": "how-to-train-your-dragon",
		"Học Go trong 30 ngày":     "hoc-go-trong-30-ngay",
		"Đường đến Echo v5!!!":     "duong-den-echo-v5",
		"  GORM & N+1: bẫy?  ":     "gorm-n-1-bay",
		"🌏🌏":                       "article",
	}
	for in, want := range cases {
		if got := slug.Make(in); got != want {
			t.Errorf("Make(%q) = %q, muốn %q", in, got, want)
		}
	}
}

func TestWithSuffix(t *testing.T) {
	a, b := slug.WithSuffix("hoc-go"), slug.WithSuffix("hoc-go")
	if !regexp.MustCompile(`^hoc-go-[a-z0-9]{6}$`).MatchString(a) {
		t.Errorf("WithSuffix = %q, muốn hoc-go-xxxxxx", a)
	}
	if a == b {
		t.Errorf("hai lần WithSuffix ra cùng %q", a)
	}
}
