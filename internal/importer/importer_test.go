package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLToDeclarativePostgres(t *testing.T) {
	sql := `
CREATE TABLE public.users (
  id uuid PRIMARY KEY,
  email varchar(255) NOT NULL,
  created_at timestamptz
);
CREATE TABLE orders (
  id serial PRIMARY KEY,
  user_id uuid NOT NULL
);
`
	doc, err := SQLToDeclarative(sql, SQLOptions{
		Dialect:    DialectPostgres,
		ServerName: "main_pg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].Protocol != "postgres" {
		t.Fatalf("servers: %+v", doc.Servers)
	}
	if len(doc.Tables) != 2 {
		t.Fatalf("tables = %d", len(doc.Tables))
	}
	if doc.Tables[0].Name != "users" || doc.Tables[0].Schema != "public" {
		t.Fatalf("users table: %+v", doc.Tables[0])
	}
	if len(doc.Tables[0].Columns) < 2 {
		t.Fatalf("columns: %+v", doc.Tables[0].Columns)
	}
}

func TestOpenAPIToDeclarative(t *testing.T) {
	spec := []byte(`
openapi: 3.0.3
servers:
  - url: https://api.example.com/v1
paths:
  /orders:
    get: {}
    post:
      operationId: createOrder
  /users/{id}:
    get: {}
`)
	doc, err := OpenAPIToDeclarative(spec, OpenAPIOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Servers) != 1 {
		t.Fatalf("servers: %+v", doc.Servers)
	}
	if doc.Servers[0].Options["endpoint"] != "https://api.example.com/v1" {
		t.Fatalf("endpoint: %+v", doc.Servers[0].Options)
	}
	if len(doc.Tables) == 0 {
		t.Fatal("expected tables")
	}
	if len(doc.Functions) == 0 {
		t.Fatal("expected functions")
	}
}

func TestParseSQLDialect(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want SQLDialect
	}{
		{"pg", DialectPostgres},
		{"mysql", DialectMySQL},
		{"sqlite3", DialectSQLite},
	} {
		got, err := ParseSQLDialect(tc.in)
		if err != nil || got != tc.want {
			t.Fatalf("ParseSQLDialect(%q) = %q, %v", tc.in, got, err)
		}
	}
}

func TestDetectKind(t *testing.T) {
	if got := DetectKind("schema.sql", []byte("CREATE TABLE t (id int)")); got != KindSQL {
		t.Fatalf("got %q", got)
	}
	if got := DetectKind("spec.yaml", []byte("openapi: 3.0.0\npaths: {}\n")); got != KindOpenAPI {
		t.Fatalf("got %q", got)
	}
}

func TestMergeDeclarativeDocs(t *testing.T) {
	a := DeclarativeDoc{
		Servers: []DeclServer{{Name: "a", Protocol: "http"}},
		Tables:  []DeclTable{{Server: "a", Name: "t1", Columns: []DeclColumn{{Name: "id"}}}},
	}
	b := DeclarativeDoc{
		Servers: []DeclServer{{Name: "b", Protocol: "postgres"}},
		Tables:  []DeclTable{{Server: "b", Name: "t2", Columns: []DeclColumn{{Name: "id"}}}},
	}
	merged, err := MergeDeclarativeDocs(a, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged.Servers) != 2 || len(merged.Tables) != 2 {
		t.Fatalf("merged: %+v", merged)
	}
}

func TestConvertWriteRoundTrip(t *testing.T) {
	out, err := Convert(ConvertOptions{
		Kind:  KindSQL,
		Input: []byte("CREATE TABLE items (id INTEGER PRIMARY KEY, name TEXT);"),
		SQL:   SQLOptions{Dialect: DialectSQLite, ServerName: "local"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "protocol: sqlite") {
		t.Fatalf("output: %s", out)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}
	doc, err := ParseDeclarativeYAML(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Tables) != 1 {
		t.Fatalf("tables: %+v", doc.Tables)
	}
}
