package gentypes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/models"
)

// MetaClient loads schema metadata from the Admin API (/api/*).
type MetaClient struct {
	BaseURL string
	APIKey  string
	Token   string
	HTTP    *http.Client
}

// TableMeta is a registered table with ordered columns.
type TableMeta struct {
	SchemaName string
	TableName  string
	RemoteName string
	ServerName string
	KeyColumns []string
	Columns    []ColumnMeta
}

// ColumnMeta is a column on a registered table.
type ColumnMeta struct {
	Name       string
	DataType   string
	RemoteName string
	Nullable   bool
	Position   int
}

// FunctionMeta is an RPC mapping.
type FunctionMeta struct {
	SchemaName string
	Name       string
	Operation  string
	ServerName string
	Method     string
	RemotePath string
}

// SchemaSnapshot is metadata grouped by PostgREST schema profile (schema_name).
type SchemaSnapshot struct {
	Schemas map[string]SchemaTables
}

// SchemaTables holds tables and functions for one schema profile.
type SchemaTables struct {
	Tables    map[string]TableMeta
	Functions map[string]FunctionMeta
}

// Fetch loads metadata, optionally restricted to schema names (empty = all).
func (c *MetaClient) Fetch(ctx context.Context, schemas []string) (*SchemaSnapshot, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	var tables []models.Table
	if err := c.getJSON(ctx, base+"/api/tables", &tables); err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	var functions []models.Function
	if err := c.getJSON(ctx, base+"/api/functions", &functions); err != nil {
		return nil, fmt.Errorf("list functions: %w", err)
	}

	snap := &SchemaSnapshot{Schemas: map[string]SchemaTables{}}
	for _, tbl := range tables {
		schema := tbl.SchemaName
		if schema == "" {
			schema = "public"
		}
		if !schemaAllowed(schema, schemas) {
			continue
		}
		cols, err := c.listColumns(ctx, base, tbl.ID)
		if err != nil {
			return nil, fmt.Errorf("columns for %s.%s: %w", schema, tbl.TableName, err)
		}
		st := snap.ensureSchema(schema)
		st.Tables[tbl.TableName] = TableMeta{
			SchemaName: schema,
			TableName:  tbl.TableName,
			RemoteName: models.RemoteTableName(tbl),
			ServerName: tbl.ServerName,
			KeyColumns: append([]string(nil), tbl.KeyColumns...),
			Columns:    cols,
		}
	}
	for _, fn := range functions {
		schema := fn.SchemaName
		if schema == "" {
			schema = "public"
		}
		if !schemaAllowed(schema, schemas) {
			continue
		}
		st := snap.ensureSchema(schema)
		st.Functions[fn.Name] = FunctionMeta{
			SchemaName: schema,
			Name:       fn.Name,
			Operation:  fn.Operation,
			ServerName: fn.ServerName,
			Method:     fn.Method,
			RemotePath: fn.RemotePath,
		}
	}
	return snap, nil
}

func (c *MetaClient) listColumns(ctx context.Context, base string, tableID uuid.UUID) ([]ColumnMeta, error) {
	var cols []models.Column
	if err := c.getJSON(ctx, fmt.Sprintf("%s/api/tables/%s/columns", base, tableID), &cols); err != nil {
		return nil, err
	}
	out := make([]ColumnMeta, len(cols))
	for i, col := range cols {
		out[i] = ColumnMeta{
			Name:       col.Name,
			DataType:   col.DataType,
			RemoteName: models.RemoteColumnName(col),
			Nullable:   col.Nullable,
			Position:   col.Position,
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Position == out[j].Position {
			return out[i].Name < out[j].Name
		}
		return out[i].Position < out[j].Position
	})
	return out, nil
}

func (s *SchemaSnapshot) ensureSchema(name string) SchemaTables {
	if s.Schemas == nil {
		s.Schemas = map[string]SchemaTables{}
	}
	st, ok := s.Schemas[name]
	if !ok {
		st = SchemaTables{
			Tables:    map[string]TableMeta{},
			Functions: map[string]FunctionMeta{},
		}
		s.Schemas[name] = st
	}
	return st
}

func schemaAllowed(name string, schemas []string) bool {
	if len(schemas) == 0 {
		return true
	}
	for _, s := range schemas {
		if strings.TrimSpace(s) == name {
			return true
		}
	}
	return false
}

func (c *MetaClient) getJSON(ctx context.Context, url string, dest any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if c.APIKey != "" {
		req.Header.Set("apikey", c.APIKey)
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s: %s", url, resp.Status, strings.TrimSpace(string(body)))
	}
	if dest == nil {
		return nil
	}
	return json.Unmarshal(body, dest)
}
