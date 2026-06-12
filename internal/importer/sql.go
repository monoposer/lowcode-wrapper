package importer

import (
	"fmt"
	"regexp"
	"strings"
)

type SQLDialect string

const (
	DialectPostgres SQLDialect = "postgres"
	DialectMySQL    SQLDialect = "mysql"
	DialectSQLite   SQLDialect = "sqlite"
)

func ParseSQLDialect(s string) (SQLDialect, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "postgres", "pg", "postgresql":
		return DialectPostgres, nil
	case "mysql", "maria", "mariadb":
		return DialectMySQL, nil
	case "sqlite", "sqlite3":
		return DialectSQLite, nil
	default:
		return "", fmt.Errorf("unsupported sql dialect %q (use postgres, mysql, or sqlite)", s)
	}
}

type SQLOptions struct {
	ServerName string
	Schema     string
	Dialect    SQLDialect
}

var createTableRe = regexp.MustCompile(`(?is)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?`)

// SQLToDeclarative parses CREATE TABLE statements into a drivers YAML document.
func SQLToDeclarative(content string, opts SQLOptions) (DeclarativeDoc, error) {
	if opts.Dialect == "" {
		return DeclarativeDoc{}, fmt.Errorf("sql dialect is required (postgres, mysql, or sqlite)")
	}
	serverName := strings.TrimSpace(opts.ServerName)
	if serverName == "" {
		serverName = "imported_db"
	}
	defaultSchema := strings.TrimSpace(opts.Schema)
	if defaultSchema == "" {
		defaultSchema = "public"
	}

	protocol, cred, serverOpts := serverTemplate(opts.Dialect, defaultSchema)
	enabled := true
	doc := DeclarativeDoc{
		Servers: []DeclServer{{
			Name:       serverName,
			Protocol:   protocol,
			Enabled:    &enabled,
			Credential: cred,
			Options:    serverOpts,
		}},
	}

	tables, err := parseCreateTables(content, opts.Dialect, defaultSchema)
	if err != nil {
		return DeclarativeDoc{}, err
	}
	if len(tables) == 0 {
		return DeclarativeDoc{}, fmt.Errorf("sql: no CREATE TABLE statements found")
	}
	for _, tbl := range tables {
		cols := make([]DeclColumn, len(tbl.Columns))
		for i, c := range tbl.Columns {
			nullable := c.Nullable
			cols[i] = DeclColumn{
				Name:     c.Name,
				DataType: c.DataType,
				Nullable: &nullable,
				Position: i,
			}
		}
		keyCols := tbl.KeyColumns
		if len(keyCols) == 0 && len(cols) > 0 {
			keyCols = []string{cols[0].Name}
		}
		doc.Tables = append(doc.Tables, DeclTable{
			Server:     serverName,
			Schema:     tbl.Schema,
			Name:       tbl.Name,
			RemoteName: tbl.Name,
			KeyColumns: keyCols,
			Columns:    cols,
		})
	}
	return doc, nil
}

func serverTemplate(d SQLDialect, schema string) (protocol string, cred map[string]any, opts map[string]any) {
	opts = map[string]any{"schema": schema}
	switch d {
	case DialectPostgres:
		return "postgres", map[string]any{
			"host":     "${DB_HOST}",
			"port":     "${DB_PORT}",
			"username": "${DB_USER}",
			"password": "${DB_PASSWORD}",
			"database": "${DB_NAME}",
		}, opts
	case DialectMySQL:
		return "mysql", map[string]any{
			"host":     "${DB_HOST}",
			"port":     "${DB_PORT}",
			"username": "${DB_USER}",
			"password": "${DB_PASSWORD}",
			"database": "${DB_NAME}",
		}, opts
	case DialectSQLite:
		return "sqlite", map[string]any{
			"dsn": "${SQLITE_DSN}",
		}, map[string]any{
			"schema": schema,
		}
	default:
		return "postgres", nil, opts
	}
}

type parsedTable struct {
	Schema     string
	Name       string
	KeyColumns []string
	Columns    []parsedColumn
}

