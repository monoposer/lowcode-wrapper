package importer

import "gopkg.in/yaml.v3"

// DeclarativeDoc matches file-mode drivers.yaml shape.
type DeclarativeDoc struct {
	Credentials []DeclCredential `yaml:"credentials,omitempty"`
	Servers     []DeclServer     `yaml:"servers"`
	Tables      []DeclTable      `yaml:"tables"`
	Functions   []DeclFunction   `yaml:"functions"`
}

type DeclCredential struct {
	Name string         `yaml:"name"`
	Data map[string]any `yaml:"data"`
}

type DeclServer struct {
	Name       string              `yaml:"name"`
	Protocol   string              `yaml:"protocol"`
	Credential map[string]any      `yaml:"credential,omitempty"`
	Enabled    *bool               `yaml:"enabled,omitempty"`
	Options    map[string]any      `yaml:"options,omitempty"`
}

type DeclTable struct {
	Server     string         `yaml:"server"`
	Schema     string         `yaml:"schema,omitempty"`
	Name       string         `yaml:"name"`
	RemoteName string         `yaml:"remoteName,omitempty"`
	KeyColumns []string       `yaml:"keyColumns,omitempty"`
	Options    map[string]any `yaml:"options,omitempty"`
	Columns    []DeclColumn   `yaml:"columns"`
}

type DeclColumn struct {
	Name       string `yaml:"name"`
	DataType   string `yaml:"dataType,omitempty"`
	RemoteName string `yaml:"remoteName,omitempty"`
	Nullable   *bool  `yaml:"nullable,omitempty"`
	Position   int    `yaml:"position,omitempty"`
}

type DeclFunction struct {
	Server     string         `yaml:"server"`
	Schema     string         `yaml:"schema,omitempty"`
	Name       string         `yaml:"name"`
	Operation  string         `yaml:"operation"`
	RemotePath string         `yaml:"remotePath,omitempty"`
	Method     string         `yaml:"method,omitempty"`
	Options    map[string]any `yaml:"options,omitempty"`
}

func (d DeclarativeDoc) MarshalYAML() ([]byte, error) {
	return yaml.Marshal(d)
}

func yamlDecode(raw []byte, dst any) error {
	return yaml.Unmarshal(raw, dst)
}
