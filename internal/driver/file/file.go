package file

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
	"gopkg.in/yaml.v3"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolFile, New)
}

type Driver struct {
	rootPath string
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.FileServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	root := strings.TrimSpace(opts.RootPath)
	if root == "" {
		return nil, fmt.Errorf("file server %q requires options.rootPath", srv.Name)
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("file rootPath: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("file rootPath must be a directory")
	}
	return &Driver{rootPath: root}, nil
}

func (d *Driver) filePath(resolved *models.ResolvedTable) (path, format string, err error) {
	name := models.RemoteTableName(resolved.Table)
	path = filepath.Join(d.rootPath, name)
	topts, err := models.ParseServerOptions[models.FileTableOptions](resolved.Table.Options)
	if err != nil {
		return "", "", err
	}
	format = strings.ToLower(strings.TrimSpace(topts.Format))
	if format == "" {
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".csv":
			format = "csv"
		case ".json":
			format = "json"
		case ".yaml", ".yml":
			format = "yaml"
		case ".xlsx", ".xlsm":
			format = "xlsx"
		case ".ndjson", ".jsonl":
			format = "ndjson"
		default:
			format = "csv"
		}
	}
	return path, format, nil
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	path, format, err := d.filePath(req.Resolved)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	switch format {
	case "json":
		rows, err = readJSONFile(path)
	case "yaml", "yml":
		rows, err = readYAMLFile(path)
	case "xlsx":
		rows, err = readXLSXFile(path, req.Resolved.Table.TableName)
	case "ndjson":
		rows, err = readNDJSONFile(path)
	default:
		rows, err = readCSVFile(path)
	}
	if err != nil {
		return nil, err
	}
	rows = applyFilters(rows, postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	rows = applySelect(rows, req.Select)
	if req.Offset > 0 && req.Offset < len(rows) {
		rows = rows[req.Offset:]
	} else if req.Offset >= len(rows) {
		rows = nil
	}
	if req.Limit > 0 && len(rows) > req.Limit {
		rows = rows[:req.Limit]
	}
	return postgrest.MapRowsFromRemote(rows, req.Resolved.Columns), nil
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	return nil, fmt.Errorf("file driver: insert not supported in phase 1")
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	return 0, fmt.Errorf("file driver: update not supported")
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	return false, nil, fmt.Errorf("file driver: upsert not supported")
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	return 0, fmt.Errorf("file driver: delete not supported")
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	return nil, fmt.Errorf("file driver: invoke not supported")
}

func readCSVFile(path string) ([]map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, nil
	}
	header := records[0]
	var rows []map[string]any
	for _, rec := range records[1:] {
		row := make(map[string]any, len(header))
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func readYAMLFile(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var arr []map[string]any
	if err := yaml.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}
	var one map[string]any
	if err := yaml.Unmarshal(data, &one); err != nil {
		return nil, err
	}
	return []map[string]any{one}, nil
}

func readXLSXFile(path, tableName string) ([]map[string]any, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sheet, err := resolveXLSXSheet(f, tableName)
	if err != nil {
		return nil, err
	}
	records, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, nil
	}
	header := records[0]
	var rows []map[string]any
	for _, rec := range records[1:] {
		row := make(map[string]any, len(header))
		for i, h := range header {
			h = strings.TrimSpace(h)
			if h == "" {
				continue
			}
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		if len(row) > 0 {
			rows = append(rows, row)
		}
	}
	return rows, nil
}

// resolveXLSXSheet uses table name as sheet name.
func resolveXLSXSheet(f *excelize.File, tableName string) (string, error) {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return "", fmt.Errorf("xlsx: table name is required (must match sheet name)")
	}
	if idx, err := f.GetSheetIndex(tableName); err != nil || idx < 0 {
		return "", fmt.Errorf("xlsx: sheet %q not found", tableName)
	}
	return tableName, nil
}

func readJSONFile(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}
	var one map[string]any
	if err := json.Unmarshal(data, &one); err != nil {
		return nil, err
	}
	return []map[string]any{one}, nil
}

func readNDJSONFile(path string) ([]map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var rows []map[string]any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func applyFilters(rows []map[string]any, filters []postgrest.Filter) []map[string]any {
	if len(filters) == 0 {
		return rows
	}
	var out []map[string]any
	for _, row := range rows {
		if matchFilters(row, filters) {
			out = append(out, row)
		}
	}
	return out
}

func matchFilters(row map[string]any, filters []postgrest.Filter) bool {
	for _, f := range filters {
		val, ok := row[f.Column]
		sval := fmt.Sprint(val)
		switch f.Op {
		case postgrest.OpEq:
			if !ok || sval != f.Value {
				return false
			}
		case postgrest.OpNeq:
			if ok && sval == f.Value {
				return false
			}
		case postgrest.OpIs:
			if f.Value == "null" && ok && val != nil {
				return false
			}
		default:
			if !ok || sval != f.Value {
				return false
			}
		}
	}
	return true
}

func applySelect(rows []map[string]any, cols []string) []map[string]any {
	if len(cols) == 0 {
		return rows
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		m := make(map[string]any, len(cols))
		for _, c := range cols {
			if v, ok := row[c]; ok {
				m[c] = v
			}
		}
		out[i] = m
	}
	return out
}
