package postgres

import (
	"context"
	"fmt"

	"github.com/monoposer/dataspan/internal/importer"
	"github.com/monoposer/dataspan/internal/store/file"
)

// ImportDeclarative imports declarative metadata into the Meta DB.
func (s *Store) ImportDeclarative(ctx context.Context, doc importer.DeclarativeDoc, mode importer.ImportMode) (importer.Result, error) {
	if mode == "" {
		mode = importer.ModeReplace
	}
	merged := doc
	if mode == importer.ModeMerge {
		current, err := s.exportDeclarative(ctx)
		if err != nil {
			return importer.Result{}, err
		}
		merged, err = importer.MergeDeclarativeDocs(current, doc)
		if err != nil {
			return importer.Result{}, err
		}
	} else if mode != importer.ModeReplace {
		return importer.Result{}, fmt.Errorf("unsupported import mode %q", mode)
	}

	snap, err := file.CompileImporterDoc(s.vault, merged)
	if err != nil {
		return importer.Result{}, err
	}
	if err := s.replaceSnapshot(ctx, snap); err != nil {
		return importer.Result{}, err
	}
	return importer.CountResult(merged), nil
}

func (s *Store) replaceSnapshot(ctx context.Context, snap file.Snapshot) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, q := range []string{
		`DELETE FROM wrapper_function`,
		`DELETE FROM wrapper_column`,
		`DELETE FROM wrapper_table`,
		`DELETE FROM wrapper_server`,
		`DELETE FROM wrapper_credential`,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			return err
		}
	}

	for _, c := range snap.Credentials {
		payload, err := file.DecodePayload(c.Payload)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO wrapper_credential (id, name, payload, created_at)
			VALUES ($1, $2, $3, $4)
		`, c.ID, c.Name, payload, c.CreatedAt); err != nil {
			return err
		}
	}
	for _, srv := range snap.Servers {
		opts := file.NodeToRaw(srv.Options)
		if _, err := tx.Exec(ctx, `
			INSERT INTO wrapper_server (id, name, protocol, options, credential_ref, enabled, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, srv.ID, srv.Name, srv.Protocol, opts, srv.CredentialRef, srv.Enabled, srv.UpdatedAt); err != nil {
			return err
		}
	}
	for _, tbl := range snap.Tables {
		opts := file.NodeToRaw(tbl.Options)
		if _, err := tx.Exec(ctx, `
			INSERT INTO wrapper_table (id, server_id, schema_name, table_name, remote_name, key_columns, options)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, tbl.ID, tbl.ServerID, tbl.SchemaName, tbl.TableName, tbl.RemoteName, tbl.KeyColumns, opts); err != nil {
			return err
		}
	}
	for _, col := range snap.Columns {
		if _, err := tx.Exec(ctx, `
			INSERT INTO wrapper_column (id, table_id, name, data_type, remote_name, nullable, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, col.ID, col.TableID, col.Name, col.DataType, col.RemoteName, col.Nullable, col.Position); err != nil {
			return err
		}
	}
	for _, fn := range snap.Functions {
		opts := file.NodeToRaw(fn.Options)
		if _, err := tx.Exec(ctx, `
			INSERT INTO wrapper_function (id, server_id, schema_name, name, operation, remote_path, method, options)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, fn.ID, fn.ServerID, fn.SchemaName, fn.Name, fn.Operation, fn.RemotePath, fn.Method, opts); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
