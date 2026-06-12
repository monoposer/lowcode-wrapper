# `postgres` — PostgreSQL

**Code**: `internal/driver/postgres/`

Direct PostgreSQL connection (`pgx`); Data API queries are translated to SQL.

## Capabilities

| Select | Insert | Update | Delete | Upsert | Invoke |
|--------|--------|--------|--------|--------|--------|
| ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |

## Server `options`

| Field | Description |
|-------|-------------|
| `dsn` | PostgreSQL connection string (alternative to credential) |
| `schema` | Default schema; default `public` |

## Credential

| Field | Description |
|-------|-------------|
| `dsn` | Full DSN; takes precedence over options |

## Table mapping

| Field | Description |
|-------|-------------|
| `remoteName` | Remote table name; defaults to `table_name` |
| `schemaName` | Overrides server default schema |

Column `remote_name` maps to the SQL column name.

## Example

```yaml
servers:
  - name: analytics_pg
    protocol: postgres
    credential:
      dsn: postgresql://user:pass@localhost:5432/analytics?sslmode=disable
    options:
      schema: public

tables:
  - server: analytics_pg
    name: orders
    remoteName: orders
    keyColumns: [id]
    columns:
      - name: id
        dataType: text
      - name: amount
        dataType: numeric
```

```bash
curl "$BASE/rest/v1/orders?select=id,amount&limit=10" -H 'Accept-Profile: public'
```

## Notes

- Filter operators support a PostgREST subset (`eq`, `neq`, etc.); the driver builds SQL `WHERE` clauses.
- This is the **remote business database**, not the Meta DB (`DATABASE_URL`).