type parsedColumn struct {
	Name     string
	DataType string
	Nullable bool
	Primary  bool
}

func parseCreateTables(content string, dialect SQLDialect, defaultSchema string) ([]parsedTable, error) {
	rest := content
	var out []parsedTable
	for {
		loc := createTableRe.FindStringIndex(rest)
		if loc == nil {
			break
		}
		rest = rest[loc[1]:]
		stmt, remainder, ok := extractStatement(rest)
		if !ok {
			break
		}
		rest = remainder
		tbl, err := parseOneCreateTable(stmt, dialect, defaultSchema)
		if err != nil {
			return nil, err
		}
		if tbl.Name != "" {
			out = append(out, tbl)
		}
	}
	return out, nil
}

func extractStatement(s string) (stmt, rest string, ok bool) {
	depth := 0
	inSingle := false
	inDouble := false
	inBacktick := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch == '\'' && !inDouble && !inBacktick:
			inSingle = !inSingle
		case ch == '"' && !inSingle && !inBacktick:
			inDouble = !inDouble
		case ch == '`' && !inSingle && !inDouble:
			inBacktick = !inBacktick
		case !inSingle && !inDouble && !inBacktick:
			if ch == '(' {
				depth++
			} else if ch == ')' {
				depth--
			} else if ch == ';' && depth == 0 {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	if depth == 0 {
		return strings.TrimSpace(s), "", true
	}
	return "", "", false
}

func parseOneCreateTable(stmt string, dialect SQLDialect, defaultSchema string) (parsedTable, error) {
	stmt = strings.TrimSpace(stmt)
	parts := strings.Fields(stmt)
	if len(parts) < 3 {
		return parsedTable{}, nil
	}
	qualified := parts[0]
	bodyStart := strings.Index(stmt, "(")
	if bodyStart < 0 {
		return parsedTable{}, nil
	}
	schema, name := splitQualifiedName(qualified, dialect, defaultSchema)
	body := stmt[bodyStart+1:]
	bodyEnd := strings.LastIndex(body, ")")
	if bodyEnd < 0 {
		return parsedTable{}, fmt.Errorf("sql: unclosed column list for table %q", name)
	}
	body = body[:bodyEnd]
	cols, tableKeys, err := parseColumnDefinitions(body, dialect)
	if err != nil {
		return parsedTable{}, err
	}
	keyCols := tableKeys
	if len(keyCols) == 0 {
		for _, c := range cols {
			if c.Primary {
				keyCols = append(keyCols, c.Name)
			}
		}
	}
	return parsedTable{
		Schema:     schema,
		Name:       name,
		KeyColumns: keyCols,
		Columns:    cols,
	}, nil
}

func splitQualifiedName(raw string, dialect SQLDialect, defaultSchema string) (schema, name string) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "`\"")
	if dialect == DialectMySQL {
		if i := strings.LastIndex(raw, "."); i >= 0 {
			return strings.Trim(raw[:i], "`\""), strings.Trim(raw[i+1:], "`\"")
		}
		return defaultSchema, raw
	}
	if strings.Contains(raw, ".") {
		parts := strings.SplitN(raw, ".", 2)
		return strings.Trim(parts[0], `"`), strings.Trim(parts[1], `"`)
	}
	return defaultSchema, strings.Trim(raw, `"`)
}

func parseColumnDefinitions(body string, dialect SQLDialect) ([]parsedColumn, []string, error) {
	var cols []parsedColumn
	var tableKeys []string
	for _, chunk := range splitColumnChunks(body) {
		chunk = strings.TrimSpace(chunk)
		if chunk == "" {
			continue
		}
		upper := strings.ToUpper(chunk)
		if strings.HasPrefix(upper, "PRIMARY KEY") {
			tableKeys = extractKeyColumns(chunk)
			continue
		}
		if strings.HasPrefix(upper, "UNIQUE") ||
			strings.HasPrefix(upper, "CONSTRAINT") ||
			strings.HasPrefix(upper, "FOREIGN KEY") ||
			strings.HasPrefix(upper, "CHECK") ||
			strings.HasPrefix(upper, "INDEX") ||
			strings.HasPrefix(upper, "KEY ") {
			continue
		}
		col, err := parseColumnLine(chunk, dialect)
		if err != nil {
			return nil, nil, err
		}
		if col.Name != "" {
			cols = append(cols, col)
		}
	}
	return cols, tableKeys, nil
}

