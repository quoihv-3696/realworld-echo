package testutil

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type Response struct {
	Status int
	Body   string
	JSON   map[string]any
}

type Option func(*http.Request) *http.Request

func WithContext(ctx context.Context) Option {
	return func(r *http.Request) *http.Request { return r.WithContext(ctx) }
}

func Do(t *testing.T, h http.Handler, method, path string, opts ...Option) Response {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	for _, opt := range opts {
		req = opt(req)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	res := Response{Status: rec.Code, Body: rec.Body.String()}
	if rec.Body.Len() > 0 {
		if err := json.Unmarshal(rec.Body.Bytes(), &res.JSON); err != nil {
			t.Fatalf("%s %s: body không phải JSON object: %q", method, path, res.Body)
		}
	}
	return res
}
