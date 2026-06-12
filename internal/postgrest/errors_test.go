package postgrest_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/monoposer/dataspan/internal/postgrest"
	"github.com/monoposer/dataspan/internal/store/errs"
)

func TestMapErrorNotFoundTable(t *testing.T) {
	got := postgrest.MapError(errs.ErrNotFound, postgrest.ErrorContext{
		Schema: "public",
		Table:  "orders",
	})
	if got.Code != "PGRST205" || got.Status != http.StatusNotFound {
		t.Fatalf("got=%+v", got)
	}
	if got.Message != "Could not find the table 'public.orders' in the schema cache" {
		t.Fatalf("message=%q", got.Message)
	}
}

func TestMapErrorNotFoundRPC(t *testing.T) {
	got := postgrest.MapError(errs.ErrNotFound, postgrest.ErrorContext{
		Schema: "public",
		RPC:    "create_order",
	})
	if got.Code != "PGRST202" {
		t.Fatalf("got=%+v", got)
	}
}

func TestMapErrorInvalidJSON(t *testing.T) {
	got := postgrest.MapError(&json.SyntaxError{}, postgrest.ErrorContext{})
	if got.Code != "PGRST102" || got.Status != http.StatusBadRequest {
		t.Fatalf("got=%+v", got)
	}
}

func TestMapErrorPreservesAPIError(t *testing.T) {
	want := postgrest.UnsupportedMethod("PUT")
	got := postgrest.MapError(want, postgrest.ErrorContext{})
	if got.Code != want.Code || got.Status != want.Status {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}
