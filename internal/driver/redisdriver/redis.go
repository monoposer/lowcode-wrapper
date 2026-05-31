package redisdriver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolRedis, New)
}

type Driver struct {
	rdb *redis.Client
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.RedisServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	var client *redis.Client
	if u := strings.TrimSpace(opts.URL); u != "" {
		opt, err := redis.ParseURL(u)
		if err != nil {
			return nil, err
		}
		client = redis.NewClient(opt)
	} else {
		addr := strings.TrimSpace(opts.Addr)
		if addr == "" && cred != nil {
			addr = strings.TrimSpace(fmt.Sprint(cred["addr"]))
		}
		if addr == "" {
			addr = "localhost:6379"
		}
		pass := ""
		if cred != nil {
			pass = fmt.Sprint(cred["password"])
		}
		user := strings.TrimSpace(opts.Username)
		client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Username: user,
			Password: pass,
			DB:       opts.DB,
		})
	}
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis ping: %w", err)
	}
	return &Driver{rdb: client}, nil
}

func (d *Driver) tableOpts(resolved *models.ResolvedTable) (prefix, keyType string) {
	topts, _ := models.ParseServerOptions[models.RedisTableOptions](resolved.Table.Options)
	prefix = strings.TrimSpace(topts.KeyPrefix)
	if prefix == "" {
		prefix = models.RemoteTableName(resolved.Table)
	}
	if prefix != "" && !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	keyType = strings.ToLower(strings.TrimSpace(topts.Type))
	if keyType == "" {
		keyType = "hash"
	}
	return prefix, keyType
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	prefix, keyType := d.tableOpts(req.Resolved)
	pattern := prefix + "*"
	var keys []string
	iter := d.rdb.Scan(ctx, 0, pattern, 200).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	var rows []map[string]any
	for _, key := range keys {
		row, err := d.readKey(ctx, key, keyType)
		if err != nil {
			return nil, err
		}
		if row == nil {
			continue
		}
		row["_key"] = key
		if len(req.Resolved.Table.KeyColumns) > 0 {
			row[req.Resolved.Table.KeyColumns[0]] = strings.TrimPrefix(key, prefix)
		}
		rows = append(rows, row)
	}
	rows = applyFilters(rows, postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	rows = applySelect(rows, req.Select)
	if req.Offset > 0 && req.Offset < len(rows) {
		rows = rows[req.Offset:]
	} else if req.Offset >= len(rows) {
		rows = nil
	}
	if req.Limit > 0 && len(rows) > req.Limit {
		rows = rows[:req.Limit]
	}
	return postgrest.MapRowsFromRemote(rows, req.Resolved.Columns), nil
}

func (d *Driver) readKey(ctx context.Context, key, keyType string) (map[string]any, error) {
	switch keyType {
	case "string":
		val, err := d.rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		return map[string]any{"value": val}, nil
	case "json":
		val, err := d.rdb.Get(ctx, key).Result()
		if err == redis.Nil {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(val), &row); err != nil {
			return map[string]any{"value": val}, nil
		}
		return row, nil
	default:
		m, err := d.rdb.HGetAll(ctx, key).Result()
		if err != nil {
			return nil, err
		}
		if len(m) == 0 {
			return nil, nil
		}
		row := make(map[string]any, len(m))
		for k, v := range m {
			row[k] = v
		}
		return row, nil
	}
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	prefix, keyType := d.tableOpts(req.Resolved)
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	key := prefix + keyFromRow(row, req.Resolved.Table.KeyColumns)
	switch keyType {
	case "hash":
		fields := make([]any, 0, len(row)*2)
		for k, v := range row {
			fields = append(fields, k, fmt.Sprint(v))
		}
		if err := d.rdb.HSet(ctx, key, fields...).Err(); err != nil {
			return nil, err
		}
	case "json":
		b, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		if err := d.rdb.Set(ctx, key, b, 0).Err(); err != nil {
			return nil, err
		}
	default:
		if v, ok := row["value"]; ok {
			if err := d.rdb.Set(ctx, key, fmt.Sprint(v), 0).Err(); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("redis string insert requires value column")
		}
	}
	if req.PreferRepresentation {
		return postgrest.MapRowsFromRemote([]map[string]any{row}, req.Resolved.Columns)[0], nil
	}
	return nil, nil
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	prefix, keyType := d.tableOpts(req.Resolved)
	if keyType != "hash" {
		return 0, fmt.Errorf("redis update only supported for hash keys")
	}
	filters := postgrest.MapFilters(req.Filters, req.Resolved.Columns)
	if len(filters) != 1 || filters[0].Op != postgrest.OpEq {
		return 0, fmt.Errorf("redis update requires single eq filter on key column")
	}
	key := prefix + filters[0].Value
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	fields := make([]any, 0, len(row)*2)
	for k, v := range row {
		fields = append(fields, k, fmt.Sprint(v))
	}
	if err := d.rdb.HSet(ctx, key, fields...).Err(); err != nil {
		return 0, err
	}
	return 1, nil
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	ret, err := d.Insert(ctx, req)
	return true, ret, err
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	prefix, _ := d.tableOpts(req.Resolved)
	filters := postgrest.MapFilters(req.Filters, req.Resolved.Columns)
	n := 0
	for _, f := range filters {
		if f.Op != postgrest.OpEq {
			continue
		}
		if err := d.rdb.Del(ctx, prefix+f.Value).Err(); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	return nil, fmt.Errorf("redis driver: invoke not supported")
}

func keyFromRow(row map[string]any, keyCols []string) string {
	if len(keyCols) > 0 {
		if v, ok := row[keyCols[0]]; ok {
			return fmt.Sprint(v)
		}
	}
	if v, ok := row["id"]; ok {
		return fmt.Sprint(v)
	}
	return fmt.Sprint(len(row))
}

func applyFilters(rows []map[string]any, filters []postgrest.Filter) []map[string]any {
	if len(filters) == 0 {
		return rows
	}
	var out []map[string]any
	for _, row := range rows {
		if matchFilters(row, filters) {
			out = append(out, row)
		}
	}
	return out
}

func matchFilters(row map[string]any, filters []postgrest.Filter) bool {
	for _, f := range filters {
		val, ok := row[f.Column]
		if f.Op == postgrest.OpEq {
			if !ok || fmt.Sprint(val) != f.Value {
				return false
			}
		}
	}
	return true
}

func applySelect(rows []map[string]any, cols []string) []map[string]any {
	if len(cols) == 0 {
		return rows
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		m := make(map[string]any, len(cols))
		for _, c := range cols {
			if v, ok := row[c]; ok {
				m[c] = v
			}
		}
		out[i] = m
	}
	return out
}
