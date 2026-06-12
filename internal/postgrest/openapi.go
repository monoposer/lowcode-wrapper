package postgrest

import (
	"context"
	"fmt"
	"strings"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store"
	"github.com/monoposer/dataspan/internal/version"
)

// OpenAPIOptions configures generated PostgREST-style Swagger 2.0 output.
type OpenAPIOptions struct {
	Host     string
	BasePath string // e.g. /rest/v1
	Schema   string // optional Accept-Profile filter; empty = all schemas
}

// BuildOpenAPI returns a PostgREST-compatible Swagger 2.0 document from store metadata.
func BuildOpenAPI(ctx context.Context, s store.Store, opts OpenAPIOptions) (map[string]any, error) {
	tables, err := s.ListTables(ctx)
	if err != nil {
		return nil, err
	}
	functions, err := s.ListFunctions(ctx)
	if err != nil {
		return nil, err
	}

	if opts.Schema != "" {
		tables = filterTablesBySchema(tables, opts.Schema)
		functions = filterFunctionsBySchema(functions, opts.Schema)
	}

	basePath := opts.BasePath
	if basePath == "" {
		basePath = "/rest/v1"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	paths := map[string]any{}
	definitions := map[string]any{}

	tablePathCount := map[string]int{}
	for _, tbl := range tables {
		tablePathCount[tbl.TableName]++
	}

	for _, tbl := range tables {
		pathKey := tablePath(tbl, tablePathCount[tbl.TableName] > 1)
		cols, err := s.ListColumns(ctx, tbl.SchemaName, tbl.TableName)
		if err != nil {
			return nil, err
		}
		defName := definitionName(tbl)
		definitions[defName] = tableDefinition(tbl, cols)
		paths[pathKey] = tablePathItem(tbl, defName)
	}

	for _, fn := range functions {
		pathKey := fmt.Sprintf("/rpc/%s", fn.Name)
		if pathItem, ok := paths[pathKey].(map[string]any); ok {
			pathItem["post"] = rpcOperation(fn, pathKey)
			continue
		}
		paths[pathKey] = map[string]any{
			"post": rpcOperation(fn, pathKey),
		}
	}

	title := "dataspan API"
	if opts.Schema != "" {
		title = fmt.Sprintf("dataspan API (%s)", opts.Schema)
	}

	return map[string]any{
		"swagger": "2.0",
		"info": map[string]any{
			"title":       title,
			"description": "PostgREST-style foreign data wrapper sidecar (tables, columns, functions).",
			"version":     version.Version,
		},
		"host":        opts.Host,
		"basePath":    basePath,
		"schemes":     []string{"http", "https"},
		"consumes":    []string{"application/json", "application/vnd.pgrst.object+json"},
		"produces":    []string{"application/json", "application/vnd.pgrst.object+json"},
		"paths":       paths,
		"definitions": definitions,
	}, nil
}

func filterTablesBySchema(tables []models.Table, schema string) []models.Table {
	out := make([]models.Table, 0, len(tables))
	for _, t := range tables {
		if t.SchemaName == schema {
			out = append(out, t)
		}
	}
	return out
}

func filterFunctionsBySchema(functions []models.Function, schema string) []models.Function {
	out := make([]models.Function, 0, len(functions))
	for _, f := range functions {
		if f.SchemaName == schema {
			out = append(out, f)
		}
	}
	return out
}

func tablePath(tbl models.Table, ambiguous bool) string {
	if ambiguous {
		return fmt.Sprintf("/%s/%s", tbl.SchemaName, tbl.TableName)
	}
	return "/" + tbl.TableName
}

func definitionName(tbl models.Table) string {
	if tbl.SchemaName == "" || tbl.SchemaName == "public" {
		return tbl.TableName
	}
	return tbl.SchemaName + "." + tbl.TableName
}

func tableDefinition(tbl models.Table, cols []models.Column) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, col := range cols {
		props[col.Name] = columnSchema(col)
		if !col.Nullable {
			required = append(required, col.Name)
		}
	}
	def := map[string]any{
		"type":       "object",
		"properties": props,
		"description": fmt.Sprintf(
			"Foreign table %s.%s (server: %s, remote: %s)",
			tbl.SchemaName, tbl.TableName, tbl.ServerName, models.RemoteTableName(tbl),
		),
	}
	if len(required) > 0 {
		def["required"] = required
	}
	return def
}

