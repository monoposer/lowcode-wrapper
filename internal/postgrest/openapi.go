package postgrest

import (
	"context"
	"fmt"
	"strings"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store"
	"github.com/monoposer/dataspan/internal/version"
)

// OpenAPIOptions configures generated OpenAPI 3.1 output.
type OpenAPIOptions struct {
	Host     string
	BasePath string // e.g. /rest/v1
	Schema   string // optional Accept-Profile filter; empty = all schemas
}

// BuildOpenAPI returns an OpenAPI 3.1 document from store metadata.
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

	servers, err := s.ListServers(ctx)
	if err != nil {
		return nil, err
	}
	protoByServer := map[string]models.Protocol{}
	for _, srv := range servers {
		protoByServer[srv.Name] = srv.Protocol
	}

	basePath := opts.BasePath
	if basePath == "" {
		basePath = "/rest/v1"
	}
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	paths := map[string]any{}
	schemas := map[string]any{}

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
		schemas[defName] = tableSchema(tbl, cols, protoByServer[tbl.ServerName])
		paths[pathKey] = tablePathItem(tbl, defName, isReadOnlyProtocol(protoByServer[tbl.ServerName]))
	}

	for _, fn := range functions {
		pathKey := fmt.Sprintf("/rpc/%s", fn.Name)
		if pathItem, ok := paths[pathKey].(map[string]any); ok {
			pathItem["post"] = rpcOperation(fn)
			continue
		}
		paths[pathKey] = map[string]any{
			"post": rpcOperation(fn),
		}
	}

	title := "dataspan API"
	if opts.Schema != "" {
		title = fmt.Sprintf("dataspan API (%s)", opts.Schema)
	}

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       title,
			"description": "PostgREST-style foreign data wrapper sidecar (tables, columns, functions).",
			"version":     version.Version,
		},
		"servers":    openAPIServers(opts.Host, basePath),
		"paths":      paths,
		"components": map[string]any{"schemas": schemas},
	}, nil
}

func openAPIServers(host, basePath string) []map[string]any {
	if host == "" {
		return []map[string]any{
			{"url": basePath, "description": "Dataspan data API (relative to request host)"},
		}
	}
	return []map[string]any{
		{"url": fmt.Sprintf("http://%s%s", host, basePath), "description": "HTTP"},
		{"url": fmt.Sprintf("https://%s%s", host, basePath), "description": "HTTPS"},
	}
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

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func tableSchema(tbl models.Table, cols []models.Column, protocol models.Protocol) map[string]any {
	props := map[string]any{}
	required := []string{}
	for _, col := range cols {
		props[col.Name] = columnSchema(col)
		if !col.Nullable {
			required = append(required, col.Name)
		}
	}
	desc := fmt.Sprintf(
		"Foreign table %s.%s (server: %s, remote: %s, protocol: %s)",
		tbl.SchemaName, tbl.TableName, tbl.ServerName, models.RemoteTableName(tbl), protocol,
	)
	if isReadOnlyProtocol(protocol) {
		desc += " [read-only: Select only]"
	}
	def := map[string]any{
		"type":        "object",
		"properties":  props,
		"description": desc,
	}
	if isReadOnlyProtocol(protocol) {
		def["readOnly"] = true
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
	if format := openAPIFormat(col.DataType); format != "" {
		schema["format"] = format
	}
	return schema
}

func openAPIFormat(dataType string) string {
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

func tablePathItem(tbl models.Table, defName string, readOnly bool) map[string]any {
	profileParam := profileHeaderParam(tbl.SchemaName)
	item := map[string]any{
		"get": map[string]any{
			"tags":       []string{tbl.SchemaName},
			"summary":    fmt.Sprintf("Read rows from %s", tbl.TableName),
			"parameters": []any{profileParam},
			"responses":  jsonArrayResponse(defName),
		},
	}
	if readOnly {
		return item
	}
	item["post"] = map[string]any{
		"tags":        []string{tbl.SchemaName},
		"summary":     fmt.Sprintf("Insert into %s", tbl.TableName),
		"parameters":  []any{profileParam},
		"requestBody": jsonRequestBody(defName, true),
		"responses":   jsonObjectResponse(defName, "201", "Created"),
	}
	item["patch"] = map[string]any{
		"tags":        []string{tbl.SchemaName},
		"summary":     fmt.Sprintf("Update %s", tbl.TableName),
		"parameters":  []any{profileParam},
		"requestBody": jsonRequestBody(defName, true),
		"responses":   affectedResponse(),
	}
	item["delete"] = map[string]any{
		"tags":       []string{tbl.SchemaName},
		"summary":    fmt.Sprintf("Delete from %s", tbl.TableName),
		"parameters": []any{profileParam},
		"responses":  affectedResponse(),
	}
	return item
}

func isReadOnlyProtocol(p models.Protocol) bool {
	return p == models.ProtocolFile || p == models.ProtocolS3
}

func rpcOperation(fn models.Function) map[string]any {
	params := []any{
		map[string]any{
			"name":        "schema",
			"in":          "query",
			"schema":      map[string]any{"type": "string", "default": fn.SchemaName},
			"description": fmt.Sprintf("Schema profile (default %q)", fn.SchemaName),
		},
	}
	op := map[string]any{
		"tags":        []string{fn.SchemaName},
		"summary":     fmt.Sprintf("RPC %s (%s)", fn.Name, fn.Operation),
		"description": fmt.Sprintf("server: %s", fn.ServerName),
		"parameters":  params,
		"responses": map[string]any{
			"200": map[string]any{
				"description": "OK",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": map[string]any{"type": "object"},
					},
				},
			},
		},
	}
	if fn.Operation == "invoke" {
		op["requestBody"] = map[string]any{
			"required": false,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"type": "object"},
				},
			},
		}
	}
	return op
}

func profileHeaderParam(schema string) map[string]any {
	return map[string]any{
		"name":        "Accept-Profile",
		"in":          "header",
		"schema":      map[string]any{"type": "string", "default": schema},
		"description": "Schema profile",
	}
}

func jsonRequestBody(defName string, required bool) map[string]any {
	return map[string]any{
		"required": required,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schemaRef(defName),
			},
		},
	}
}

func jsonArrayResponse(defName string) map[string]any {
	return map[string]any{
		"200": map[string]any{
			"description": "OK",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"type":  "array",
						"items": schemaRef(defName),
					},
				},
			},
		},
	}
}

func jsonObjectResponse(defName, status, description string) map[string]any {
	return map[string]any{
		status: map[string]any{
			"description": description,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": schemaRef(defName),
				},
			},
		},
	}
}

func affectedResponse() map[string]any {
	return map[string]any{
		"200": map[string]any{
			"description": "OK",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"affected": map[string]any{"type": "integer"},
						},
					},
				},
			},
		},
	}
}
