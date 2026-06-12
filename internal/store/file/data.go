package file

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type Snapshot struct {
	Credentials []CredentialRecord `yaml:"credentials"`
	Servers     []ServerRecord     `yaml:"servers"`
	Tables      []TableRecord      `yaml:"tables"`
	Columns     []ColumnRecord     `yaml:"columns"`
	Functions   []FunctionRecord   `yaml:"functions"`
}

type snapshot = Snapshot

type CredentialRecord struct {
	ID        uuid.UUID `yaml:"id"`
	Name      string    `yaml:"name"`
	Payload   string    `yaml:"payload"`
	CreatedAt time.Time `yaml:"createdAt"`
}

type ServerRecord struct {
	ID            uuid.UUID  `yaml:"id"`
	Name          string     `yaml:"name"`
	Protocol      string     `yaml:"protocol"`
	Options       yaml.Node  `yaml:"options"`
	CredentialRef *uuid.UUID `yaml:"credentialRef,omitempty"`
	Enabled       bool       `yaml:"enabled"`
	UpdatedAt     time.Time  `yaml:"updatedAt"`
}

type TableRecord struct {
	ID         uuid.UUID `yaml:"id"`
	ServerID   uuid.UUID `yaml:"serverId"`
	SchemaName string    `yaml:"schemaName"`
	TableName  string    `yaml:"tableName"`
	RemoteName string    `yaml:"remoteName,omitempty"`
	KeyColumns []string  `yaml:"keyColumns"`
	Options    yaml.Node `yaml:"options"`
}

type ColumnRecord struct {
	ID         uuid.UUID `yaml:"id"`
	TableID    uuid.UUID `yaml:"tableId"`
	Name       string    `yaml:"name"`
	DataType   string    `yaml:"dataType"`
	RemoteName string    `yaml:"remoteName,omitempty"`
	Nullable   bool      `yaml:"nullable"`
	Position   int       `yaml:"position"`
}

type FunctionRecord struct {
	ID         uuid.UUID `yaml:"id"`
	ServerID   uuid.UUID `yaml:"serverId"`
	SchemaName string    `yaml:"schemaName"`
	Name       string    `yaml:"name"`
	Operation  string    `yaml:"operation"`
	RemotePath string    `yaml:"remotePath,omitempty"`
	Method     string    `yaml:"method,omitempty"`
	Options    yaml.Node `yaml:"options"`
}

type credentialRecord = CredentialRecord
type serverRecord = ServerRecord
type tableRecord = TableRecord
type columnRecord = ColumnRecord
type functionRecord = FunctionRecord

func DecodePayload(s string) ([]byte, error) {
	return decodePayload(s)
}

func NodeToRaw(node yaml.Node) json.RawMessage {
	return nodeToRaw(node)
}

func encodePayload(payload []byte) string {
	return base64.StdEncoding.EncodeToString(payload)
}

func decodePayload(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func nodeToRaw(node yaml.Node) json.RawMessage {
	if node.Kind == 0 {
		return json.RawMessage(`{}`)
	}
	var v any
	if err := node.Decode(&v); err != nil {
		return json.RawMessage(`{}`)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return b
}

func rawToNode(raw json.RawMessage) yaml.Node {
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		v = map[string]any{}
	}
	var node yaml.Node
	_ = node.Encode(v)
	return node
}
