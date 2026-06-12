# Architecture

PostgREST-style **sidecar** (not a PostgreSQL FDW extension): metadata store, encrypted credentials, and a unified `/v1/...` data plane.

## Request flow

```
Client → /v1/{schema}/{table} | /v1/rpc/{fn}
  → internal/api/postgrest.go
  → internal/postgrest/query.go
  → internal/service/engine.go
  → internal/store (ResolveTable)
  → internal/driver/*
```

## Metadata registration (postgres mode / Admin API)

```
POST /api/servers → POST /api/tables → (optional) POST /api/functions
```

File mode: edit [`drivers.yaml`](../drivers.yaml.example); see [store.md](store.md).

## Package map

| Layer | Packages |
|-------|----------|
| Entry | `cmd/server`, `cmd/migrate`, `cmd/convert` |
| HTTP | `internal/api` |
| Orchestration | `internal/service` |
| Query | `internal/postgrest` |
| Metadata | `internal/store` |
| Drivers | `internal/driver` |
| Credentials | `internal/auth` |

## Column mapping

The API uses **local** column names; drivers send **remote** names via `remote_name`. Filters and PATCH bodies also use local names.
