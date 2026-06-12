package file

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/errs"
)

func (s *Store) GetTableByID(ctx context.Context, id uuid.UUID) (*models.Table, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.data.Tables {
		if rec.ID == id {
			tbl := rec.toModel()
			tbl.ServerName = s.serverNameLocked(rec.ServerID)
			return &tbl, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (s *Store) UpdateTable(ctx context.Context, id uuid.UUID, req models.UpdateTableRequest) (*models.Table, error) {
	var updated models.Table
	err := s.withLock(func() error {
		idx := s.tableIndexLocked(id)
		if idx < 0 {
			return errs.ErrNotFound
		}
		rec := &s.data.Tables[idx]
		if req.RemoteName != nil {
			rec.RemoteName = *req.RemoteName
		}
		if req.KeyColumns != nil {
			rec.KeyColumns = append([]string(nil), req.KeyColumns...)
		}
		if len(req.Options) > 0 {
			rec.Options = rawToNode(req.Options)
		}
		updated = rec.toModel()
		updated.ServerName = s.serverNameLocked(rec.ServerID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Store) DeleteTable(ctx context.Context, id uuid.UUID) error {
	return s.withLock(func() error {
		idx := s.tableIndexLocked(id)
		if idx < 0 {
			return errs.ErrNotFound
		}
		s.data.Tables = append(s.data.Tables[:idx], s.data.Tables[idx+1:]...)
		s.data.Columns = filterColumnsByTableIDs(s.data.Columns, map[uuid.UUID]struct{}{id: {}})
		return nil
	})
}

func (s *Store) ListColumnsByTableID(ctx context.Context, tableID uuid.UUID) ([]models.Column, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tableIndexLocked(tableID) < 0 {
		return nil, errs.ErrNotFound
	}
	return s.columnsForTableLocked(tableID), nil
}

func (s *Store) GetColumnByID(ctx context.Context, id uuid.UUID) (*models.Column, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.data.Columns {
		if rec.ID == id {
			out := rec.toModel()
			return &out, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (s *Store) CreateColumn(ctx context.Context, tableID uuid.UUID, req models.CreateColumnRequest) (*models.Column, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	var col models.Column
	err := s.withLock(func() error {
		if s.tableIndexLocked(tableID) < 0 {
			return errs.ErrNotFound
		}
		for _, existing := range s.data.Columns {
			if existing.TableID == tableID && existing.Name == req.Name {
				return fmt.Errorf("column %q already exists", req.Name)
			}
		}
		dt := req.DataType
		if dt == "" {
			dt = "text"
		}
		nullable := true
		if req.Nullable != nil {
			nullable = *req.Nullable
		}
		col = models.Column{
			ID:         uuid.New(),
			TableID:    tableID,
			Name:       req.Name,
			DataType:   dt,
			RemoteName: req.RemoteName,
			Nullable:   nullable,
			Position:   req.Position,
		}
		s.data.Columns = append(s.data.Columns, columnRecord{
			ID:         col.ID,
			TableID:    col.TableID,
			Name:       col.Name,
			DataType:   col.DataType,
			RemoteName: col.RemoteName,
			Nullable:   col.Nullable,
			Position:   col.Position,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &col, nil
}

func (s *Store) UpdateColumn(ctx context.Context, id uuid.UUID, req models.UpdateColumnRequest) (*models.Column, error) {
	var updated models.Column
	err := s.withLock(func() error {
		idx := -1
		for i, rec := range s.data.Columns {
			if rec.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		rec := &s.data.Columns[idx]
		if req.DataType != nil {
			rec.DataType = *req.DataType
		}
		if req.RemoteName != nil {
			rec.RemoteName = *req.RemoteName
		}
		if req.Nullable != nil {
			rec.Nullable = *req.Nullable
		}
		if req.Position != nil {
			rec.Position = *req.Position
		}
		updated = rec.toModel()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Store) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	return s.withLock(func() error {
		idx := -1
		for i, rec := range s.data.Columns {
			if rec.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		s.data.Columns = append(s.data.Columns[:idx], s.data.Columns[idx+1:]...)
		return nil
	})
}

func (s *Store) GetFunctionByID(ctx context.Context, id uuid.UUID) (*models.Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.data.Functions {
		if rec.ID == id {
			fn := rec.toModel()
			fn.ServerName = s.serverNameLocked(rec.ServerID)
			return &fn, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (s *Store) UpdateFunction(ctx context.Context, id uuid.UUID, req models.UpdateFunctionRequest) (*models.Function, error) {
	var updated models.Function
	err := s.withLock(func() error {
		idx := -1
		for i, rec := range s.data.Functions {
			if rec.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		rec := &s.data.Functions[idx]
		if req.Operation != nil {
			rec.Operation = *req.Operation
		}
		if req.RemotePath != nil {
			rec.RemotePath = *req.RemotePath
		}
		if req.Method != nil {
			rec.Method = *req.Method
		}
		if len(req.Options) > 0 {
			rec.Options = rawToNode(req.Options)
		}
		updated = rec.toModel()
		updated.ServerName = s.serverNameLocked(rec.ServerID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Store) DeleteFunction(ctx context.Context, id uuid.UUID) error {
	return s.withLock(func() error {
		idx := -1
		for i, rec := range s.data.Functions {
			if rec.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		s.data.Functions = append(s.data.Functions[:idx], s.data.Functions[idx+1:]...)
		return nil
	})
}

func (s *Store) tableIndexLocked(id uuid.UUID) int {
	for i, rec := range s.data.Tables {
		if rec.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) serverNameLocked(serverID uuid.UUID) string {
	for _, srv := range s.data.Servers {
		if srv.ID == serverID {
			return srv.Name
		}
	}
	return ""
}

func (s *Store) columnsForTableLocked(tableID uuid.UUID) []models.Column {
	out := make([]models.Column, 0)
	for _, rec := range s.data.Columns {
		if rec.TableID != tableID {
			continue
		}
		out = append(out, rec.toModel())
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Position == out[j].Position {
			return out[i].Name < out[j].Name
		}
		return out[i].Position < out[j].Position
	})
	return out
}
