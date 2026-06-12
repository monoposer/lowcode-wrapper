package db

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/errs"
)

func (s *Store) GetTableByID(ctx context.Context, id uuid.UUID) (*models.Table, error) {
	var row models.MetaForeignTable
	if err := s.db.WithContext(ctx).Preload("Server").First(&row, "id = ?", id).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	out := toTable(row, row.Server.Name)
	return &out, nil
}

func (s *Store) UpdateTable(ctx context.Context, id uuid.UUID, req models.UpdateTableRequest) (*models.Table, error) {
	tbl, err := s.GetTableByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if req.RemoteName != nil {
		updates["remote_name"] = *req.RemoteName
		tbl.RemoteName = *req.RemoteName
	}
	if req.KeyColumns != nil {
		updates["key_columns"] = encodeKeyColumns(req.KeyColumns)
		tbl.KeyColumns = append([]string(nil), req.KeyColumns...)
	}
	if len(req.Options) > 0 {
		updates["options"] = toJSONRaw(req.Options)
		tbl.Options = req.Options
	}
	if len(updates) == 0 {
		return tbl, nil
	}
	if err := s.db.WithContext(ctx).Model(&models.MetaForeignTable{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return tbl, nil
}

func (s *Store) DeleteTable(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Delete(&models.MetaForeignTable{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (s *Store) ListColumnsByTableID(ctx context.Context, tableID uuid.UUID) ([]models.Column, error) {
	var rows []models.MetaForeignColumn
	if err := s.db.WithContext(ctx).
		Where("table_id = ?", tableID).
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

func (s *Store) GetColumnByID(ctx context.Context, id uuid.UUID) (*models.Column, error) {
	var row models.MetaForeignColumn
	if err := s.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	out := toColumn(row)
	return &out, nil
}

func (s *Store) CreateColumn(ctx context.Context, tableID uuid.UUID, req models.CreateColumnRequest) (*models.Column, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if _, err := s.GetTableByID(ctx, tableID); err != nil {
		return nil, err
	}
	dt := req.DataType
	if dt == "" {
		dt = "text"
	}
	nullable := true
	if req.Nullable != nil {
		nullable = *req.Nullable
	}
	row := models.MetaForeignColumn{
		ID:         newUUID(),
		TableID:    tableID,
		Name:       req.Name,
		DataType:   dt,
		RemoteName: req.RemoteName,
		Nullable:   nullable,
		Position:   req.Position,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	out := toColumn(row)
	return &out, nil
}

func (s *Store) UpdateColumn(ctx context.Context, id uuid.UUID, req models.UpdateColumnRequest) (*models.Column, error) {
	col, err := s.GetColumnByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if req.DataType != nil {
		updates["data_type"] = *req.DataType
		col.DataType = *req.DataType
	}
	if req.RemoteName != nil {
		updates["remote_name"] = *req.RemoteName
		col.RemoteName = *req.RemoteName
	}
	if req.Nullable != nil {
		updates["nullable"] = *req.Nullable
		col.Nullable = *req.Nullable
	}
	if req.Position != nil {
		updates["position"] = *req.Position
		col.Position = *req.Position
	}
	if len(updates) == 0 {
		return col, nil
	}
	if err := s.db.WithContext(ctx).Model(&models.MetaForeignColumn{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return col, nil
}

func (s *Store) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Delete(&models.MetaForeignColumn{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.ErrNotFound
	}
	return nil
}

func (s *Store) GetFunctionByID(ctx context.Context, id uuid.UUID) (*models.Function, error) {
	var row models.MetaForeignFunction
	if err := s.db.WithContext(ctx).Preload("Server").First(&row, "id = ?", id).Error; err != nil {
		if notFound(err) {
			return nil, errs.ErrNotFound
		}
		return nil, err
	}
	out := toFunction(row, row.Server.Name)
	return &out, nil
}

func (s *Store) UpdateFunction(ctx context.Context, id uuid.UUID, req models.UpdateFunctionRequest) (*models.Function, error) {
	fn, err := s.GetFunctionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if req.Operation != nil {
		updates["operation"] = *req.Operation
		fn.Operation = *req.Operation
	}
	if req.RemotePath != nil {
		updates["remote_path"] = *req.RemotePath
		fn.RemotePath = *req.RemotePath
	}
	if req.Method != nil {
		updates["method"] = *req.Method
		fn.Method = *req.Method
	}
	if len(req.Options) > 0 {
		updates["options"] = toJSONRaw(req.Options)
		fn.Options = req.Options
	}
	if len(updates) == 0 {
		return fn, nil
	}
	if err := s.db.WithContext(ctx).Model(&models.MetaForeignFunction{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return fn, nil
}

func (s *Store) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	res := s.db.WithContext(ctx).Delete(&models.MetaForeignFunction{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errs.ErrNotFound
	}
	return nil
}
