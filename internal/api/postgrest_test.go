package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/monoposer/dataspan/internal/postgrest"
	"github.com/monoposer/dataspan/internal/store/errs"
)

func TestWritePostgRESTError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/rest/v1/orders", nil)

	writePostgRESTError(rec, req, errs.ErrNotFound, postgrest.ErrorContext{
		Schema: "public",
		Table:  "orders",
	})

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type=%q", ct)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PGRST205" {
		t.Fatalf("code=%v", body["code"])
	}
	if body["message"] == nil || body["message"] == "" {
		t.Fatalf("message missing")
	}
	for _, key := range []string{"details", "hint"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing %q", key)
		}
	}
}

func TestStripDataAPIPrefix(t *testing.T) {
	cases := map[string]string{
		"/v1/items":              "items",
		"/rest/v1/public/orders": "public/orders",
		"/rest/v1/rpc/fn":        "rpc/fn",
	}
	for in, want := range cases {
		if got := stripDataAPIPrefix(in); got != want {
			t.Fatalf("%s: got %q want %q", in, got, want)
		}
	}
}
