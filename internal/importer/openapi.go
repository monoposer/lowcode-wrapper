package importer

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type OpenAPIOptions struct {
	ServerName string
	Endpoint   string
}

type oasDoc struct {
	OpenAPI    string              `json:"openapi" yaml:"openapi"`
	Servers    []oasServer         `json:"servers" yaml:"servers"`
	Paths      map[string]oasPathItem `json:"paths" yaml:"paths"`
	Components *oasComponents         `json:"components" yaml:"components"`
}

type oasServer struct {
	URL string `json:"url" yaml:"url"`
}

type oasPathItem struct {
	Get    *oasOperation `json:"get" yaml:"get"`
	Post   *oasOperation `json:"post" yaml:"post"`
	Put    *oasOperation `json:"put" yaml:"put"`
	Patch  *oasOperation `json:"patch" yaml:"patch"`
	Delete *oasOperation `json:"delete" yaml:"delete"`
}

type oasOperation struct {
	OperationID string `json:"operationId" yaml:"operationId"`
	Summary     string `json:"summary" yaml:"summary"`
}

type oasComponents struct {
	SecuritySchemes map[string]oasSecurityScheme `json:"securitySchemes" yaml:"securitySchemes"`
}

type oasSecurityScheme struct {
	Type   string `json:"type" yaml:"type"`
	In     string `json:"in" yaml:"in"`
	Name   string `json:"name" yaml:"name"`
	Scheme string `json:"scheme" yaml:"scheme"`
}

var pathParamRe = regexp.MustCompile(`\{[^}]+\}`)

// OpenAPIToDeclarative converts an OpenAPI 3.x spec into drivers YAML document.
func OpenAPIToDeclarative(raw []byte, opts OpenAPIOptions) (DeclarativeDoc, error) {
	var doc oasDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		if err2 := json.Unmarshal(raw, &doc); err2 != nil {
			return DeclarativeDoc{}, fmt.Errorf("parse openapi: %w", err)
		}
	}
	if len(doc.Paths) == 0 {
		return DeclarativeDoc{}, fmt.Errorf("openapi: no paths found")
	}

	serverName := strings.TrimSpace(opts.ServerName)
	if serverName == "" {
		serverName = "openapi_api"
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" && len(doc.Servers) > 0 {
		endpoint = strings.TrimSpace(doc.Servers[0].URL)
	}
	if endpoint == "" {
		endpoint = "https://api.example.com"
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	enabled := true
	httpServer := DeclServer{
		Name:     serverName,
		Protocol: "http",
		Enabled:  &enabled,
		Options: map[string]any{
			"endpoint": endpoint,
		},
	}
	if auth := inferHTTPAuth(doc.Components); auth != nil {
		httpServer.Options["auth"] = auth
	}
	httpServer.Credential = map[string]any{
		"token": "${API_TOKEN}",
	}

	out := DeclarativeDoc{
		Servers: []DeclServer{httpServer},
	}

	seenTables := map[string]struct{}{}
	seenFuncs := map[string]struct{}{}

	for path, item := range doc.Paths {
		baseName := tableNameFromPath(path)
		if baseName == "" {
			continue
		}
		if item.Get != nil {
			if _, ok := seenTables[baseName]; !ok {
				seenTables[baseName] = struct{}{}
				out.Tables = append(out.Tables, DeclTable{
					Server:     serverName,
					Schema:     "public",
					Name:       baseName,
					RemoteName: sanitizeRemotePath(path),
					KeyColumns: []string{"id"},
					Columns: []DeclColumn{
						{Name: "id", DataType: "text"},
					},
				})
			}
		}
		for _, entry := range []struct {
			method string
			op     *oasOperation
			opName string
		}{
			{"POST", item.Post, "create_" + baseName},
			{"PUT", item.Put, "update_" + baseName},
			{"PATCH", item.Patch, "patch_" + baseName},
			{"DELETE", item.Delete, "delete_" + baseName},
		} {
			if entry.op == nil {
				continue
			}
			fnName := strings.TrimSpace(entry.op.OperationID)
			if fnName == "" {
				fnName = entry.opName
			}
			fnName = sanitizeIdentifier(fnName)
			if fnName == "" {
				fnName = strings.ToLower(entry.method) + "_" + baseName
			}
			if _, ok := seenFuncs[fnName]; ok {
				fnName = fnName + "_" + strings.ToLower(entry.method)
			}
			seenFuncs[fnName] = struct{}{}
			out.Functions = append(out.Functions, DeclFunction{
				Server:     serverName,
				Schema:     "public",
				Name:       fnName,
				Operation:  "invoke",
				RemotePath: sanitizeRemotePath(path),
				Method:     entry.method,
			})
		}
	}
	if len(out.Tables) == 0 && len(out.Functions) == 0 {
		return DeclarativeDoc{}, fmt.Errorf("openapi: no importable paths")
	}
	return out, nil
}

func inferHTTPAuth(comp *oasComponents) map[string]any {
	if comp == nil || len(comp.SecuritySchemes) == 0 {
		return nil
	}
	for _, scheme := range comp.SecuritySchemes {
		switch strings.ToLower(scheme.Type) {
		case "http":
			if strings.EqualFold(scheme.Scheme, "bearer") {
				return map[string]any{
					"type": "BEARER_TOKEN",
					"options": map[string]any{
						"label": "Authorization",
					},
				}
			}
		case "apikey":
			label := strings.TrimSpace(scheme.Name)
			if label == "" {
				label = "X-Api-Key"
			}
			return map[string]any{
				"type": "API_KEY",
				"options": map[string]any{
					"label": label,
				},
			}
		}
	}
	return nil
}

func tableNameFromPath(path string) string {
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	for i, p := range parts {
		parts[i] = pathParamRe.ReplaceAllString(p, "")
	}
	var name string
	for _, p := range parts {
		p = strings.Trim(p, "/")
		if p == "" {
			continue
		}
		name = p
	}
	return sanitizeIdentifier(name)
}

func sanitizeRemotePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func sanitizeIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_':
			b.WriteRune('_')
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	return strings.ToLower(out)
}
