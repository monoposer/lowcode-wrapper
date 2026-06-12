package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monoposer/dataspan/internal/auth"
)

func TestAdminAuthRequiresKey(t *testing.T) {
	a := &auth.Admin{Key: "admin-secret", Enabled: true}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/servers", nil)
	AdminAuth(a, next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminAuthPassesWithKey(t *testing.T) {
	a := &auth.Admin{Key: "admin-secret", Enabled: true}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/servers", nil)
	req.Header.Set("X-Api-Key", "admin-secret")
	AdminAuth(a, next).ServeHTTP(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Fatalf("called=%v status=%d", called, rec.Code)
	}
}

func TestAdminAuthOpenWhenDisabled(t *testing.T) {
	a := &auth.Admin{}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/servers", nil)
	AdminAuth(a, next).ServeHTTP(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Fatalf("called=%v status=%d", called, rec.Code)
	}
}

func TestAdminAuthSkipsHealth(t *testing.T) {
	a := &auth.Admin{Key: "admin-secret", Enabled: true}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for _, path := range []string{"/health", "/health/ready"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, path, nil)
		AdminAuth(a, next).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("path=%s status=%d", path, rec.Code)
		}
	}
}
