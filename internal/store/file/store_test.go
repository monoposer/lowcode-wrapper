package file_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/store/file"
)

func testVault(t *testing.T) *auth.Vault {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAB
	}
	t.Setenv("WRAPPER_MASTER_KEY", base64.StdEncoding.EncodeToString(key))
	v, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestFileStoreCreateAndResolve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "meta.yaml")
	st, err := file.New(testVault(t), file.Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	srv, err := st.CreateServer(context.Background(), models.CreateServerRequest{
		Name:     "local",
		Protocol: models.ProtocolFile,
		Options:  json.RawMessage(`{"rootPath":"/tmp"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, cols, err := st.CreateTable(context.Background(), models.CreateTableRequest{
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
	if len(cols) != 2 {
		t.Fatalf("columns = %d", len(cols))
	}

	rt, err := st.ResolveTable(context.Background(), "public", "items")
	if err != nil {
		t.Fatal(err)
	}
	if rt.Server.Name != "local" || len(rt.Columns) != 2 {
		t.Fatalf("resolved: %+v", rt)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("expected yaml file to be written")
	}
}
