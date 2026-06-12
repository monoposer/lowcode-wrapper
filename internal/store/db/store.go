package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/errs"
)

type Store struct {
	db    *gorm.DB
	vault *auth.Vault
}

func New(vault *auth.Vault, cfg Config) (*Store, error) {
	gdb, err := Open(cfg.DSN)
	if err != nil {
		return nil, err
	}
	return &Store{db: gdb, vault: vault}, nil
}

func Ping(ctx context.Context, cfg Config) error {
	gdb, err := Open(cfg.DSN)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.PingContext(ctx)
}

func (s *Store) Close() {
	if s == nil || s.db == nil {
		return
	}
	if sqlDB, err := s.db.DB(); err == nil {
		sqlDB.Close()
	}
}

func (s *Store) Vault() *auth.Vault {
	return s.vault
}

func (s *Store) DB() *gorm.DB {
	return s.db
}

func notFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
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
	row := models.MetaCredential{
		ID:        newUUID(),
		Name:      name,
		Payload:   payload,
		CreatedAt: time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	out := toCredential(row)
	return &out, nil
}

func (s *Store) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Delete(&models.MetaCredential{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (s *Store) ResolveCredential(ctx context.Context, ref uuid.UUID) (map[string]any, error) {
	var row models.MetaCredential
	if err := s.db.WithContext(ctx).Select("payload").First(&row, "id = ?", ref).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	return s.vault.Decrypt(row.Payload)
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
	row := models.MetaServer{
		ID:            newUUID(),
		Name:          req.Name,
		Protocol:      req.Protocol,
		Options:       toJSONRaw(opts),
		CredentialRef: req.CredentialRef,
		Enabled:       enabled,
		UpdatedAt:     time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	out := toServer(row)
	return &out, nil
}

func (s *Store) ListServers(ctx context.Context) ([]models.Server, error) {
	var rows []models.MetaServer
	if err := s.db.WithContext(ctx).Order("name").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Server, len(rows))
	for i, row := range rows {
		out[i] = toServer(row)
	}
	return out, nil
}

func (s *Store) GetServerByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	var row models.MetaServer
	if err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	out := toServer(row)
	return &out, nil
}

func (s *Store) GetServerByName(ctx context.Context, name string) (*models.Server, error) {
	var row models.MetaServer
	if err := s.db.WithContext(ctx).First(&row, "name = ?", name).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	out := toServer(row)
	return &out, nil
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
	updates := map[string]any{
		"options":        toJSONRaw(srv.Options),
		"credential_ref": srv.CredentialRef,
		"enabled":        srv.Enabled,
		"updated_at":     srv.UpdatedAt,
	}
	if err := s.db.WithContext(ctx).Model(&models.MetaServer{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Store) DeleteServer(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Delete(&models.MetaServer{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.ErrNotFound
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

	var tbl models.Table
	var cols []models.Column
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		row := models.MetaForeignTable{
			ID:         newUUID(),
			ServerID:   srv.ID,
			SchemaName: schema,
			Name:       req.TableName,
			RemoteName: req.RemoteName,
			KeyColumns: encodeKeyColumns(keyCols),
			Options:    toJSONRaw(opts),
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		tbl = toTable(row, srv.Name)

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
			colRow := models.MetaForeignColumn{
				ID:         newUUID(),
				TableID:    row.ID,
				Name:       c.Name,
				DataType:   dt,
				RemoteName: c.RemoteName,
				Nullable:   nullable,
				Position:   pos,
			}
			if err := tx.Create(&colRow).Error; err != nil {
				return err
			}
			cols = append(cols, toColumn(colRow))
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return &tbl, cols, nil
}

func (s *Store) ListTables(ctx context.Context) ([]models.Table, error) {
	var rows []models.MetaForeignTable
	if err := s.db.WithContext(ctx).
		Preload("Server").
		Order("schema_name, table_name").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Table, len(rows))
	for i, row := range rows {
		out[i] = toTable(row, row.Server.Name)
	}
	return out, nil
}

func (s *Store) ListColumns(ctx context.Context, schema, table string) ([]models.Column, error) {
	var metaTable models.MetaForeignTable
	if err := s.db.WithContext(ctx).
		Select("id").
		Where("schema_name = ? AND table_name = ?", schema, table).
		First(&metaTable).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	var rows []models.MetaForeignColumn
	if err := s.db.WithContext(ctx).
		Where("table_id = ?", metaTable.ID).
		Order("position, name").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Column, len(rows))
	for i, row := range rows {
		out[i] = toColumn(row)
	}
	return out, nil
}

func (s *Store) ResolveTable(ctx context.Context, schema, table string) (*models.ResolvedTable, error) {
	var row models.MetaForeignTable
	if err := s.db.WithContext(ctx).
		Preload("Server").
		Where("schema_name = ? AND table_name = ?", schema, table).
		First(&row).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	srv := toServer(row.Server)
	if !srv.Enabled {
		return nil, fmt.Errorf("server %q is disabled", srv.Name)
	}
	cols, err := s.ListColumns(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	return &models.ResolvedTable{
		Table:   toTable(row, srv.Name),
		Server:  srv,
		Columns: cols,
	}, nil
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
	row := models.MetaForeignFunction{
		ID:         newUUID(),
		ServerID:   srv.ID,
		SchemaName: schema,
		Name:       req.Name,
		Operation:  req.Operation,
		RemotePath: req.RemotePath,
		Method:     req.Method,
		Options:    toJSONRaw(opts),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	out := toFunction(row, srv.Name)
	return &out, nil
}

func (s *Store) ListFunctions(ctx context.Context) ([]models.Function, error) {
	var rows []models.MetaForeignFunction
	if err := s.db.WithContext(ctx).
		Preload("Server").
		Order("schema_name, name").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Function, len(rows))
	for i, row := range rows {
		out[i] = toFunction(row, row.Server.Name)
	}
	return out, nil
}

func (s *Store) ResolveFunction(ctx context.Context, schema, name string) (*models.ResolvedFunction, error) {
	var row models.MetaForeignFunction
	if err := s.db.WithContext(ctx).
		Preload("Server").
		Where("schema_name = ? AND name = ?", schema, name).
		First(&row).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	srv := toServer(row.Server)
	if !srv.Enabled {
		return nil, fmt.Errorf("server %q is disabled", srv.Name)
	}
	return &models.ResolvedFunction{
		Function: toFunction(row, srv.Name),
		Server:   srv,
	}, nil
}

func (s *Store) ServerCredential(ctx context.Context, srv *models.Server) (map[string]any, error) {
	if srv == nil || srv.CredentialRef == nil {
		return nil, nil
	}
	return s.ResolveCredential(ctx, *srv.CredentialRef)
}
