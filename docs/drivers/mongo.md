# `mongo` — MongoDB

**Code**: `internal/driver/mongo/`

Official Go driver; collection-level CRUD.

## Capabilities

| Select | Insert | Update | Delete | Upsert | Invoke |
|--------|--------|--------|--------|--------|--------|
| ✓ | ✓ | ✓ | ✓ | ✓ | ✗ |

## Server `options`

| Field | Required | Description |
|-------|----------|-------------|
| `uri` | ✓* | MongoDB connection URI |
| `database` | ✓ | Default database name |

\* `uri` may be provided in credential instead.

## Credential

| Field | Description |
|-------|-------------|
| `uri` | MongoDB URI |
| `database` | Alternative to `options.database` |

## Table `options`

| Field | Description |
|-------|-------------|
| `collection` | Collection name; defaults to `remoteName` / `table_name` |

## Example

```yaml
servers:
  - name: mongo_main
    protocol: mongo
    credential:
      uri: mongodb://localhost:27017
    options:
      database: app

tables:
  - server: mongo_main
    name: users
    remoteName: users
    options:
      collection: users
    keyColumns: [_id]
```

## Notes

- `_id` may be auto-populated in the returned row after insert.
- Filters and sort map to MongoDB query and sort documents.
