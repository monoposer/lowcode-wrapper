# `mysql` — MySQL

**Code**: `internal/driver/mysql/`

Direct MySQL connection (`database/sql` + go-sql-driver).

## Capabilities

| Select | Insert | Update | Delete | Upsert | Invoke |
|--------|--------|--------|--------|--------|--------|
| ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |

## Server `options`

| Field | Description |
|-------|-------------|
| `dsn` | Full DSN |
| `database` | Database name when DSN omits it |

## Credential

Option A — full DSN:

```yaml
credential:
  dsn: user:pass@tcp(localhost:3306)/app?parseTime=true
```

Option B — individual fields:

| Field | Default |
|-------|---------|
| `username` | required |
| `password` | |
| `host` | `127.0.0.1` |
| `port` | `3306` |
| `database` | required (or `options.database`) |

## Table mapping

`remoteName` → table name; column `remote_name` → column name.

## Example

```yaml
servers:
  - name: app_mysql
    protocol: mysql
    credential:
      username: app
      password: ${MYSQL_PASSWORD}
      host: db.internal
      database: shop
```

## Notes

- Enable `parseTime=true` in the DSN for correct time handling.
- Upsert uses MySQL `ON DUPLICATE KEY UPDATE` (requires a suitable unique key).
