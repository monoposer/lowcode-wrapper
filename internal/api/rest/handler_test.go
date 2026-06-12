package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/monoposer/dataspan/internal/engine"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/file"
)

func TestOpenAPI(t *testing.T) {
	st := testFileStore(t)
	if _, _, err := st.CreateTable(context.Background(), models.CreateTableRequest{
		ServerName: seedTestServer(t, st),
		SchemaName: "public",
		TableName:  "orders",
		Columns:    []models.ColumnInput{{Name: "id"}, {Name: "amount"}},
	}); err != nil {
		t.Fatal(err)
	}
	h := New(engine.NewEngine(st))
	mux := http.NewServeMux()
	h.Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/openapi+json" {
		t.Fatalf("content-type=%q", ct)
	}
	var spec map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("openapi=%v", spec["openapi"])
	}
}

func testFileStore(t *testing.T) *file.Store {
	t.Helper()
	st, err := file.New(nil, file.Config{Path: filepath.Join(t.TempDir(), "meta.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}

func seedTestServer(t *testing.T, st *file.Store) string {
	t.Helper()
	srv, err := st.CreateServer(context.Background(), models.CreateServerRequest{
		Name:     "api",
		Protocol: models.ProtocolHTTP,
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv.Name
}
