# `redis` — Redis

**Code**: `internal/driver/redis/`

Key-value logical tables via `go-redis`.

## Capabilities

| Select | Insert | Update | Delete | Upsert | Invoke |
|--------|--------|--------|--------|--------|--------|
| ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |

## Server `options`

| Field | Description |
|-------|-------------|
| `url` | Redis URL (preferred), e.g. `redis://:pass@localhost:6379/0` |
| `addr` | `host:port`; default `localhost:6379` |
| `username` | ACL username |
| `db` | Logical DB index |

## Credential

| Field | Description |
|-------|-------------|
| `addr` | Overrides `options.addr` |
| `password` | Redis password |

## Table `options`

| Field | Default | Description |
|-------|---------|-------------|
| `keyPrefix` | `table_name` | Key prefix; `:` appended automatically |
| `type` | `hash` | Value layout: `hash` · `string` · `json` |

Select uses `SCAN` with pattern `prefix*`; each key is one row; `_key` holds the full Redis key.

## Example

```yaml
servers:
  - name: cache
    protocol: redis
    options:
      url: redis://localhost:6379/0

tables:
  - server: cache
    name: sessions
    options:
      keyPrefix: session:
      type: hash
    keyColumns: [id]
```

## Notes

- Large keyspaces make `SCAN` slow; constrain prefix scope in production.
- Not relational; column mapping depends on hash fields or JSON structure.