func splitColumnChunks(body string) []string {
	var chunks []string
	var b strings.Builder
	depth := 0
	inSingle := false
	inDouble := false
	inBacktick := false
	for i := 0; i < len(body); i++ {
		ch := body[i]
		switch {
		case ch == '\'' && !inDouble && !inBacktick:
			inSingle = !inSingle
			b.WriteByte(ch)
		case ch == '"' && !inSingle && !inBacktick:
			inDouble = !inDouble
			b.WriteByte(ch)
		case ch == '`' && !inSingle && !inDouble:
			inBacktick = !inBacktick
			b.WriteByte(ch)
		case !inSingle && !inDouble && !inBacktick:
			if ch == '(' {
				depth++
				b.WriteByte(ch)
			} else if ch == ')' {
				depth--
				b.WriteByte(ch)
			} else if ch == ',' && depth == 0 {
				chunks = append(chunks, b.String())
				b.Reset()
			} else {
				b.WriteByte(ch)
			}
		default:
			b.WriteByte(ch)
		}
	}
	if tail := strings.TrimSpace(b.String()); tail != "" {
		chunks = append(chunks, tail)
	}
	return chunks
}

func parseColumnLine(line string, dialect SQLDialect) (parsedColumn, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return parsedColumn{}, nil
	}
	name := extractFirstIdentifier(line)
	if name == "" {
		return parsedColumn{}, nil
	}
	rest := strings.TrimSpace(line[len(name):])
	if strings.HasPrefix(name, "`") || strings.HasPrefix(name, `"`) {
		name = strings.Trim(name, "`\"")
	}
	upper := strings.ToUpper(rest)
	primary := strings.Contains(upper, "PRIMARY KEY")
	nullable := !strings.Contains(upper, "NOT NULL")
	dataType := extractDataType(rest, dialect)
	if strings.EqualFold(dataType, "serial") || strings.EqualFold(dataType, "bigserial") {
		primary = true
	}
	return parsedColumn{
		Name:     name,
		DataType: normalizeDataType(dataType),
		Nullable: nullable && !primary,
		Primary:  primary,
	}, nil
}

func extractFirstIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if s[0] == '`' || s[0] == '"' {
		quote := s[0]
		if end := strings.IndexByte(s[1:], quote); end >= 0 {
			return s[:end+2]
		}
		return s
	}
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func extractDataType(rest string, dialect SQLDialect) string {
	rest = strings.TrimSpace(rest)
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "text"
	}
	typ := fields[0]
	if len(fields) > 1 && strings.EqualFold(typ, "character") && strings.EqualFold(fields[1], "varying") {
		return "varchar"
	}
	if strings.Contains(typ, "(") {
		typ = strings.Split(typ, "(")[0]
	}
	return typ
}

func normalizeDataType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	switch t {
	case "int", "integer", "int4", "serial", "bigserial", "smallint", "mediumint", "tinyint", "bigint", "int8", "int2":
		return "integer"
	case "bool", "boolean":
		return "boolean"
	case "float", "float4", "float8", "double", "real", "decimal", "numeric":
		return "numeric"
	case "timestamp", "timestamptz", "datetime", "date", "time":
		return "timestamp"
	case "uuid":
		return "uuid"
	case "json", "jsonb":
		return "json"
	case "bytea", "blob", "binary":
		return "binary"
	default:
		return "text"
	}
}

func extractKeyColumns(line string) []string {
	start := strings.Index(line, "(")
	end := strings.LastIndex(line, ")")
	if start < 0 || end <= start {
		return nil
	}
	inner := line[start+1 : end]
	var keys []string
	for _, part := range strings.Split(inner, ",") {
		part = strings.Trim(strings.TrimSpace(part), "`\"")
		if part != "" {
			keys = append(keys, part)
		}
	}
	return keys
}
