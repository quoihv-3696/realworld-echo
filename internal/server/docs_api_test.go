package server_test

import (
	"bufio"
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/quoihv-3696/realworld-echo/api"
)

func TestDocs_ServeSpecAndUI(t *testing.T) {
	e, _ := newTestServer(t)
	cases := []struct {
		path, contentType string
		contains          []string
	}{
		{"/openapi.yml", "application/yaml", []string{"openapi: 3", "- url: /api"}},
		{"/docs", "text/html", []string{"swagger-ui", `url: "/openapi.yml"`}},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s: %d, muốn 200", tc.path, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, tc.contentType) {
			t.Errorf("GET %s: Content-Type %q, muốn %s", tc.path, ct, tc.contentType)
		}
		for _, s := range tc.contains {
			if !strings.Contains(rec.Body.String(), s) {
				t.Errorf("GET %s: body thiếu %q", tc.path, s)
			}
		}
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/openapi.yml", nil))
	if strings.Contains(rec.Body.String(), "api.realworld.show") {
		t.Error("/openapi.yml vẫn trỏ server công khai api.realworld.show: Try it out sẽ gửi request ra ngoài, không tới máy bạn")
	}
	if !bytes.Contains(api.Spec, []byte("api.realworld.show")) {
		t.Error("api.Spec phải là file openapi.yml gốc, không sửa (đổi server lúc phục vụ, không đổi file)")
	}
}

type operation struct{ method, path string }

func specOperations(t *testing.T) map[operation]bool {
	t.Helper()
	pathRe := regexp.MustCompile(`^  (/\S*):\s*$`)
	methodRe := regexp.MustCompile(`^    (get|post|put|delete|patch):\s*$`)
	ops := map[operation]bool{}
	inPaths, current := false, ""
	sc := bufio.NewScanner(bytes.NewReader(api.Spec))
	for sc.Scan() {
		line := sc.Text()
		if line == "paths:" {
			inPaths = true
			continue
		}
		if inPaths && line != "" && !strings.HasPrefix(line, " ") {
			break
		}
		if m := pathRe.FindStringSubmatch(line); inPaths && m != nil {
			current = m[1]
		} else if m := methodRe.FindStringSubmatch(line); inPaths && m != nil && current != "" {
			ops[operation{strings.ToUpper(m[1]), current}] = true
		}
	}
	if len(ops) == 0 {
		t.Fatal("không đọc được route nào từ openapi.yml")
	}
	return ops
}

func TestDocs_SpecMatchesRoutes(t *testing.T) {
	e, _ := newTestServer(t)
	paramRe := regexp.MustCompile(`:(\w+)`)
	registered := map[operation]bool{}
	for _, r := range e.Router().Routes() {
		if !strings.HasPrefix(r.Path, "/api/") {
			continue
		}
		path := paramRe.ReplaceAllString(strings.TrimPrefix(r.Path, "/api"), "{$1}")
		registered[operation{r.Method, path}] = true
	}
	spec := specOperations(t)
	for op := range spec {
		if !registered[op] {
			t.Errorf("%s %s có trong openapi.yml nhưng server chưa đăng ký", op.method, op.path)
		}
	}
	for op := range registered {
		if !spec[op] {
			t.Errorf("%s /api%s đã đăng ký nhưng không có trong openapi.yml", op.method, op.path)
		}
	}
	if len(spec) != 19 {
		t.Errorf("spec có %d thao tác, muốn 19", len(spec))
	}
}
