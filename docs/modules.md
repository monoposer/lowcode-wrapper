# Internal modules

## `internal/api`

Admin + PostgREST data plane + Swagger.

| Path | Description |
|------|-------------|
| `/health` | Health check (includes `version`) |
| `/api/credentials` `/api/servers` `/api/tables` `/api/columns` `/api/functions` | Admin CRUD |
| `/` `/rest/v1/` | Data API OpenAPI (Swagger 2.0, dynamic) |
| `/rest/v1/{table}` | GET/POST/PATCH/DELETE (schema via profile headers) |
| `/rest/v1/rpc/{name}` | RPC |
| `/swagger/` | Swagger UI（动态 Data API OpenAPI） |

Metadata introspection: Admin API only (`/api/servers`, `/api/tables`, `/api/columns`, `/api/functions`). Curl examples: [.cursor/DEVELOPMENT.md](../.cursor/DEVELOPMENT.md).

## `internal/auth`

- `DATASPAN_VAULT_KEY` → AES-GCM vault
- Admin writes encrypted payloads; outbound requests decrypt; Admin **never** returns plaintext
- HTTP auth (`httpauth.go`): `NONE` / `BASIC` / `API_KEY` / `BEARER_TOKEN` / `CLIENT_CREDENTIALS` / `UNIVERSAL`

## `internal/service`

Engine: `ResolveTable` → `driver.New` → CRUD / RPC. Depends on `store.Store`.

## `internal/postgrest`

Parses `id=eq.1`, `select`, `order`, `limit`; `MapFilters` / `MapRowToRemote` for column mapping.

## `internal/migrate` + `internal/models/entities.go`

**DB store mode only**. GORM `AutoMigrate` from entity models. Run `make migrate`; server startup does not auto-migrate.

## `cmd/`

| Entry | Description |
|-------|-------------|
| `cmd/server` | Vault → `store.NewFromEnv` → HTTP handlers |
| `cmd/migrate` | `up` / `down`, requires `DATABASE_URL` or `DATABASE_DSN` |
| `cmd/cli` | `import`, `generate types`（Admin API meta → TS types） |

Drivers are registered via blank imports in `cmd/server/main.go`.
