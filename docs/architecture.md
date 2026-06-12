# Architecture

PostgREST-style **sidecar** (not a PostgreSQL FDW extension): metadata store, encrypted credentials, and a unified `/v1/...` data plane.

**Full architecture (Chinese)**: [技术架构.md](技术架构.md)

## Request flow

```
Client → /rest/v1/{table} (Accept-Profile / Content-Profile) | /rest/v1/rpc/{fn}
  → internal/api/postgrest.go
  → internal/postgrest/query.go
  → internal/service/engine.go
  → internal/store (ResolveTable)
  → internal/driver/*
```

Schema introspection: Admin API (`/api/servers`, `/api/tables`, …). OpenAPI: `GET /` or `/rest/v1/` → `internal/postgrest/openapi.go`.

## Metadata registration (db mode / Admin API)

```
POST /api/servers → POST /api/tables → (optional) POST /api/functions
```

File mode: edit [`drivers.yaml`](../drivers.yaml.example); see [store.md](store.md).

## Package map

| Layer | Packages |
|-------|----------|
| Entry | `cmd/server`, `cmd/migrate`, `cmd/cli` |
| HTTP | `internal/api` |
| Orchestration | `internal/service` |
| Query | `internal/postgrest` |
| Metadata | `internal/store` |
| Drivers | `internal/driver` |
| Credentials | `internal/auth` |

## Column mapping

The API uses **local** column names; drivers send **remote** names via `remote_name`. Filters and PATCH bodies also use local names.
