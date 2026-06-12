package filedriver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/postgrest"
)

func TestSelectCSVWithFilter(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "items.csv")
	if err := os.WriteFile(csvPath, []byte("id,name\n1,alpha\n2,beta\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts, _ := json.Marshal(models.FileServerOptions{RootPath: dir})
	drv, err := New(context.Background(), models.Server{
		Name:     "test",
		Protocol: models.ProtocolFile,
		Options:  opts,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	resolved := &models.ResolvedTable{
		Table: models.Table{
			RemoteName: "items.csv",
			Options:    json.RawMessage(`{"format":"csv"}`),
		},
		Columns: []models.Column{
			{Name: "id", DataType: "text"},
			{Name: "name", DataType: "text"},
		},
	}

	rows, err := drv.Select(context.Background(), driver.SelectRequest{
		Resolved: resolved,
		Filters:  []postgrest.Filter{{Column: "id", Op: postgrest.OpEq, Value: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "alpha" {
		t.Fatalf("rows: %+v", rows)
	}
}
