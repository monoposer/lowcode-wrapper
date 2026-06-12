package db

import (
	"context"
	"fmt"

	"github.com/monoposer/dataspan/internal/importer"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/file"
	"gorm.io/gorm"
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
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, model := range []any{
			&models.MetaForeignFunction{},
			&models.MetaForeignColumn{},
			&models.MetaForeignTable{},
			&models.MetaServer{},
			&models.MetaCredential{},
		} {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error; err != nil {
				return err
			}
		}

		for _, c := range snap.Credentials {
			payload, err := file.DecodePayload(c.Payload)
			if err != nil {
				return err
			}
			row := models.MetaCredential{
				ID:        c.ID,
				Name:      c.Name,
				Payload:   payload,
				CreatedAt: c.CreatedAt,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, srv := range snap.Servers {
			row := models.MetaServer{
				ID:            srv.ID,
				Name:          srv.Name,
				Protocol:      models.Protocol(srv.Protocol),
				Options:       toJSONRaw(file.NodeToRaw(srv.Options)),
				CredentialRef: srv.CredentialRef,
				Enabled:       srv.Enabled,
				UpdatedAt:     srv.UpdatedAt,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, tbl := range snap.Tables {
			row := models.MetaForeignTable{
				ID:         tbl.ID,
				ServerID:   tbl.ServerID,
				SchemaName: tbl.SchemaName,
				Name:       tbl.TableName,
				RemoteName: tbl.RemoteName,
				KeyColumns: encodeKeyColumns(tbl.KeyColumns),
				Options:    toJSONRaw(file.NodeToRaw(tbl.Options)),
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, col := range snap.Columns {
			row := models.MetaForeignColumn{
				ID:         col.ID,
				TableID:    col.TableID,
				Name:       col.Name,
				DataType:   col.DataType,
				RemoteName: col.RemoteName,
				Nullable:   col.Nullable,
				Position:   col.Position,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, fn := range snap.Functions {
			row := models.MetaForeignFunction{
				ID:         fn.ID,
				ServerID:   fn.ServerID,
				SchemaName: fn.SchemaName,
				Name:       fn.Name,
				Operation:  fn.Operation,
				RemotePath: fn.RemotePath,
				Method:     fn.Method,
				Options:    toJSONRaw(file.NodeToRaw(fn.Options)),
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
