package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monoposer/dataspan/internal/auth"
)

func TestDataAPIAuthRequiresKeys(t *testing.T) {
	g := &auth.Gateway{AnonKey: "anon-key", Enabled: true}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/public/orders", nil)
	DataAPIAuth(g, next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PGRST301") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestDataAPIAuthPassesWithAnonKey(t *testing.T) {
	g := &auth.Gateway{AnonKey: "anon-key", Enabled: true}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/public/orders", nil)
	req.Header.Set("apikey", "anon-key")
	req.Header.Set("Authorization", "Bearer anon-key")
	DataAPIAuth(g, next).ServeHTTP(rec, req)

	if !called || rec.Code != http.StatusOK {
		t.Fatalf("called=%v status=%d", called, rec.Code)
	}
}

func TestDataAPIAuthSkipsAdmin(t *testing.T) {
	g := &auth.Gateway{AnonKey: "anon-key", Enabled: true}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/servers", nil)
	DataAPIAuth(g, next).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
}

