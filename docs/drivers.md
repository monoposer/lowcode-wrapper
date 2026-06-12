# Protocol drivers

Drivers adapt **remote data sources** to the unified API. After registering `wrapper_server.protocol` via Admin API, the engine instantiates the driver with `driver.New(protocol)`.

Registry: `internal/driver/registry.go` (each driver calls `driver.Register` in `init()`).

## Where metadata comes from (store mode)

Drivers are registered in code; Foreign Server / Table **configuration** is loaded per store mode:

| `WRAPPER_STORE_MODE` | Config source | Typical use |
|----------------------|---------------|-------------|
| `postgres` (default) | Meta DB + Admin API | Production, dynamic registration |
| `file` | [`drivers.yaml`](../drivers.yaml.example) | Local dev, GitOps-style config |

```bash
WRAPPER_STORE_MODE=file
WRAPPER_DRIVERS_FILE=./drivers.yaml
cp drivers.yaml.example drivers.yaml
```

YAML selects drivers via `servers[].protocol`. See [store.md](store.md) and [`drivers.yaml.example`](../drivers.yaml.example).

## Overview

```
┌─────────────────────────────────────────────────────────────┐
│  HTTP generic layer                                          │
│  http — any REST / PostgREST-style upstream                   │
└─────────────────────────────────────────────────────────────┘
         ▲ delegates
┌────────┴────────┐
│  HTTP presets    │  notion, firebase, airtable, sheets
└─────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  Native protocols                                            │
│  postgres · mysql · mongo · redis                            │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  File / object layer                                         │
│  file (local dir) · s3 (object storage)                      │
└─────────────────────────────────────────────────────────────┘
```

## 1. HTTP generic — `http`

**Use when**

- Custom REST APIs, internal microservices
- Any third-party HTTP API (Stripe, GitHub, ERP, etc.)
- Upstream is already PostgREST or compatible

**Why prefer `http`**

No dedicated driver per SaaS. Configure `endpoint`, `auth`, `headers`, and `remote_name` column mapping.

```json
{
  "name": "partner_api",
  "protocol": "http",
  ...
}
```

File-mode YAML equivalent: `partner_api` in `drivers.yaml.example`.

**Capabilities**: Select / Insert / Update / Delete / Invoke (RPC)

**Auth types** (`options.auth.type`): `NONE` · `BASIC` · `API_KEY` · `BEARER_TOKEN` · `CLIENT_CREDENTIALS` · `UNIVERSAL`

**Code**: `internal/driver/http/` — tests: `go test ./internal/driver/http/... -v`

---

## 2. HTTP presets — via `httpwrap`

Pre-fills endpoint, headers, and Bearer rules for common SaaS; still uses `http` underneath.

| protocol | Default endpoint | Credential fields | Code |
|----------|------------------|---------------------|------|
| `notion` | `https://api.notion.com/v1` | `token` / `integrationToken` | `notion/` |
| `firebase` | Firestore REST (from `projectId`) | `accessToken` / `token` | `firebase/` |
| `airtable` | `https://api.airtable.com/v0` | `token` / `personalAccessToken` | `airtable/` |
| `sheets` | `https://sheets.googleapis.com/v4` | `accessToken` / `token` | `sheets/` |

Wrapper: `internal/driver/httpwrap/wrap.go` → `NewHTTPDriver()` merges defaults then calls `http.New()`.

If presets do not match your upstream, use **`protocol: http`** with a manual endpoint.

---

## 3. Native protocols

Direct database or KV access (no HTTP hop).

| protocol | Description | Main options | CRUD | RPC |
|----------|-------------|--------------|------|-----|
| `postgres` | PostgreSQL | `dsn`, `schema?` | ✓ | ✗ |
| `mysql` | MySQL | `dsn`, `database?` | ✓ | ✗ |
| `mongo` | MongoDB | `uri`, `database`; table: `collection` | ✓ | ✗ |
| `redis` | Redis | `addr`/`url`, `db?`; table: `keyPrefix`, `type` | ✓ | ✗ |

See each driver source or [DEVELOPMENT.md](../.cursor/DEVELOPMENT.md) for credential fields.

---

## 4. File / object layer

Read-only or limited writes for static datasets and object files.

| protocol | Description | Main options | Phase 1 |
|----------|-------------|--------------|---------|
| `file` | Local CSV / JSON / NDJSON / YAML / XLSX | `rootPath`; table: `format` | **SELECT only** |
| `s3` | AWS S3 objects | `bucket`, `region?`; table: `prefix`, `format` | **SELECT only** |

Local **YAML / Excel** tabular files use the `file` driver with `format`. **Google Sheets / Airtable** are online APIs — use `sheets` / `airtable` presets (HTTP underneath).

**file driver mapping**

| Field | Role |
|-------|------|
| `schema` | Metadata namespace (`Accept-Profile` / `Content-Profile`); unrelated to disk layout |
| `name` | Logical table name; for xlsx, **equals sheet name** |
| `remoteName` | File path relative to `rootPath`; defaults to `name` |
| `options.format` | File format; inferred from extension if omitted |

Large files are read fully into memory — use `limit` in production.

---

## Selection guide

| Need | Recommendation |
|------|----------------|
| Any REST API | **`http`** |
| Notion / Firestore / Airtable / Sheets with presets | `notion` / `firebase` / `airtable` / `sheets` |
| Direct PG / MySQL / Mongo / Redis | matching native protocol |
| Local CSV / YAML / Excel / S3 static files | `file` / `s3` |
| New SaaS without a dedicated driver | **`http`** + credential + `remote_name` columns |

## Adding a driver

1. Implement `driver.Driver` in `internal/driver/<name>/`
2. `init()` → `driver.Register(models.ProtocolXxx, New)`
3. Blank import in `cmd/server/main.go`: `_ "github.com/monoposer/dataspan/internal/driver/<name>"`
4. For REST-only differences, **prefer `httpwrap`** instead of duplicating CRUD
5. Update `wrapper_server.protocol` CHECK in `scripts/migrations/init.up.sql`
6. Add tests (see `http`, `file`)

## Driver interface

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
