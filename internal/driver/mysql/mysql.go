package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolMySQL, New)
}

type Driver struct {
	db       *sql.DB
	database string
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.MySQLServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	dsn := strings.TrimSpace(opts.DSN)
	if dsn == "" && cred != nil {
		dsn = strCred(cred, "dsn")
	}
	if dsn == "" {
		user := strCred(cred, "username")
		pass := strCred(cred, "password")
		host := strCred(cred, "host")
		if host == "" {
			host = "127.0.0.1"
		}
		port := strCred(cred, "port")
		if port == "" {
			port = "3306"
		}
		dbName := strings.TrimSpace(opts.Database)
		if dbName == "" {
			dbName = strCred(cred, "database")
		}
		if user == "" || dbName == "" {
			return nil, fmt.Errorf("mysql server %q requires options.dsn or credential fields", srv.Name)
		}
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, dbName)
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("mysql ping: %w", err)
	}
	dbName := strings.TrimSpace(opts.Database)
	if dbName == "" && cred != nil {
		dbName = strCred(cred, "database")
	}
	return &Driver{db: db, database: dbName}, nil
}

func strCred(cred map[string]any, key string) string {
	if cred == nil {
		return ""
	}
	if v, ok := cred[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func (d *Driver) qualified(table string) string {
	if d.database != "" {
		return fmt.Sprintf("`%s`.`%s`", d.database, table)
	}
	return fmt.Sprintf("`%s`", table)
}

func (d *Driver) remoteTable(resolved *models.ResolvedTable) string {
	return models.RemoteTableName(resolved.Table)
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	table := d.remoteTable(req.Resolved)
	cols := "*"
	if len(req.Select) > 0 {
		quoted := make([]string, len(req.Select))
		for i, c := range req.Select {
			quoted[i] = fmt.Sprintf("`%s`", c)
		}
		cols = strings.Join(quoted, ", ")
	}
	where, args := buildWhereMySQL(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	query := fmt.Sprintf("SELECT %s FROM %s", cols, d.qualified(table))
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
			parts[i] = fmt.Sprintf("`%s` %s", o.Column, dir)
		}
		query += " ORDER BY " + strings.Join(parts, ", ")
	}
	if req.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", req.Limit)
	}
	if req.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", req.Offset)
	}
	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSQLRows(rows)
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	table := d.remoteTable(req.Resolved)
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	cols, args := rowToCols(row)
	if len(cols) == 0 {
		return nil, fmt.Errorf("empty row")
	}
	placeholders := make([]string, len(cols))
	for i := range cols {
		placeholders[i] = "?"
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		d.qualified(table),
		backtickCols(cols),
		strings.Join(placeholders, ", "),
	)
	res, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	if req.PreferRepresentation {
		id, _ := res.LastInsertId()
		if id > 0 {
			return map[string]any{"id": id}, nil
		}
	}
	return nil, nil
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	table := d.remoteTable(req.Resolved)
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	keySet := toSet(req.Resolved.Table.KeyColumns)
	cols, args := rowToColsNonKeys(row, keySet)
	if len(cols) == 0 {
		return 0, fmt.Errorf("no columns to update")
	}
	where, wArgs := buildWhereMySQL(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	setParts := make([]string, len(cols))
	for i, c := range cols {
		setParts[i] = fmt.Sprintf("`%s`=?", c)
	}
	query := fmt.Sprintf("UPDATE %s SET %s", d.qualified(table), strings.Join(setParts, ", "))
	if where != "" {
		query += " WHERE " + where
		args = append(args, wArgs...)
	}
	res, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	table := d.remoteTable(req.Resolved)
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	cols, args := rowToCols(row)
	if len(cols) == 0 {
		return false, nil, fmt.Errorf("empty row")
	}
	placeholders := make([]string, len(cols))
	updates := make([]string, 0, len(cols))
	for i, c := range cols {
		placeholders[i] = "?"
		updates = append(updates, fmt.Sprintf("`%s`=VALUES(`%s`)", c, c))
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		d.qualified(table),
		backtickCols(cols),
		strings.Join(placeholders, ", "),
		strings.Join(updates, ", "),
	)
	res, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, nil, err
	}
	n, _ := res.RowsAffected()
	created := n == 1
	return created, nil, nil
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	table := d.remoteTable(req.Resolved)
	where, args := buildWhereMySQL(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	query := fmt.Sprintf("DELETE FROM %s", d.qualified(table))
	if where != "" {
		query += " WHERE " + where
	}
	res, err := d.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	return nil, fmt.Errorf("mysql driver: invoke not supported")
}

func buildWhereMySQL(filters []postgrest.Filter) (string, []any) {
	if len(filters) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(filters))
	var args []any
	for _, f := range filters {
		col := fmt.Sprintf("`%s`", f.Column)
		switch f.Op {
		case postgrest.OpEq:
			parts = append(parts, col+"=?")
			args = append(args, f.Value)
		case postgrest.OpIn:
			vals := strings.Split(f.Value, ",")
			placeholders := make([]string, len(vals))
			for i, v := range vals {
				placeholders[i] = "?"
				args = append(args, strings.TrimSpace(v))
			}
			parts = append(parts, fmt.Sprintf("%s IN (%s)", col, strings.Join(placeholders, ", ")))
		case postgrest.OpIs:
			if f.Value == "null" {
				parts = append(parts, col+" IS NULL")
			}
		default:
			parts = append(parts, fmt.Sprintf("%s %s ?", col, opSQL(f.Op)))
			args = append(args, f.Value)
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

func backtickCols(cols []string) string {
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = fmt.Sprintf("`%s`", c)
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

func scanSQLRows(rows *sql.Rows) ([]map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(cols))
		for i, c := range cols {
			row[c] = vals[i]
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
