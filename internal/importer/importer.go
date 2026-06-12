package importer

import (
	"fmt"
	"strings"
)

type Kind string

const (
	KindOpenAPI Kind = "openapi"
	KindSQL     Kind = "sql"
	KindYAML    Kind = "yaml"
)

type ConvertOptions struct {
	Kind       Kind
	Input      []byte
	OpenAPI    OpenAPIOptions
	SQL        SQLOptions
}

// Convert transforms input into declarative drivers YAML bytes.
func Convert(opts ConvertOptions) ([]byte, error) {
	var doc DeclarativeDoc
	var err error
	switch opts.Kind {
	case KindOpenAPI:
		doc, err = OpenAPIToDeclarative(opts.Input, opts.OpenAPI)
	case KindSQL:
		doc, err = SQLToDeclarative(string(opts.Input), opts.SQL)
	case KindYAML:
		return opts.Input, nil
	default:
		return nil, fmt.Errorf("unsupported import kind %q", opts.Kind)
	}
	if err != nil {
		return nil, err
	}
	out, err := doc.MarshalYAML()
	if err != nil {
		return nil, fmt.Errorf("marshal yaml: %w", err)
	}
	return out, nil
}

func DetectKind(filename string, content []byte) Kind {
	name := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(name, ".sql"):
		return KindSQL
	case strings.HasSuffix(name, ".yaml"), strings.HasSuffix(name, ".yml"):
		if looksLikeOpenAPI(content) {
			return KindOpenAPI
		}
		return KindYAML
	case strings.HasSuffix(name, ".json"):
		if looksLikeOpenAPI(content) {
			return KindOpenAPI
		}
	}
	if looksLikeOpenAPI(content) {
		return KindOpenAPI
	}
	return KindYAML
}

func looksLikeOpenAPI(content []byte) bool {
	s := strings.ToLower(string(content))
	return strings.Contains(s, "openapi:") ||
		strings.Contains(s, `"openapi"`) ||
		(strings.Contains(s, "paths:") && strings.Contains(s, "components:"))
}
