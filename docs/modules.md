# Internal modules

## `internal/api`

HTTP entry: auth middleware, Admin plane, PostgREST data plane.

| Package / file | Description |
|----------------|-------------|
| `api/auth.go` | `AdminAuth` / `DataAuth` middleware (`DATASPAN_ADMIN_KEY`, `DATASPAN_ANON_KEY`) |
| `api/admin/` | Admin handler — `/admin/api/*`, `/health`, import |
| `api/rest/` | Data handler — `/rest/v1/*`, `/v1/*`, OpenAPI root |

| Path | Description |
|------|-------------|
| `/health` | Liveness |
| `/health/ready` | Readiness (meta store ping) |
| `/admin/api/credentials` `/admin/api/servers` `/admin/api/tables` `/admin/api/columns` `/admin/api/functions` | Admin CRUD (db mode only) |
| `/` `/rest/v1/` | Data API OpenAPI 3.1.0 (dynamic) |
| `/rest/v1/{table}` | GET/POST/PATCH/DELETE (schema via profile headers) |
| `/rest/v1/rpc/{name}` | RPC |
| Data API OpenAPI | `GET /` · `GET /rest/v1/` · `GET /v1/` (`application/openapi+json`) |

Metadata introspection: Admin API (`/admin/api/servers`, …). **db mode**: full CRUD; **file mode**: GET only (writes return 405). Curl examples: [.cursor/DEVELOPMENT.md](../.cursor/DEVELOPMENT.md).

## `internal/httpx`

Shared HTTP helpers: `WriteJSON`, `WriteAdminError`, `DecodeJSON`, `CORS`, `Logging`.

## `internal/auth`

- `DATASPAN_VAULT_KEY` → AES-GCM vault
- Admin writes encrypted payloads; outbound requests decrypt; Admin **never** returns plaintext
- HTTP auth (`httpauth.go`): `NONE` / `BASIC` / `API_KEY` / `BEARER_TOKEN` / `CLIENT_CREDENTIALS` / `UNIVERSAL`

## `internal/engine`

Engine: `ResolveTable` → `DriverFor`（池化 `driver.New`）→ CRUD / RPC. Optional metadata cache (`DATASPAN_META_CACHE_TTL`). Depends on `store.Store`.

| File | Description |
|------|-------------|
| `engine.go` | `Engine` orchestration |
| `driver_pool.go` | Driver instance cache by server ID + config fingerprint |
| `meta_cache.go` | Short-TTL resolve cache + revision invalidation |

## `internal/postgrest`

Parses `id=eq.1`, `select`, `order`, `limit`; `MapFilters` / `MapRowToRemote` for column mapping; OpenAPI 3.1.0 generation.

## `internal/migrate` + `internal/models/entities.go`

**DB store mode only**. GORM `AutoMigrate` on server start (`internal/store/db`, skippable via `DATASPAN_AUTOMIGRATE=0`). `internal/migrate` retains `Up`/`Down` helpers for tests.

## `cmd/`

| Entry | Description |
|-------|-------------|
| `cmd/server` | Vault → `store.NewFromEnv` → `engine.NewEngine` → `admin.New` + `rest.New` → HTTP |
| `cmd/cli` | `import`, `generate types`（Admin API meta → TS types） |

Drivers are registered via blank imports in `cmd/server/main.go`.
