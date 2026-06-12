package importer

import (
	"fmt"
	"strings"
)

// MergeDeclarativeDocs merges multiple declarative documents. Later entries override
// earlier ones on name conflicts (server name, schema.table+server, schema.function+server).
func MergeDeclarativeDocs(docs ...DeclarativeDoc) (DeclarativeDoc, error) {
	out := DeclarativeDoc{}
	creds := map[string]DeclCredential{}
	servers := map[string]DeclServer{}
	tables := map[string]DeclTable{}
	functions := map[string]DeclFunction{}

	for _, doc := range docs {
		for _, c := range doc.Credentials {
			name := strings.TrimSpace(c.Name)
			if name == "" {
				continue
			}
			creds[name] = c
		}
		for _, s := range doc.Servers {
			name := strings.TrimSpace(s.Name)
			if name == "" {
				return DeclarativeDoc{}, fmt.Errorf("server name is required")
			}
			servers[name] = s
		}
		for _, t := range doc.Tables {
			key := tableKey(t)
			if key == "" {
				continue
			}
			tables[key] = t
		}
		for _, f := range doc.Functions {
			key := functionKey(f)
			if key == "" {
				continue
			}
			functions[key] = f
		}
	}

	for _, c := range creds {
		out.Credentials = append(out.Credentials, c)
	}
	for _, s := range servers {
		out.Servers = append(out.Servers, s)
	}
	for _, t := range tables {
		out.Tables = append(out.Tables, t)
	}
	for _, f := range functions {
		out.Functions = append(out.Functions, f)
	}
	return out, nil
}

func tableKey(t DeclTable) string {
	server := strings.TrimSpace(t.Server)
	name := strings.TrimSpace(t.Name)
	if server == "" || name == "" {
		return ""
	}
	schema := strings.TrimSpace(t.Schema)
	if schema == "" {
		schema = "public"
	}
	return schema + "\x00" + name + "\x00" + server
}

func functionKey(f DeclFunction) string {
	server := strings.TrimSpace(f.Server)
	name := strings.TrimSpace(f.Name)
	if server == "" || name == "" {
		return ""
	}
	schema := strings.TrimSpace(f.Schema)
	if schema == "" {
		schema = "public"
	}
	return schema + "\x00" + name + "\x00" + server
}

// ParseDeclarativeYAML parses declarative drivers YAML.
func ParseDeclarativeYAML(raw []byte) (DeclarativeDoc, error) {
	var doc DeclarativeDoc
	if err := yamlUnmarshal(raw, &doc); err != nil {
		return DeclarativeDoc{}, err
	}
	return doc, nil
}

func yamlUnmarshal(raw []byte, dst any) error {
	// local wrapper to keep yaml import in types.go only
	return yamlDecode(raw, dst)
}
