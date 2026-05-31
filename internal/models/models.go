package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Protocol string

const (
	ProtocolHTTP     Protocol = "http"
	ProtocolPostgres Protocol = "postgres"
	ProtocolMySQL    Protocol = "mysql"
	ProtocolFile     Protocol = "file"
)

type AuthType string

const (
	AuthNone              AuthType = "NONE"
	AuthBasic             AuthType = "BASIC"
	AuthAPIKey            AuthType = "API_KEY"
	AuthBearerToken       AuthType = "BEARER_TOKEN"
	AuthClientCredentials AuthType = "CLIENT_CREDENTIALS"
	AuthUniversal         AuthType = "UNIVERSAL"
)

type HTTPAuth struct {
	Type    AuthType       `json:"type"`
	Options map[string]any `json:"options,omitempty"`
}

type Credential struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

type Server struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	Protocol      Protocol        `json:"protocol"`
	Options       json.RawMessage `json:"options,omitempty"`
	CredentialRef *uuid.UUID      `json:"credentialRef,omitempty"`
	Enabled       bool            `json:"enabled"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type Table struct {
	ID          uuid.UUID       `json:"id"`
	ServerID    uuid.UUID       `json:"serverId"`
	SchemaName  string          `json:"schemaName"`
	TableName   string          `json:"tableName"`
	RemoteName  string          `json:"remoteName,omitempty"`
	KeyColumns  []string        `json:"keyColumns"`
	Options     json.RawMessage `json:"options,omitempty"`
	ServerName  string          `json:"serverName,omitempty"`
}

type Column struct {
	ID         uuid.UUID `json:"id"`
	TableID    uuid.UUID `json:"tableId"`
	Name       string    `json:"name"`
	DataType   string    `json:"dataType"`
	RemoteName string    `json:"remoteName,omitempty"`
	Nullable   bool      `json:"nullable"`
	Position   int       `json:"position"`
}

type Function struct {
	ID          uuid.UUID       `json:"id"`
	ServerID    uuid.UUID       `json:"serverId"`
	SchemaName  string          `json:"schemaName"`
	Name        string          `json:"name"`
	Operation   string          `json:"operation"`
	RemotePath  string          `json:"remotePath,omitempty"`
	Method      string          `json:"method,omitempty"`
	Options     json.RawMessage `json:"options,omitempty"`
	ServerName  string          `json:"serverName,omitempty"`
}

type ResolvedTable struct {
	Table   Table
	Server  Server
	Columns []Column
}

type ResolvedFunction struct {
	Function Function
	Server   Server
}

// Admin request bodies

type CreateCredentialRequest struct {
	Name string         `json:"name"`
	Data map[string]any `json:"data"`
}

type CreateServerRequest struct {
	Name          string          `json:"name"`
	Protocol      Protocol        `json:"protocol"`
	Options       json.RawMessage `json:"options,omitempty"`
	CredentialRef *uuid.UUID      `json:"credentialRef,omitempty"`
	Enabled       *bool           `json:"enabled,omitempty"`
}

type UpdateServerRequest struct {
	Options       json.RawMessage `json:"options,omitempty"`
	CredentialRef *uuid.UUID      `json:"credentialRef,omitempty"`
	Enabled       *bool           `json:"enabled,omitempty"`
}

type ColumnInput struct {
	Name       string `json:"name"`
	DataType   string `json:"dataType,omitempty"`
	RemoteName string `json:"remoteName,omitempty"`
	Nullable   *bool  `json:"nullable,omitempty"`
	Position   int    `json:"position,omitempty"`
}

type CreateTableRequest struct {
	ServerName string        `json:"serverName"`
	SchemaName string        `json:"schemaName,omitempty"`
	TableName  string        `json:"tableName"`
	RemoteName string        `json:"remoteName,omitempty"`
	KeyColumns []string      `json:"keyColumns,omitempty"`
	Options    json.RawMessage `json:"options,omitempty"`
	Columns    []ColumnInput `json:"columns"`
}

type CreateFunctionRequest struct {
	ServerName string          `json:"serverName"`
	SchemaName string          `json:"schemaName,omitempty"`
	Name       string          `json:"name"`
	Operation  string          `json:"operation"`
	RemotePath string          `json:"remotePath,omitempty"`
	Method     string          `json:"method,omitempty"`
	Options    json.RawMessage `json:"options,omitempty"`
}

// ServerOptions helpers

type HTTPServerOptions struct {
	Endpoint string            `json:"endpoint"`
	BasePath string            `json:"basePath,omitempty"`
	Auth     *HTTPAuth         `json:"auth,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// HTTPTableOptions applies to wrapper_table.options when server protocol is http.
type HTTPTableOptions struct {
	Headers map[string]string `json:"headers,omitempty"`
}

// HTTPFunctionOptions applies to wrapper_function.options for http invoke.
type HTTPFunctionOptions struct {
	Headers map[string]string `json:"headers,omitempty"`
}

type PostgresServerOptions struct {
	DSN    string `json:"dsn,omitempty"`
	Schema string `json:"schema,omitempty"`
}

type MySQLServerOptions struct {
	DSN      string `json:"dsn,omitempty"`
	Database string `json:"database,omitempty"`
}

type FileServerOptions struct {
	RootPath string `json:"rootPath"`
}

type FileTableOptions struct {
	Format string `json:"format"` // csv, json, ndjson
}

func ParseServerOptions[T any](raw json.RawMessage) (T, error) {
	var out T
	if len(raw) == 0 || string(raw) == "null" {
		return out, nil
	}
	err := json.Unmarshal(raw, &out)
	return out, err
}

func RemoteColumnName(col Column) string {
	if col.RemoteName != "" {
		return col.RemoteName
	}
	return col.Name
}

func RemoteTableName(t Table) string {
	if t.RemoteName != "" {
		return t.RemoteName
	}
	return t.TableName
}
