package integration_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"lowcode-wrapper/internal/api"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/service"

	_ "lowcode-wrapper/internal/driver/filedriver"
	_ "lowcode-wrapper/internal/driver/httpdriver"
	_ "lowcode-wrapper/internal/driver/mysqldriver"
	_ "lowcode-wrapper/internal/driver/pgdriver"
)

func TestFileDriverE2E(t *testing.T) {
	s := initIntegrationEnv(t)

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "items.csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,alpha\n2,beta\n"), 0644); err != nil {
		t.Fatal(err)
	}

	srv, err := s.CreateServer(context.Background(), models.CreateServerRequest{
		Name:     fmt.Sprintf("local_files_%d", time.Now().UnixNano()),
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
