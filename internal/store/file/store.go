package file

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/importer"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/errs"
)

type Config struct {
	Path string // file or directory (directory loads all *.yaml / *.yml)
}

type Store struct {
	path       string
	persistPath string
	vault      *auth.Vault
	mu         sync.Mutex
	data       snapshot
}

func New(vault *auth.Vault, cfg Config) (*Store, error) {
	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		return nil, fmt.Errorf("drivers yaml path is required")
	}
	persistPath, err := resolvePersistPath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(persistPath), 0755); err != nil {
		return nil, fmt.Errorf("create drivers dir: %w", err)
	}

	s := &Store{path: path, persistPath: persistPath, vault: vault}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func Ping(cfg Config) error {
	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		return fmt.Errorf("drivers yaml path is required")
	}
	persistPath, err := resolvePersistPath(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(persistPath), 0755); err != nil {
		return fmt.Errorf("create drivers dir: %w", err)
	}
	return nil
}

func resolvePersistPath(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if strings.HasSuffix(path, string(os.PathSeparator)) || looksLikeDirPath(path) {
				return filepath.Join(path, "drivers.yaml"), nil
			}
			return path, nil
		}
		return "", err
	}
	if info.IsDir() {
		return filepath.Join(path, "drivers.yaml"), nil
	}
	return path, nil
}

func looksLikeDirPath(path string) bool {
	base := filepath.Base(path)
	return !strings.Contains(base, ".")
}

func (s *Store) Close() {}

func (s *Store) Vault() *auth.Vault {
	return s.vault
}

func (s *Store) load() error {
	rawDocs, err := s.readDeclarativeSources()
	if err != nil {
		return err
	}
	if len(rawDocs) == 0 {
		return s.persistLocked()
	}
	var importerDocs []importer.DeclarativeDoc
	for _, raw := range rawDocs {
		if len(strings.TrimSpace(string(raw))) == 0 {
			continue
		}
		if isDeclarativeYAML(raw) {
			var doc declarativeDoc
			if err := yaml.Unmarshal(raw, &doc); err != nil {
				return fmt.Errorf("parse drivers yaml: %w", err)
			}
			importerDocs = append(importerDocs, declarativeToImporter(doc))
			continue
		}
		if err := yaml.Unmarshal(raw, &s.data); err != nil {
			return fmt.Errorf("parse drivers file: %w", err)
		}
		return nil
	}
	if len(importerDocs) == 0 {
		return nil
	}
	merged, err := importer.MergeDeclarativeDocs(importerDocs...)
	if err != nil {
		return err
	}
	snap, err := compileDeclarative(s.vault, declarativeFromImporter(merged))
	if err != nil {
		return err
	}
	s.data = snap
	return nil
}

func (s *Store) readDeclarativeSources() ([][]byte, error) {
	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read drivers path: %w", err)
	}
	if !info.IsDir() {
		data, err := os.ReadFile(s.path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("read drivers file: %w", err)
		}
		return [][]byte{data}, nil
	}
	entries, err := os.ReadDir(s.path)
	if err != nil {
		return nil, fmt.Errorf("read drivers dir: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var out [][]byte
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(s.path, name))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		out = append(out, data)
	}
	return out, nil
}

func (s *Store) persistLocked() error {
	doc, err := decompileDeclarative(s.vault, s.data)
	if err != nil {
		return fmt.Errorf("serialize drivers yaml: %w", err)
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal drivers file: %w", err)
	}
	dir := filepath.Dir(s.persistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".meta-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(out); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, s.persistPath)
}

func (s *Store) withLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(); err != nil {
		return err
	}
	return s.persistLocked()
}

