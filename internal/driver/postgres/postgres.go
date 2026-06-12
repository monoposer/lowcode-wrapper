package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolPostgres, New)
}

type Driver struct {
	pool   *pgxpool.Pool
	schema string
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.PostgresServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	dsn := strings.TrimSpace(opts.DSN)
	if dsn == "" && cred != nil {
		dsn = strCred(cred, "dsn")
	}
	if dsn == "" {
		return nil, fmt.Errorf("postgres server %q requires options.dsn or credential.dsn", srv.Name)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres ping: %w", err)
	}
	schema := strings.TrimSpace(opts.Schema)
	if schema == "" {
		schema = "public"
	}
	return &Driver{pool: pool, schema: schema}, nil
}

func strCred(cred map[string]any, key string) string {
	if v, ok := cred[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func (d *Driver) qualified(schema, table string) string {
	if schema == "" {
		schema = d.schema
	}
	return fmt.Sprintf("%q.%q", schema, table)
}

func (d *Driver) remoteTable(resolved *models.ResolvedTable) (schema, table string) {
	schema = d.schema
	if resolved.Table.SchemaName != "" {
		schema = resolved.Table.SchemaName
	}
	table = models.RemoteTableName(resolved.Table)
	return schema, table
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	schema, table := d.remoteTable(req.Resolved)
	cols := "*"
	if len(req.Select) > 0 {
		quoted := make([]string, len(req.Select))
		for i, c := range req.Select {
			quoted[i] = fmt.Sprintf("%q", c)
		}
		cols = strings.Join(quoted, ", ")
	}
	where, args := buildWhere(postgrest.MapFilters(req.Filters, req.Resolved.Columns), 1)
	query := fmt.Sprintf("SELECT %s FROM %s", cols, d.qualified(schema, table))
	if where != "" {
		query += " WHERE " + where
	}
	if len(req.Order) > 0 {
		parts := make([]string, len(req.Order))
		for i, o := range req.Order {
			dir := "ASC"
			if o.Desc {
				dir = "DESC"
			}
			parts[i] = fmt.Sprintf("%q %s", o.Column, dir)
		}
		query += " ORDER BY " + strings.Join(parts, ", ")
	}
	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", req.Limit)
	}
	if req.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", req.Offset)
	}
	rows, err := d.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	schema, table := d.remoteTable(req.Resolved)
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	cols, args := rowToCols(row)
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty row")
	}
	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.qualified(schema, table),
		quoteCols(cols),
		strings.Join(placeholders, ", "),
	)
	if req.PreferRepresentation && len(req.Returning) > 0 {
		query += " RETURNING " + quoteCols(req.Returning)
		row := d.pool.QueryRow(ctx, query, args...)
		return scanOne(row, req.Returning)
	}
	_, err := d.pool.Exec(ctx, query, args...)
	return nil, err
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	schema, table := d.remoteTable(req.Resolved)
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	keySet := toSet(req.Resolved.Table.KeyColumns)
	cols, args := rowToColsNonKeys(row, keySet)
	if len(cols) == 0 {
		return 0, fmt.Errorf("no columns to update")
	}
	where, wArgs := buildWhere(postgrest.MapFilters(req.Filters, req.Resolved.Columns), len(args)+1)
	setParts := make([]string, len(cols))
	for i, c := range cols {
		setParts[i] = fmt.Sprintf("%q=$%d", c, i+1)
	}
	query := fmt.Sprintf("UPDATE %s SET %s", d.qualified(schema, table), strings.Join(setParts, ", "))
	if where != "" {
		query += " WHERE " + where
		args = append(args, wArgs...)
	}
	tag, err := d.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	ret, err := d.Insert(ctx, req)
	if err == nil {
		return true, ret, nil
	}
	if !isConflict(err) {
		return false, nil, err
	}
	n, err := d.Update(ctx, req)
	if err != nil {
		return false, nil, err
	}
	_ = n
	return false, nil, nil
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	schema, table := d.remoteTable(req.Resolved)
	where, args := buildWhere(postgrest.MapFilters(req.Filters, req.Resolved.Columns), 1)
	query := fmt.Sprintf("DELETE FROM %s", d.qualified(schema, table))
	if where != "" {
		query += " WHERE " + where
	}
	tag, err := d.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	return nil, fmt.Errorf("postgres driver: invoke not supported; use sql operation via table mapping")
}

func buildWhere(filters []postgrest.Filter, startIdx int) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(filters))
	var args []any
	idx := startIdx
	for _, f := range filters {
		col := fmt.Sprintf("%q", f.Column)
		switch f.Op {
		case postgrest.OpEq:
			parts = append(parts, fmt.Sprintf("%s=$%d", col, idx))
			args = append(args, f.Value)
			idx++
		case postgrest.OpIn:
			vals := strings.Split(f.Value, ",")
			placeholders := make([]string, len(vals))
			for i, v := range vals {
				placeholders[i] = fmt.Sprintf("$%d", idx)
				args = append(args, strings.TrimSpace(v))
				idx++
			}
			parts = append(parts, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")))
		case postgrest.OpIs:
			if f.Value == "null" {
				parts = append(parts, col+" IS NULL")
			} else {
				parts = append(parts, fmt.Sprintf("%s IS %s", col, f.Value))
			}
		default:
			parts = append(parts, fmt.Sprintf("%s %s $%d", col, opSQL(f.Op), idx))
			args = append(args, f.Value)
			idx++
		}
	}
	return strings.Join(parts, " AND "), args
}

func opSQL(op postgrest.Op) string {
	switch op {
	case postgrest.OpNeq:
		return "<>"
	case postgrest.OpGt:
		return ">"
	case postgrest.OpGte:
		return ">="
	case postgrest.OpLt:
		return "<"
	case postgrest.OpLte:
		return "<="
	case postgrest.OpLike:
		return "LIKE"
	default:
		return "="
	}
}

func rowToCols(row map[string]any) ([]string, []any) {
	cols := make([]string, 0, len(row))
	args := make([]any, 0, len(row))
	for k, v := range row {
		cols = append(cols, k)
		args = append(args, v)
	}
	return cols, args
}

func rowToColsNonKeys(row map[string]any, keys map[string]bool) ([]string, []any) {
	cols := make([]string, 0, len(row))
	args := make([]any, 0, len(row))
	for k, v := range row {
		if keys[k] {
			continue
		}
		cols = append(cols, k)
		args = append(args, v)
	}
	return cols, args
}

func quoteCols(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = fmt.Sprintf("%q", c)
	}
	return strings.Join(quoted, ", ")
}

func toSet(keys []string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out
}

func isConflict(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "duplicate") || strings.Contains(s, "unique") || strings.Contains(s, "23505")
}

func scanRows(rows pgx.Rows) ([]map[string]any, error) {
	descs := rows.FieldDescriptions()
	var out []map[string]any
	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			return nil, err
		}
		row := make(map[string]any, len(descs))
		for i, d := range descs {
			row[string(d.Name)] = vals[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func scanOne(row pgx.Row, returning []string) (map[string]any, error) {
	dest := make([]any, len(returning))
	for i := range returning {
		dest[i] = new(any)
	}
	if err := row.Scan(dest...); err != nil {
		return nil, err
	}
	out := make(map[string]any, len(returning))
	for i, col := range returning {
		out[col] = *(dest[i].(*any))
	}
	return out, nil
}
