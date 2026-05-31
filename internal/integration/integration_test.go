package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"lowcode-wrapper/internal/api"
	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/service"
	store "lowcode-wrapper/internal/store/postgres"

	_ "lowcode-wrapper/internal/driver/filedriver"
	_ "lowcode-wrapper/internal/driver/httpdriver"
	_ "lowcode-wrapper/internal/driver/mysqldriver"
	_ "lowcode-wrapper/internal/driver/pgdriver"
)

func TestFileDriverE2E(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = 1
	}
	os.Setenv("WRAPPER_MASTER_KEY", base64.StdEncoding.EncodeToString(key))

	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.NewFromEnv(vault)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "items.csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,alpha\n2,beta\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srv, err := s.CreateServer(context.Background(), models.CreateServerRequest{
		Name:     "local_files",
		Protocol: models.ProtocolFile,
		Options:  json.RawMessage(`{"rootPath":"` + dir + `"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.CreateTable(context.Background(), models.CreateTableRequest{
		ServerName: srv.Name,
		SchemaName: "public",
		TableName:  "items",
		RemoteName: "items.csv",
		KeyColumns: []string{"id"},
		Options:    json.RawMessage(`{"format":"csv"}`),
		Columns: []models.ColumnInput{
			{Name: "id", DataType: "text"},
			{Name: "name", DataType: "text"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	engine := service.NewEngine(s)
	mux := http.NewServeMux()
	api.NewAdminHandler(s).Register(mux)
	api.NewPostgRESTHandler(engine).Register(mux)
	ts := httptest.NewServer(api.CORS(mux))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/public/items?id=eq.1")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	var rows []map[string]any
	if err := json.Unmarshal(body, &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "alpha" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestHTTPDriverE2E(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = 2
	}
	os.Setenv("WRAPPER_MASTER_KEY", base64.StdEncoding.EncodeToString(key))

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/orders":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":"1","total_amount":"99"}]`))
		case r.Method == http.MethodPost && r.URL.Path == "/orders":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"2","total_amount":"50"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(mock.Close)

	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.NewFromEnv(vault)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	srv, err := s.CreateServer(context.Background(), models.CreateServerRequest{
		Name:     "mock_api",
		Protocol: models.ProtocolHTTP,
		Options:  json.RawMessage(`{"endpoint":"` + mock.URL + `"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = s.CreateTable(context.Background(), models.CreateTableRequest{
		ServerName: srv.Name,
		SchemaName: "public",
		TableName:  "orders",
		RemoteName: "orders",
		KeyColumns: []string{"id"},
		Columns: []models.ColumnInput{
			{Name: "id", DataType: "text"},
			{Name: "amount", DataType: "text", RemoteName: "total_amount"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	engine := service.NewEngine(s)
	mux := http.NewServeMux()
	api.NewPostgRESTHandler(engine).Register(mux)
	ts := httptest.NewServer(api.CORS(mux))
	t.Cleanup(ts.Close)

	resp, err := http.Get(ts.URL + "/v1/public/orders")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET status %d: %s", resp.StatusCode, body)
	}

	postBody := bytes.NewBufferString(`{"id":"2","amount":"50"}`)
	postResp, err := http.Post(ts.URL+"/v1/public/orders", "application/json", postBody)
	if err != nil {
		t.Fatal(err)
	}
	defer postResp.Body.Close()
	if postResp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(postResp.Body)
		t.Fatalf("POST status %d: %s", postResp.StatusCode, b)
	}
}
