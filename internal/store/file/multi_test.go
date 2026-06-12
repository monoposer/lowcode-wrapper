package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lowcode-wrapper/internal/store/file"
)

func TestMultiFileDirectoryLoad(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.yaml"), []byte(`
servers:
  - name: api_a
    protocol: http
    options:
      endpoint: https://a.example.com
`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.yml"), []byte(`
tables:
  - server: api_a
    name: orders
    keyColumns: [id]
    columns:
      - name: id
`), 0644); err != nil {
		t.Fatal(err)
	}

	st, err := file.New(testVault(t), file.Config{Path: dir})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	srvs, err := st.ListServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(srvs) != 1 || srvs[0].Name != "api_a" {
		t.Fatalf("servers: %+v", srvs)
	}
	tables, err := st.ListTables(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) != 1 || tables[0].TableName != "orders" {
		t.Fatalf("tables: %+v", tables)
	}
}