func (s *Store) CreateCredential(ctx context.Context, name string, data map[string]any) (*models.Credential, error) {
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	payload, err := s.vault.Encrypt(data)
	if err != nil {
		return nil, err
	}
	c := models.Credential{
		ID:        uuid.New(),
		Name:      name,
		CreatedAt: time.Now().UTC(),
	}
	rec := credentialRecord{
		ID:        c.ID,
		Name:      c.Name,
		Payload:   encodePayload(payload),
		CreatedAt: c.CreatedAt,
	}
	err = s.withLock(func() error {
		for _, existing := range s.data.Credentials {
			if existing.Name == name {
				return fmt.Errorf("credential %q already exists", name)
			}
		}
		s.data.Credentials = append(s.data.Credentials, rec)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) DeleteCredential(ctx context.Context, id uuid.UUID) error {
	return s.withLock(func() error {
		idx := -1
		for i, c := range s.data.Credentials {
			if c.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		for _, srv := range s.data.Servers {
			if srv.CredentialRef != nil && *srv.CredentialRef == id {
				return fmt.Errorf("credential is referenced by server %q", srv.Name)
			}
		}
		s.data.Credentials = append(s.data.Credentials[:idx], s.data.Credentials[idx+1:]...)
		return nil
	})
}

func (s *Store) ResolveCredential(ctx context.Context, ref uuid.UUID) (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.data.Credentials {
		if c.ID != ref {
			continue
		}
		payload, err := decodePayload(c.Payload)
		if err != nil {
			return nil, err
		}
		return s.vault.Decrypt(payload)
	}
	return nil, errs.ErrNotFound
}

func (s *Store) CreateServer(ctx context.Context, req models.CreateServerRequest) (*models.Server, error) {
	if req.Name == "" || req.Protocol == "" {
		return nil, fmt.Errorf("name and protocol are required")
	}
	if req.CredentialRef == nil && len(req.Credential) > 0 {
		credName := strings.TrimSpace(req.CredentialName)
		if credName == "" {
			credName = req.Name + "-credential"
		}
		cred, err := s.CreateCredential(ctx, credName, req.Credential)
		if err != nil {
			return nil, err
		}
		req.CredentialRef = &cred.ID
	}
	if req.CredentialRef != nil && len(req.Credential) > 0 {
		return nil, fmt.Errorf("provide either credentialRef or inline credential, not both")
	}
	opts := req.Options
	if len(opts) == 0 {
		opts = json.RawMessage(`{}`)
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	srv := models.Server{
		ID:            uuid.New(),
		Name:          req.Name,
		Protocol:      req.Protocol,
		Options:       opts,
		CredentialRef: req.CredentialRef,
		Enabled:       enabled,
		UpdatedAt:     time.Now().UTC(),
	}
	rec := serverRecord{
		ID:            srv.ID,
		Name:          srv.Name,
		Protocol:      string(srv.Protocol),
		Options:       rawToNode(srv.Options),
		CredentialRef: srv.CredentialRef,
		Enabled:       srv.Enabled,
		UpdatedAt:     srv.UpdatedAt,
	}
	err := s.withLock(func() error {
		for _, existing := range s.data.Servers {
			if existing.Name == req.Name {
				return fmt.Errorf("server %q already exists", req.Name)
			}
		}
		s.data.Servers = append(s.data.Servers, rec)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &srv, nil
}

func (s *Store) ListServers(ctx context.Context) ([]models.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]models.Server, 0, len(s.data.Servers))
	for _, rec := range s.data.Servers {
		out = append(out, rec.toModel())
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *Store) GetServerByID(ctx context.Context, id uuid.UUID) (*models.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.data.Servers {
		if rec.ID == id {
			srv := rec.toModel()
			return &srv, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (s *Store) GetServerByName(ctx context.Context, name string) (*models.Server, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, rec := range s.data.Servers {
		if rec.Name == name {
			srv := rec.toModel()
			return &srv, nil
		}
	}
	return nil, errs.ErrNotFound
}

func (s *Store) UpdateServer(ctx context.Context, id uuid.UUID, req models.UpdateServerRequest) (*models.Server, error) {
	var updated models.Server
	err := s.withLock(func() error {
		idx := -1
		for i, rec := range s.data.Servers {
			if rec.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		rec := s.data.Servers[idx]
		if len(req.Options) > 0 {
			rec.Options = rawToNode(req.Options)
		}
		if req.CredentialRef != nil {
			rec.CredentialRef = req.CredentialRef
		}
		if req.Enabled != nil {
			rec.Enabled = *req.Enabled
		}
		rec.UpdatedAt = time.Now().UTC()
		s.data.Servers[idx] = rec
		updated = rec.toModel()
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func (s *Store) DeleteServer(ctx context.Context, id uuid.UUID) error {
	return s.withLock(func() error {
		idx := -1
		for i, rec := range s.data.Servers {
			if rec.ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return errs.ErrNotFound
		}
		tableIDs := map[uuid.UUID]struct{}{}
		for _, tbl := range s.data.Tables {
			if tbl.ServerID == id {
				tableIDs[tbl.ID] = struct{}{}
			}
		}
		s.data.Servers = append(s.data.Servers[:idx], s.data.Servers[idx+1:]...)
		s.data.Tables = filterTables(s.data.Tables, id)
		s.data.Functions = filterFunctions(s.data.Functions, id)
		s.data.Columns = filterColumnsByTableIDs(s.data.Columns, tableIDs)
		return nil
	})
}

func (s *Store) CreateTable(ctx context.Context, req models.CreateTableRequest) (*models.Table, []models.Column, error) {
	if req.ServerName == "" || req.TableName == "" {
		return nil, nil, fmt.Errorf("serverName and tableName are required")
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
	err := s.withLock(func() error {
		srvIdx := -1
		for i, rec := range s.data.Servers {
			if rec.Name == req.ServerName {
				srvIdx = i
				break
			}
		}
		if srvIdx < 0 {
			return errs.ErrNotFound
		}
		srv := s.data.Servers[srvIdx]
		for _, existing := range s.data.Tables {
			if existing.SchemaName == schema && existing.TableName == req.TableName {
				return fmt.Errorf("table %s.%s already exists", schema, req.TableName)
			}
		}

		tbl = models.Table{
			ID:         uuid.New(),
			ServerID:   srv.ID,
			SchemaName: schema,
			TableName:  req.TableName,
			RemoteName: req.RemoteName,
			KeyColumns: keyCols,
			Options:    opts,
			ServerName: srv.Name,
		}
		s.data.Tables = append(s.data.Tables, tableRecord{
			ID:         tbl.ID,
			ServerID:   tbl.ServerID,
			SchemaName: tbl.SchemaName,
			TableName:  tbl.TableName,
			RemoteName: tbl.RemoteName,
			KeyColumns: append([]string(nil), tbl.KeyColumns...),
			Options:    rawToNode(tbl.Options),
		})

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
			col := models.Column{
				ID:         uuid.New(),
				TableID:    tbl.ID,
				Name:       c.Name,
				DataType:   dt,
				RemoteName: c.RemoteName,
				Nullable:   nullable,
				Position:   pos,
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
			cols = append(cols, col)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return nil, nil, err
		}
		return nil, nil, err
	}
	return &tbl, cols, nil
}

func (s *Store) ListTables(ctx context.Context) ([]models.Table, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	serverNames := map[uuid.UUID]string{}
	for _, srv := range s.data.Servers {
		serverNames[srv.ID] = srv.Name
	}
	out := make([]models.Table, 0, len(s.data.Tables))
	for _, rec := range s.data.Tables {
		tbl := rec.toModel()
		tbl.ServerName = serverNames[rec.ServerID]
		out = append(out, tbl)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SchemaName == out[j].SchemaName {
			return out[i].TableName < out[j].TableName
		}
		return out[i].SchemaName < out[j].SchemaName
	})
	return out, nil
}

func (s *Store) ListColumns(ctx context.Context, schema, table string) ([]models.Column, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var tableID uuid.UUID
	found := false
	for _, rec := range s.data.Tables {
		if rec.SchemaName == schema && rec.TableName == table {
			tableID = rec.ID
			found = true
			break
		}
	}
	if !found {
		return nil, errs.ErrNotFound
	}
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
	return out, nil
}

func (s *Store) ResolveTable(ctx context.Context, schema, table string) (*models.ResolvedTable, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var tblRec *tableRecord
	for i := range s.data.Tables {
		rec := &s.data.Tables[i]
		if rec.SchemaName == schema && rec.TableName == table {
			tblRec = rec
			break
		}
	}
	if tblRec == nil {
		return nil, errs.ErrNotFound
	}
	var srvRec *serverRecord
	for i := range s.data.Servers {
		rec := &s.data.Servers[i]
		if rec.ID == tblRec.ServerID {
			srvRec = rec
			break
		}
	}
	if srvRec == nil {
		return nil, errs.ErrNotFound
	}
	srv := srvRec.toModel()
	if !srv.Enabled {
		return nil, fmt.Errorf("server %q is disabled", srv.Name)
	}
	rt := models.ResolvedTable{
		Table:  tblRec.toModel(),
		Server: srv,
	}
	rt.Table.ServerName = srv.Name
	for _, colRec := range s.data.Columns {
		if colRec.TableID == tblRec.ID {
			rt.Columns = append(rt.Columns, colRec.toModel())
		}
	}
	sort.Slice(rt.Columns, func(i, j int) bool {
		if rt.Columns[i].Position == rt.Columns[j].Position {
			return rt.Columns[i].Name < rt.Columns[j].Name
		}
		return rt.Columns[i].Position < rt.Columns[j].Position
	})
	return &rt, nil
}

func (s *Store) CreateFunction(ctx context.Context, req models.CreateFunctionRequest) (*models.Function, error) {
	if req.ServerName == "" || req.Name == "" || req.Operation == "" {
		return nil, fmt.Errorf("serverName, name and operation are required")
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
	err := s.withLock(func() error {
		var srvID uuid.UUID
		found := false
		for _, rec := range s.data.Servers {
			if rec.Name == req.ServerName {
				srvID = rec.ID
				found = true
				break
			}
		}
		if !found {
			return errs.ErrNotFound
		}
		for _, existing := range s.data.Functions {
			if existing.SchemaName == schema && existing.Name == req.Name {
				return fmt.Errorf("function %s.%s already exists", schema, req.Name)
			}
		}
		fn = models.Function{
			ID:         uuid.New(),
			ServerID:   srvID,
			SchemaName: schema,
			Name:       req.Name,
			Operation:  req.Operation,
			RemotePath: req.RemotePath,
			Method:     req.Method,
			Options:    opts,
			ServerName: req.ServerName,
		}
		s.data.Functions = append(s.data.Functions, functionRecord{
			ID:         fn.ID,
			ServerID:   fn.ServerID,
			SchemaName: fn.SchemaName,
			Name:       fn.Name,
			Operation:  fn.Operation,
			RemotePath: fn.RemotePath,
			Method:     fn.Method,
			Options:    rawToNode(fn.Options),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &fn, nil
}

func (s *Store) ListFunctions(ctx context.Context) ([]models.Function, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	serverNames := map[uuid.UUID]string{}
	for _, srv := range s.data.Servers {
		serverNames[srv.ID] = srv.Name
	}
	out := make([]models.Function, 0, len(s.data.Functions))
	for _, rec := range s.data.Functions {
		fn := rec.toModel()
		fn.ServerName = serverNames[rec.ServerID]
		out = append(out, fn)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].SchemaName == out[j].SchemaName {
			return out[i].Name < out[j].Name
		}
		return out[i].SchemaName < out[j].SchemaName
	})
	return out, nil
}

func (s *Store) ResolveFunction(ctx context.Context, schema, name string) (*models.ResolvedFunction, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var fnRec *functionRecord
	for i := range s.data.Functions {
		rec := &s.data.Functions[i]
		if rec.SchemaName == schema && rec.Name == name {
			fnRec = rec
			break
		}
	}
	if fnRec == nil {
		return nil, errs.ErrNotFound
	}
	var srvRec *serverRecord
	for i := range s.data.Servers {
		rec := &s.data.Servers[i]
		if rec.ID == fnRec.ServerID {
			srvRec = rec
			break
		}
	}
	if srvRec == nil {
		return nil, errs.ErrNotFound
	}
	srv := srvRec.toModel()
	if !srv.Enabled {
		return nil, fmt.Errorf("server %q is disabled", srv.Name)
	}
	fn := fnRec.toModel()
	fn.ServerName = srv.Name
	return &models.ResolvedFunction{Function: fn, Server: srv}, nil
}

func (s *Store) ServerCredential(ctx context.Context, srv *models.Server) (map[string]any, error) {
	if srv == nil || srv.CredentialRef == nil {
		return nil, nil
	}
	return s.ResolveCredential(ctx, *srv.CredentialRef)
}

func (r serverRecord) toModel() models.Server {
	return models.Server{
		ID:            r.ID,
		Name:          r.Name,
		Protocol:      models.Protocol(r.Protocol),
		Options:       nodeToRaw(r.Options),
		CredentialRef: r.CredentialRef,
		Enabled:       r.Enabled,
		UpdatedAt:     r.UpdatedAt,
	}
}

func (r tableRecord) toModel() models.Table {
	return models.Table{
		ID:         r.ID,
		ServerID:   r.ServerID,
		SchemaName: r.SchemaName,
		TableName:  r.TableName,
		RemoteName: r.RemoteName,
		KeyColumns: append([]string(nil), r.KeyColumns...),
		Options:    nodeToRaw(r.Options),
	}
}

func (r columnRecord) toModel() models.Column {
	return models.Column{
		ID:         r.ID,
		TableID:    r.TableID,
		Name:       r.Name,
		DataType:   r.DataType,
		RemoteName: r.RemoteName,
		Nullable:   r.Nullable,
		Position:   r.Position,
	}
}

func (r functionRecord) toModel() models.Function {
	return models.Function{
		ID:         r.ID,
		ServerID:   r.ServerID,
		SchemaName: r.SchemaName,
		Name:       r.Name,
		Operation:  r.Operation,
		RemotePath: r.RemotePath,
		Method:     r.Method,
		Options:    nodeToRaw(r.Options),
	}
}

func filterTables(in []tableRecord, serverID uuid.UUID) []tableRecord {
	out := in[:0]
	for _, rec := range in {
		if rec.ServerID != serverID {
			out = append(out, rec)
		}
	}
	return out
}

func filterFunctions(in []functionRecord, serverID uuid.UUID) []functionRecord {
	out := in[:0]
	for _, rec := range in {
		if rec.ServerID != serverID {
			out = append(out, rec)
		}
	}
	return out
}

func filterColumnsByTableIDs(in []columnRecord, tableIDs map[uuid.UUID]struct{}) []columnRecord {
	out := in[:0]
	for _, rec := range in {
		if _, drop := tableIDs[rec.TableID]; !drop {
			out = append(out, rec)
		}
	}
	return out
}
