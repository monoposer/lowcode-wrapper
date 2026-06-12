# Protocol drivers

Drivers adapt **remote data sources** to the unified Data API. Register via Admin API (`POST /admin/api/servers`, db mode) or [`drivers.yaml`](../drivers.yaml.example) (`servers[].protocol`).

Registry: `internal/driver/registry.go` · **Per-driver docs**: [`drivers/`](drivers/README.md)

## Store mode

| `DATASPAN_STORE_MODE` | Config source |
|----------------------|---------------|
| `db` (default) | Meta DB + Admin API |
| `file` | `drivers.yaml` |

See [store.md](store.md).

## Architecture

```
HTTP generic     http
HTTP presets     notion · firebase · airtable · sheets  → httpwrap → http
Native protocols postgres · mysql · mongo · redis
File / object    file · s3
```

## Driver index

| protocol | Doc | Summary |
|----------|-----|---------|
| `http` | [drivers/http.md](drivers/http.md) | Any REST / PostgREST upstream |
| `notion` | [drivers/notion.md](drivers/notion.md) | Notion API preset |
| `firebase` | [drivers/firebase.md](drivers/firebase.md) | Firestore REST preset |
| `airtable` | [drivers/airtable.md](drivers/airtable.md) | Airtable API preset |
| `sheets` | [drivers/sheets.md](drivers/sheets.md) | Google Sheets API preset |
| `postgres` | [drivers/postgres.md](drivers/postgres.md) | PostgreSQL |
| `mysql` | [drivers/mysql.md](drivers/mysql.md) | MySQL |
| `mongo` | [drivers/mongo.md](drivers/mongo.md) | MongoDB |
| `redis` | [drivers/redis.md](drivers/redis.md) | Redis |
| `file` | [drivers/file.md](drivers/file.md) | Local CSV/JSON/YAML/XLSX |
| `s3` | [drivers/s3.md](drivers/s3.md) | AWS S3 objects |

## Selection guide

| Need | Use |
|------|-----|
| Any REST API | **`http`** |
| Notion / Firestore / Airtable / Sheets | matching preset, or **`http`** |
| Direct PG / MySQL / Mongo / Redis | native protocol |
| Local files / S3 static data | `file` / `s3` |
| New SaaS | **`http`** + auth + `remote_name` |

## Adding a driver

1. Implement `driver.Driver` in `internal/driver/<name>/`
2. `init()` → `driver.Register(models.ProtocolXxx, New)`
3. Blank import in `cmd/server/main.go`
4. REST-only SaaS → prefer **`httpwrap`** ([notion](drivers/notion.md) as template)
5. Add `models.Protocol` constant + [driver doc](drivers/README.md)
6. Tests (see `internal/driver/http`, `file`)

## Interface

```go
type Driver interface {
    Select(ctx, SelectRequest) ([]map[string]any, error)
    Insert(ctx, RowRequest) (map[string]any, error)
    Update(ctx, RowRequest) (int, error)
    Upsert(ctx, RowRequest) (bool, map[string]any, error)
    Delete(ctx, DeleteRequest) (int, error)
    Invoke(ctx, InvokeRequest) (any, error)
}
```

Defined in `internal/driver/driver.go`.
