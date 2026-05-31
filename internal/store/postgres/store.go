package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/models"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	pool  *pgxpool.Pool
	vault *auth.Vault
}

func NewFromEnv(vault *auth.Vault) (*Store, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}
	s := &Store{pool: pool, vault: vault}
	if err := s.Migrate(context.Background()); err != nil {
		pool.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Vault() *auth.Vault {
	return s.vault
}

func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// -------- Credentials --------

func (s *Store) CreateCredential(ctx context.Context, name string, data map[string]any) (*models.Credential, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	payload, err := s.vault.Encrypt(data)
	if err != nil {
		return nil, err
	}
	var c models.Credential
	err = s.pool.QueryRow(ctx, `
		INSERT INTO wrapper_credential (name, payload)
		VALUES ($1, $2)
		RETURNING id, name, created_at
	`, name, payload).Scan(&c.ID, &c.Name, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM wrapper_credential WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) ResolveCredential(ctx context.Context, ref uuid.UUID) (map[string]any, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `SELECT payload FROM wrapper_credential WHERE id = $1`, ref).Scan(&payload)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.vault.Decrypt(payload)
}

// -------- Servers --------

func (s *Store) CreateServer(ctx context.Context, req models.CreateServerRequest) (*models.Server, error) {
	if req.Name == "" || req.Protocol == "" {
		return nil, fmt.Errorf("name and protocol are required")
	}
	opts := req.Options
	if len(opts) == 0 {
		opts = json.RawMessage(`{}`)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	var srv models.Server
	err := s.pool.QueryRow(ctx, `
		INSERT INTO wrapper_server (name, protocol, options, credential_ref, enabled, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, protocol, options, credential_ref, enabled, updated_at
	`, req.Name, req.Protocol, opts, req.CredentialRef, enabled, time.Now()).Scan(
		&srv.ID, &srv.Name, &srv.Protocol, &srv.Options, &srv.CredentialRef, &srv.Enabled, &srv.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &srv, nil
}

func (s *Store) ListServers(ctx context.Context) ([]models.Server, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, protocol, options, credential_ref, enabled, updated_at
		FROM wrapper_server ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Server
	for rows.Next() {
		var srv models.Server
		if err := rows.Scan(&srv.ID, &srv.Name, &srv.Protocol, &srv.Options, &srv.CredentialRef, &srv.Enabled, &srv.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, srv)
	}
	return out, rows.Err()
}

func (s *Store) GetServerByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	var srv models.Server
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, protocol, options, credential_ref, enabled, updated_at
		FROM wrapper_server WHERE id = $1
	`, id).Scan(&srv.ID, &srv.Name, &srv.Protocol, &srv.Options, &srv.CredentialRef, &srv.Enabled, &srv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &srv, nil
}

func (s *Store) GetServerByName(ctx context.Context, name string) (*models.Server, error) {
	var srv models.Server
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, protocol, options, credential_ref, enabled, updated_at
		FROM wrapper_server WHERE name = $1
	`, name).Scan(&srv.ID, &srv.Name, &srv.Protocol, &srv.Options, &srv.CredentialRef, &srv.Enabled, &srv.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &srv, nil
}

func (s *Store) UpdateServer(ctx context.Context, id uuid.UUID, req models.UpdateServerRequest) (*models.Server, error) {
	srv, err := s.GetServerByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(req.Options) > 0 {
		srv.Options = req.Options
	}
	if req.CredentialRef != nil {
		srv.CredentialRef = req.CredentialRef
	}
	if req.Enabled != nil {
		srv.Enabled = *req.Enabled
	}
	srv.UpdatedAt = time.Now()
	_, err = s.pool.Exec(ctx, `
		UPDATE wrapper_server SET options = $2, credential_ref = $3, enabled = $4, updated_at = $5
		WHERE id = $1
	`, id, srv.Options, srv.CredentialRef, srv.Enabled, srv.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Store) DeleteServer(ctx context.Context, id uuid.UUID) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM wrapper_server WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// -------- Tables --------

func (s *Store) CreateTable(ctx context.Context, req models.CreateTableRequest) (*models.Table, []models.Column, error) {
	if req.ServerName == "" || req.TableName == "" {
		return nil, nil, fmt.Errorf("serverName and tableName are required")
	}
	srv, err := s.GetServerByName(ctx, req.ServerName)
	if err != nil {
		return nil, nil, err
	}
	schema := req.SchemaName
	if schema == "" {
		schema = "public"
	}
	opts := req.Options
	if len(opts) == 0 {
		opts = json.RawMessage(`{}`)
	}
	keyCols := req.KeyColumns
	if keyCols == nil {
		keyCols = []string{}
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback(ctx)

	var tbl models.Table
	err = tx.QueryRow(ctx, `
		INSERT INTO wrapper_table (server_id, schema_name, table_name, remote_name, key_columns, options)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, server_id, schema_name, table_name, remote_name, key_columns, options
	`, srv.ID, schema, req.TableName, req.RemoteName, keyCols, opts).Scan(
		&tbl.ID, &tbl.ServerID, &tbl.SchemaName, &tbl.TableName, &tbl.RemoteName, &tbl.KeyColumns, &tbl.Options,
	)
	if err != nil {
		return nil, nil, err
	}
	tbl.ServerName = srv.Name

	var cols []models.Column
	for i, c := range req.Columns {
		if c.Name == "" {
			continue
		}
		dt := c.DataType
		if dt == "" {
			dt = "text"
		}
		nullable := true
		if c.Nullable != nil {
			nullable = *c.Nullable
		}
		pos := c.Position
		if pos == 0 {
			pos = i
		}
		var col models.Column
		err = tx.QueryRow(ctx, `
			INSERT INTO wrapper_column (table_id, name, data_type, remote_name, nullable, position)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, table_id, name, data_type, remote_name, nullable, position
		`, tbl.ID, c.Name, dt, c.RemoteName, nullable, pos).Scan(
			&col.ID, &col.TableID, &col.Name, &col.DataType, &col.RemoteName, &col.Nullable, &col.Position,
		)
		if err != nil {
			return nil, nil, err
		}
		cols = append(cols, col)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, err
	}
	return &tbl, cols, nil
}

func (s *Store) ListTables(ctx context.Context) ([]models.Table, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.server_id, t.schema_name, t.table_name, t.remote_name, t.key_columns, t.options, s.name
		FROM wrapper_table t
		JOIN wrapper_server s ON s.id = t.server_id
		ORDER BY t.schema_name, t.table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Table
	for rows.Next() {
		var t models.Table
		if err := rows.Scan(&t.ID, &t.ServerID, &t.SchemaName, &t.TableName, &t.RemoteName, &t.KeyColumns, &t.Options, &t.ServerName); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ListColumns(ctx context.Context, schema, table string) ([]models.Column, error) {
	var tableID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		SELECT id FROM wrapper_table WHERE schema_name = $1 AND table_name = $2
	`, schema, table).Scan(&tableID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, table_id, name, data_type, remote_name, nullable, position
		FROM wrapper_column WHERE table_id = $1 ORDER BY position, name
	`, tableID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Column
	for rows.Next() {
		var c models.Column
		if err := rows.Scan(&c.ID, &c.TableID, &c.Name, &c.DataType, &c.RemoteName, &c.Nullable, &c.Position); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) ResolveTable(ctx context.Context, schema, table string) (*models.ResolvedTable, error) {
	var rt models.ResolvedTable
	err := s.pool.QueryRow(ctx, `
		SELECT t.id, t.server_id, t.schema_name, t.table_name, t.remote_name, t.key_columns, t.options,
		       s.id, s.name, s.protocol, s.options, s.credential_ref, s.enabled, s.updated_at
		FROM wrapper_table t
		JOIN wrapper_server s ON s.id = t.server_id
		WHERE t.schema_name = $1 AND t.table_name = $2
	`, schema, table).Scan(
		&rt.Table.ID, &rt.Table.ServerID, &rt.Table.SchemaName, &rt.Table.TableName,
		&rt.Table.RemoteName, &rt.Table.KeyColumns, &rt.Table.Options,
		&rt.Server.ID, &rt.Server.Name, &rt.Server.Protocol, &rt.Server.Options,
		&rt.Server.CredentialRef, &rt.Server.Enabled, &rt.Server.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rt.Table.ServerName = rt.Server.Name
	if !rt.Server.Enabled {
		return nil, fmt.Errorf("server %q is disabled", rt.Server.Name)
	}

	cols, err := s.ListColumns(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	rt.Columns = cols
	return &rt, nil
}

// -------- Functions --------

func (s *Store) CreateFunction(ctx context.Context, req models.CreateFunctionRequest) (*models.Function, error) {
	if req.ServerName == "" || req.Name == "" || req.Operation == "" {
		return nil, fmt.Errorf("serverName, name and operation are required")
	}
	srv, err := s.GetServerByName(ctx, req.ServerName)
	if err != nil {
		return nil, err
	}
	schema := req.SchemaName
	if schema == "" {
		schema = "public"
	}
	opts := req.Options
	if len(opts) == 0 {
		opts = json.RawMessage(`{}`)
	}
	var fn models.Function
	err = s.pool.QueryRow(ctx, `
		INSERT INTO wrapper_function (server_id, schema_name, name, operation, remote_path, method, options)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, server_id, schema_name, name, operation, remote_path, method, options
	`, srv.ID, schema, req.Name, req.Operation, req.RemotePath, req.Method, opts).Scan(
		&fn.ID, &fn.ServerID, &fn.SchemaName, &fn.Name, &fn.Operation, &fn.RemotePath, &fn.Method, &fn.Options,
	)
	if err != nil {
		return nil, err
	}
	fn.ServerName = srv.Name
	return &fn, nil
}

func (s *Store) ListFunctions(ctx context.Context) ([]models.Function, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT f.id, f.server_id, f.schema_name, f.name, f.operation, f.remote_path, f.method, f.options, s.name
		FROM wrapper_function f
		JOIN wrapper_server s ON s.id = f.server_id
		ORDER BY f.schema_name, f.name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Function
	for rows.Next() {
		var f models.Function
		if err := rows.Scan(&f.ID, &f.ServerID, &f.SchemaName, &f.Name, &f.Operation, &f.RemotePath, &f.Method, &f.Options, &f.ServerName); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (s *Store) ResolveFunction(ctx context.Context, schema, name string) (*models.ResolvedFunction, error) {
	var rf models.ResolvedFunction
	err := s.pool.QueryRow(ctx, `
		SELECT f.id, f.server_id, f.schema_name, f.name, f.operation, f.remote_path, f.method, f.options,
		       s.id, s.name, s.protocol, s.options, s.credential_ref, s.enabled, s.updated_at
		FROM wrapper_function f
		JOIN wrapper_server s ON s.id = f.server_id
		WHERE f.schema_name = $1 AND f.name = $2
	`, schema, name).Scan(
		&rf.Function.ID, &rf.Function.ServerID, &rf.Function.SchemaName, &rf.Function.Name,
		&rf.Function.Operation, &rf.Function.RemotePath, &rf.Function.Method, &rf.Function.Options,
		&rf.Server.ID, &rf.Server.Name, &rf.Server.Protocol, &rf.Server.Options,
		&rf.Server.CredentialRef, &rf.Server.Enabled, &rf.Server.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	rf.Function.ServerName = rf.Server.Name
	if !rf.Server.Enabled {
		return nil, fmt.Errorf("server %q is disabled", rf.Server.Name)
	}
	return &rf, nil
}

// ServerCredential resolves credential data for a server (nil if no ref).
func (s *Store) ServerCredential(ctx context.Context, srv *models.Server) (map[string]any, error) {
	if srv == nil || srv.CredentialRef == nil {
		return nil, nil
	}
	return s.ResolveCredential(ctx, *srv.CredentialRef)
}