func columnSchema(col models.Column) map[string]any {
	schema := map[string]any{
		"description": columnDescription(col),
	}
	switch strings.ToLower(col.DataType) {
	case "integer", "int", "bigint", "smallint", "numeric", "number", "float", "double":
		schema["type"] = "number"
	case "boolean", "bool":
		schema["type"] = "boolean"
	case "json", "jsonb", "object":
		schema["type"] = "object"
	case "array":
		schema["type"] = "array"
		schema["items"] = map[string]any{"type": "string"}
	default:
		schema["type"] = "string"
	}
	if format := swaggerFormat(col.DataType); format != "" {
		schema["format"] = format
	}
	return schema
}

func swaggerFormat(dataType string) string {
	switch strings.ToLower(dataType) {
	case "uuid":
		return "uuid"
	case "date":
		return "date"
	case "timestamp", "timestamptz", "datetime":
		return "date-time"
	default:
		return ""
	}
}

func columnDescription(col models.Column) string {
	if col.RemoteName != "" && col.RemoteName != col.Name {
		return fmt.Sprintf("remote: %s", col.RemoteName)
	}
	return ""
}

func tablePathItem(tbl models.Table, defName string) map[string]any {
	profileParam := profileHeaderParam(tbl.SchemaName)
	return map[string]any{
		"get": map[string]any{
			"tags":        []string{tbl.SchemaName},
			"summary":     fmt.Sprintf("Read rows from %s", tbl.TableName),
			"parameters":  []any{profileParam},
			"responses":   jsonArrayResponse(defName),
			"produces":    []string{"application/json"},
		},
		"post": map[string]any{
			"tags":        []string{tbl.SchemaName},
			"summary":     fmt.Sprintf("Insert into %s", tbl.TableName),
			"parameters":  []any{profileParam, bodyParam(defName)},
			"responses":   jsonObjectResponse(defName),
			"consumes":    []string{"application/json"},
		},
		"patch": map[string]any{
			"tags":        []string{tbl.SchemaName},
			"summary":     fmt.Sprintf("Update %s", tbl.TableName),
			"parameters":  []any{profileParam, bodyParam(defName)},
			"responses":   affectedResponse(),
			"consumes":    []string{"application/json"},
		},
		"delete": map[string]any{
			"tags":        []string{tbl.SchemaName},
			"summary":     fmt.Sprintf("Delete from %s", tbl.TableName),
			"parameters":  []any{profileParam},
			"responses":   affectedResponse(),
		},
	}
}

func rpcOperation(fn models.Function, _ string) map[string]any {
	params := []any{
		map[string]any{
			"name":        "schema",
			"in":          "query",
			"type":        "string",
			"description": fmt.Sprintf("Schema profile (default %q)", fn.SchemaName),
			"default":     fn.SchemaName,
		},
	}
	if fn.Operation == "invoke" {
		params = append(params, map[string]any{
			"name":     "body",
			"in":       "body",
			"required": false,
			"schema": map[string]any{
				"type": "object",
			},
		})
	}
	return map[string]any{
		"tags":        []string{fn.SchemaName},
		"summary":     fmt.Sprintf("RPC %s (%s)", fn.Name, fn.Operation),
		"description": fmt.Sprintf("server: %s", fn.ServerName),
		"parameters":  params,
		"responses": map[string]any{
			"200": map[string]any{
				"description": "OK",
				"schema":      map[string]any{"type": "object"},
			},
		},
		"consumes": []string{"application/json"},
		"produces": []string{"application/json"},
	}
}

func profileHeaderParam(schema string) map[string]any {
	return map[string]any{
		"name":        "Accept-Profile",
		"in":          "header",
		"type":        "string",
		"description": "Schema profile",
		"default":     schema,
	}
}

func bodyParam(defName string) map[string]any {
	return map[string]any{
		"name":     "body",
		"in":       "body",
		"required": true,
		"schema": map[string]any{
			"$ref": "#/definitions/" + defName,
		},
	}
}

func jsonArrayResponse(defName string) map[string]any {
	return map[string]any{
		"200": map[string]any{
			"description": "OK",
			"schema": map[string]any{
				"type": "array",
				"items": map[string]any{
					"$ref": "#/definitions/" + defName,
				},
			},
		},
	}
}

func jsonObjectResponse(defName string) map[string]any {
	return map[string]any{
		"201": map[string]any{
			"description": "Created",
			"schema": map[string]any{
				"$ref": "#/definitions/" + defName,
			},
		},
	}
}

func affectedResponse() map[string]any {
	return map[string]any{
		"200": map[string]any{
			"description": "OK",
			"schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"affected": map[string]any{"type": "integer"},
				},
			},
		},
	}
}
