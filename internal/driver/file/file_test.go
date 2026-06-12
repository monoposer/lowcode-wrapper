package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

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

func TestSelectYAML(t *testing.T) {
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "items.yaml")
	if err := os.WriteFile(yamlPath, []byte(`
- id: "1"
  name: alpha
- id: "2"
  name: beta
`), 0644); err != nil {
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
			RemoteName: "items.yaml",
			Options:    json.RawMessage(`{"format":"yaml"}`),
		},
		Columns: []models.Column{
			{Name: "id", DataType: "text"},
			{Name: "name", DataType: "text"},
		},
	}

	rows, err := drv.Select(context.Background(), driver.SelectRequest{
		Resolved: resolved,
		Filters:  []postgrest.Filter{{Column: "id", Op: postgrest.OpEq, Value: "2"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "beta" {
		t.Fatalf("rows: %+v", rows)
	}
}

func TestSelectXLSXSheet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "book.xlsx")
	f := excelize.NewFile()
	if err := f.SetSheetName("Sheet1", "Alpha"); err != nil {
		t.Fatal(err)
	}
	idx, err := f.NewSheet("Beta")
	if err != nil {
		t.Fatal(err)
	}
	_ = idx
	if err := f.SetSheetRow("Alpha", "A1", &[]any{"id", "name"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Alpha", "A2", &[]any{"1", "from-alpha"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Beta", "A1", &[]any{"id", "name"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SetSheetRow("Beta", "A2", &[]any{"1", "from-beta"}); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(path); err != nil {
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
			TableName:  "Beta",
			RemoteName: "book.xlsx",
			Options:    json.RawMessage(`{"format":"xlsx"}`),
		},
		Columns: []models.Column{
			{Name: "id", DataType: "text"},
			{Name: "name", DataType: "text"},
		},
	}

	rows, err := drv.Select(context.Background(), driver.SelectRequest{Resolved: resolved})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["name"] != "from-beta" {
		t.Fatalf("rows: %+v", rows)
	}
}
