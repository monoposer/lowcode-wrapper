# Driver documentation index

Per-protocol registration, options, capability boundaries, and examples.

| protocol | Doc | Type | CRUD | RPC |
|----------|-----|------|------|-----|
| `http` | [http.md](http.md) | Generic REST | ✓ | ✓ |
| `notion` | [notion.md](notion.md) | HTTP preset | ✓* | ✓* |
| `firebase` | [firebase.md](firebase.md) | HTTP preset | ✓* | ✓* |
| `airtable` | [airtable.md](airtable.md) | HTTP preset | ✓* | ✓* |
| `sheets` | [sheets.md](sheets.md) | HTTP preset | ✓* | ✓* |
| `postgres` | [postgres.md](postgres.md) | Native SQL | ✓ | ✗ |
| `mysql` | [mysql.md](mysql.md) | Native SQL | ✓ | ✗ |
| `mongo` | [mongo.md](mongo.md) | Native document DB | ✓ | ✗ |
| `redis` | [redis.md](redis.md) | Native KV | ✓ | ✗ |
| `file` | [file.md](file.md) | Local files | Select | ✗ |
| `s3` | [s3.md](s3.md) | Object storage | Select | ✗ |

\* HTTP presets delegate to `internal/driver/http`; capabilities depend on whether the upstream API understands the REST semantics dataspan emits.

## Common conventions

- **Registration**: Admin API `POST /admin/api/servers` (`protocol` field, db mode) or `servers[].protocol` in `drivers.yaml`
- **Column mapping**: Data API uses local names `columns[].name`; drivers map outbound to `remote_name`
- **Table mapping**: `tables[].name` is the Data API table name; `remoteName` is the remote resource path
- **Schema**: `tables[].schema` / `schemaName` maps to PostgREST `Accept-Profile` (default `public`); independent of driver protocol

See the selection guide in [../drivers.md](../drivers.md).
