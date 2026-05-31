package service

import (
	"context"
	"fmt"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/postgrest"
	store "lowcode-wrapper/internal/store/postgres"
)

type Engine struct {
	Store *store.Store
}

func NewEngine(s *store.Store) *Engine {
	return &Engine{Store: s}
}

func (e *Engine) DriverFor(ctx context.Context, srv models.Server) (driver.Driver, error) {
	cred, err := e.Store.ServerCredential(ctx, &srv)
	if err != nil {
		return nil, err
	}
	return driver.New(ctx, srv, cred)
}

func (e *Engine) Select(ctx context.Context, schema, table string, q postgrest.Query) ([]map[string]any, error) {
	resolved, err := e.Store.ResolveTable(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	drv, err := e.DriverFor(ctx, resolved.Server)
	if err != nil {
		return nil, err
	}
	return drv.Select(ctx, driver.SelectRequest{
		Resolved: resolved,
		Select:   q.Select,
		Filters:  q.Filters,
		Order:    q.Order,
		Limit:    q.Limit,
		Offset:   q.Offset,
	})
}

func (e *Engine) Insert(ctx context.Context, schema, table string, row map[string]any, prefer postgrest.Prefer) (map[string]any, error) {
	resolved, err := e.Store.ResolveTable(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	drv, err := e.DriverFor(ctx, resolved.Server)
	if err != nil {
		return nil, err
	}
	req := driver.RowRequest{
		Resolved:             resolved,
		Row:                  row,
		PreferRepresentation: prefer.Representation,
		PreferUpsert:         prefer.Upsert,
	}
	if prefer.Upsert {
		created, ret, err := drv.Upsert(ctx, req)
		if err != nil {
			return nil, err
		}
		_ = created
		return ret, nil
	}
	return drv.Insert(ctx, req)
}

func (e *Engine) Update(ctx context.Context, schema, table string, q postgrest.Query, row map[string]any) (int, error) {
	resolved, err := e.Store.ResolveTable(ctx, schema, table)
	if err != nil {
		return 0, err
	}
	drv, err := e.DriverFor(ctx, resolved.Server)
	if err != nil {
		return 0, err
	}
	return drv.Update(ctx, driver.RowRequest{
		Resolved: resolved,
		Row:      row,
		Filters:  q.Filters,
	})
}

func (e *Engine) Delete(ctx context.Context, schema, table string, q postgrest.Query) (int, error) {
	resolved, err := e.Store.ResolveTable(ctx, schema, table)
	if err != nil {
		return 0, err
	}
	drv, err := e.DriverFor(ctx, resolved.Server)
	if err != nil {
		return 0, err
	}
	return drv.Delete(ctx, driver.DeleteRequest{
		Resolved: resolved,
		Filters:  q.Filters,
	})
}

func (e *Engine) InvokeRPC(ctx context.Context, schema, name string, body map[string]any, query map[string]string) (any, error) {
	resolved, err := e.Store.ResolveFunction(ctx, schema, name)
	if err != nil {
		return nil, err
	}
	drv, err := e.DriverFor(ctx, resolved.Server)
	if err != nil {
		return nil, err
	}

	switch resolved.Function.Operation {
	case "invoke":
		return drv.Invoke(ctx, driver.InvokeRequest{
			Resolved: resolved,
			Body:     body,
			Query:    query,
		})
	case "select", "insert", "update", "upsert", "delete":
		return e.invokeTableOp(ctx, resolved, body, query)
	default:
		return nil, fmt.Errorf("unknown function operation %q", resolved.Function.Operation)
	}
}

func (e *Engine) invokeTableOp(ctx context.Context, rf *models.ResolvedFunction, body map[string]any, query map[string]string) (any, error) {
	// Functions can target a table via options.tableSchema + tableName
	opts, err := models.ParseServerOptions[struct {
		TableSchema string `json:"tableSchema"`
		TableName   string `json:"tableName"`
	}](rf.Function.Options)
	if err != nil {
		return nil, err
	}
	schema := opts.TableSchema
	if schema == "" {
		schema = rf.Function.SchemaName
	}
	if opts.TableName == "" {
		return nil, fmt.Errorf("function options.tableName required for operation %q", rf.Function.Operation)
	}
	q := postgrest.Query{}
	for k, v := range query {
		q.Filters = append(q.Filters, postgrest.Filter{Column: k, Op: postgrest.OpEq, Value: v})
	}
	switch rf.Function.Operation {
	case "select":
		return e.Select(ctx, schema, opts.TableName, q)
	case "insert":
		return e.Insert(ctx, schema, opts.TableName, body, postgrest.Prefer{Representation: true})
	case "update":
		n, err := e.Update(ctx, schema, opts.TableName, q, body)
		return map[string]any{"affected": n}, err
	case "delete":
		n, err := e.Delete(ctx, schema, opts.TableName, q)
		return map[string]any{"affected": n}, err
	case "upsert":
		return e.Insert(ctx, schema, opts.TableName, body, postgrest.Prefer{Representation: true, Upsert: true})
	default:
		return nil, fmt.Errorf("unsupported table operation %q", rf.Function.Operation)
	}
}
