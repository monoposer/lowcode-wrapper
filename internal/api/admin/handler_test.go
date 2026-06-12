package admin_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monoposer/dataspan/internal/api/admin"
	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/store"
	"github.com/monoposer/dataspan/internal/store/file"
)

func TestReadOnlyInFileMode(t *testing.T) {
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
	admin.New(st, store.ModeFile).Register(mux)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/api/tables", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET tables status=%d body=%s", rec.Code, rec.Body.String())
	}
	var tables []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &tables); err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 {
		t.Fatalf("tables=%v", tables)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/admin/api/tables", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST tables status=%d body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body["error"], "file") {
		t.Fatalf("body=%v", body)
	}
}
