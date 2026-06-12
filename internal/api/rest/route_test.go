package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSchemaFromRequest(t *testing.T) {
	cases := []struct {
		method  string
		accept  string
		content string
		want    string
	}{
		{"GET", "analytics", "", "analytics"},
		{"HEAD", "staging", "", "staging"},
		{"POST", "", "private", "private"},
		{"PATCH", "", "private", "private"},
		{"DELETE", "", "private", "private"},
		{"GET", "", "", "public"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, "/rest/v1/items", nil)
		if c.accept != "" {
			req.Header.Set("Accept-Profile", c.accept)
		}
		if c.content != "" {
			req.Header.Set("Content-Profile", c.content)
		}
		if got := schemaFromRequest(req); got != c.want {
			t.Fatalf("%s accept=%q content=%q: got %q want %q", c.method, c.accept, c.content, got, c.want)
		}
	}
}

func TestParseDataAPIResource(t *testing.T) {
	reqWithProfile := func(method, accept, content string) *http.Request {
		r := httptest.NewRequest(method, "/rest/v1/items", nil)
		if accept != "" {
			r.Header.Set("Accept-Profile", accept)
		}
		if content != "" {
			r.Header.Set("Content-Profile", content)
		}
		return r
	}
	req := func(method, accept, content string) *http.Request {
		return reqWithProfile(method, accept, content)
	}

	cases := []struct {
		path   string
		req    *http.Request
		ok     bool
		schema string
		table  string
		rpc    string
	}{
		{"/rest/v1/items", req("GET", "public", ""), true, "public", "items", ""},
		{"/v1/orders", req("GET", "", ""), true, "public", "orders", ""},
		{"/rest/v1/public/items", req("GET", "", ""), false, "", "", ""},
		{"/rest/v1/events", req("GET", "analytics", ""), true, "analytics", "events", ""},
		{"/rest/v1/rpc/create_order", httptest.NewRequest("POST", "/rest/v1/rpc/create_order", nil), true, "public", "", "create_order"},
		{"/rest/v1/foreign_tables", reqWithProfile("GET", "pg_catalog", ""), true, "pg_catalog", "foreign_tables", ""},
		{"/rest/v1/", req("GET", "", ""), false, "", "", ""},
		{"/rest/v1/a/b/c", req("GET", "", ""), false, "", "", ""},
	}
	for _, c := range cases {
		got, ok := parseDataAPIResource(c.path, c.req)
		if ok != c.ok {
			t.Fatalf("%s: ok=%v want %v", c.path, ok, c.ok)
		}
		if !c.ok {
			continue
		}
		if got.Schema != c.schema || got.Table != c.table || got.RPC != c.rpc {
			t.Fatalf("%s: got schema=%q table=%q rpc=%q want schema=%q table=%q rpc=%q",
				c.path, got.Schema, got.Table, got.RPC, c.schema, c.table, c.rpc)
		}
	}
}
