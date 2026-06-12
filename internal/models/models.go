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
	ProtocolSQLite   Protocol = "sqlite"
	ProtocolFile     Protocol = "file"
	ProtocolMongo    Protocol = "mongo"
	ProtocolS3       Protocol = "s3"
	ProtocolFirebase Protocol = "firebase"
	ProtocolNotion   Protocol = "notion"
	ProtocolRedis    Protocol = "redis"
	ProtocolAirtable Protocol = "airtable"
	ProtocolSheets   Protocol = "sheets"
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
	Format string `json:"format"` // csv, json, ndjson, yaml, xlsx
}

// NotionServerOptions — REST API; driver delegates to http with Notion defaults.
type NotionServerOptions struct {
	Endpoint      string            `json:"endpoint,omitempty"`
	NotionVersion string            `json:"notionVersion,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// FirebaseServerOptions — Firestore REST; delegates to http when endpoint is set or built from projectId.
type FirebaseServerOptions struct {
	ProjectID string `json:"projectId"`
	Database  string `json:"database,omitempty"` // default (default)
	Endpoint  string `json:"endpoint,omitempty"`
}

// AirtableServerOptions — Airtable REST; delegates to http with Airtable defaults.
type AirtableServerOptions struct {
	Endpoint string            `json:"endpoint,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// SheetsServerOptions — Google Sheets REST; delegates to http with Sheets API defaults.
type SheetsServerOptions struct {
	Endpoint string            `json:"endpoint,omitempty"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// MongoServerOptions — native MongoDB driver.
type MongoServerOptions struct {
	URI      string `json:"uri,omitempty"`
	Database string `json:"database,omitempty"`
}

// MongoTableOptions — collection name defaults to table remote_name / table_name.
type MongoTableOptions struct {
	Collection string `json:"collection,omitempty"`
}

// RedisServerOptions — go-redis.
type RedisServerOptions struct {
	Addr     string `json:"addr,omitempty"`
	URL      string `json:"url,omitempty"`
	DB       int    `json:"db,omitempty"`
	Username string `json:"username,omitempty"`
}

// RedisTableOptions — key layout for a logical table.
type RedisTableOptions struct {
	KeyPrefix string `json:"keyPrefix,omitempty"`
	Type      string `json:"type,omitempty"` // hash (default), string, json
}

// S3ServerOptions — AWS S3; phase 1 SELECT only.
type S3ServerOptions struct {
	Region string `json:"region,omitempty"`
	Bucket string `json:"bucket"`
}

// S3TableOptions — object prefix or single object key.
type S3TableOptions struct {
	Prefix string `json:"prefix,omitempty"`
	Format string `json:"format,omitempty"` // json, csv, ndjson (single object)
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
