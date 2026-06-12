package rest_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monoposer/dataspan/internal/api/rest"
	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/engine"
	"github.com/monoposer/dataspan/internal/store/file"

	_ "github.com/monoposer/dataspan/internal/driver/file"
)

func TestUpsertNotSupportedOnFileDriver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drivers.yaml")
	const yaml = `servers:
  - name: local
    protocol: file
    enabled: true
    options:
      rootPath: /tmp
tables:
  - server: local
    schema: public
    name: items
    remoteName: items.csv
    columns:
      - name: id
        dataType: text
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAB
	}
	t.Setenv("DATASPAN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st, err := file.New(vault, file.Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	mux := http.NewServeMux()
	rest.New(engine.NewEngine(st)).Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rest/v1/items", strings.NewReader(`{"id":"1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "return=representation, resolution=merge-duplicates")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PGRST000" {
		t.Fatalf("code=%v body=%v", body["code"], body)
	}
	if !strings.Contains(body["message"].(string), "upsert") {
		t.Fatalf("message=%v", body["message"])
	}
}

func TestInsertNotSupportedOnFileDriver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "drivers.yaml")
	const yaml = `servers:
  - name: local
    protocol: file
    enabled: true
    options:
      rootPath: /tmp
tables:
  - server: local
    schema: public
    name: items
    remoteName: items.csv
    columns:
      - name: id
        dataType: text
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAB
	}
	t.Setenv("DATASPAN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st, err := file.New(vault, file.Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	mux := http.NewServeMux()
	rest.New(engine.NewEngine(st)).Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rest/v1/items", strings.NewReader(`{"id":"1"}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "PGRST000" {
		t.Fatalf("code=%v", body["code"])
	}
}
