package postgrest_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/monoposer/dataspan/internal/postgrest"
)

func TestParseQueryFilters(t *testing.T) {
	q := postgrest.ParseQuery(url.Values{
		"id":     {"eq.42"},
		"status": {"eq.active"},
		"select": {"id,name"},
		"limit":  {"10"},
		"order":  {"created_at.desc"},
	})
	if len(q.Filters) != 2 {
		t.Fatalf("filters: got %d want 2", len(q.Filters))
	}
	if q.Filters[0].Column != "id" || q.Filters[0].Op != postgrest.OpEq || q.Filters[0].Value != "42" {
		t.Fatalf("id filter: %+v", q.Filters[0])
	}
	if len(q.Select) != 2 || q.Select[0] != "id" {
		t.Fatalf("select: %+v", q.Select)
	}
	if q.Limit != 10 {
		t.Fatalf("limit: %d", q.Limit)
	}
	if len(q.Order) != 1 || !q.Order[0].Desc || q.Order[0].Column != "created_at" {
		t.Fatalf("order: %+v", q.Order)
	}
}

func TestParsePrefer(t *testing.T) {
	h := http.Header{}
	h.Set("Prefer", "return=representation, resolution=merge-duplicates")
	p := postgrest.ParsePrefer(h)
	if !p.Representation || !p.Upsert {
		t.Fatalf("prefer: %+v", p)
	}
}
