# Meta store

`DATASPAN_STORE_MODE` controls **where Foreign Server / Table metadata is read from**. Meta DB connection is **DSN only** in `.env` (`DATABASE_URL` or `DATABASE_DSN`), not in YAML.

| Mode | Metadata source | Admin API | Typical use |
|------|-----------------|-----------|-------------|
| `db` (default) | SQL database (postgres / mysql / sqlite) | Full CRUD + import | Production, dynamic registration |
| `file` | `drivers.yaml` | **Read-only GET** (writes → 405) | Local dev, GitOps |

After load, both modes expose the **same in-memory model** (`models.Server` / `Table` / `Column` / `Function`); `Engine` and Data API behave identically.

## Environment

```bash
DATASPAN_STORE_MODE=db|file          # default: db
DATASPAN_DRIVERS_FILE=./drivers.yaml # file mode path (default ./drivers.yaml)
```

### DB mode DSN examples

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/dataspan_meta?sslmode=disable
# DATABASE_DSN=mysql://user:pass@tcp(localhost:3306)/dataspan_meta
# DATABASE_DSN=file:./dataspan_meta.db
```

Schema is applied on **server startup** via GORM AutoMigrate (`internal/store/db/migrate.go`).

## Metadata workflows

**db mode**

```
POST /admin/api/servers (optional inline credential)
  → POST /admin/api/tables
  → (optional) POST /admin/api/functions
```

Admin writes take effect immediately; Engine reads Store on each request.

**file mode**

```
Edit drivers.yaml → restart server
```

- **Admin GET** works (`/admin/api/servers`, `/admin/api/tables`, …) for introspection and `dataspan generate types`.
- **Admin writes** (POST / PATCH / DELETE / import) return **405** with a hint to edit `drivers.yaml`.
- YAML changes require a **process restart** to reload.

See [技术架构.md](技术架构.md) §5.1 for the full Admin endpoint matrix.

## Meta DB tables (db mode)

GORM models in `internal/models/entities.go`.

| Entity | Table |
|--------|-------|
| `MetaCredential` | `credentials` |
| `MetaServer` | `servers` |
| `MetaForeignTable` | `tables` |
| `MetaForeignColumn` | `columns` |
| `MetaForeignFunction` | `functions` |

## drivers.yaml (file mode)

On disk the shape differs from DB rows; after `compileDeclarative` the in-memory model matches db store (`Server` + `CredentialRef` + encrypted payload).

```yaml
servers:
  - name: partner_api
    protocol: http
    credential:
      apiKey: ${PARTNER_API_KEY}
    options:
      endpoint: https://api.example.com

tables:
  - server: partner_api
    name: orders
    columns: [...]
```

| Persistence | On disk | In memory |
|-------------|---------|-----------|
| db | `credentials` + `servers` + `tables` + `columns` + `functions` | `models.Server` + `CredentialRef` |
| file | `servers` + `tables` + `functions` | same as db |

Use top-level `credentials` + `credential: name` only when multiple servers share one secret (see [`drivers.yaml.example`](../drivers.yaml.example)).

## Security

| Secret | Location |
|--------|----------|
| Meta DB DSN | `.env` `DATABASE_URL` / `DATABASE_DSN` (db mode only) |
| Foreign credentials | db: Admin API (Vault-encrypted); file: `servers[].credential` in YAML |
| `DATASPAN_VAULT_KEY` | `.env` only (32-byte base64) |

Prefer `${ENV_VAR}` expansion in YAML; restrict file permissions on `drivers.yaml`.

Admin **never returns credential plaintext** in either mode.

## Implementation

| Package | Role |
|---------|------|
| `internal/store/config.go` | Mode selection, `Store` factory |
| `internal/store/db/` | GORM Meta DB |
| `internal/store/file/` | YAML load / optional persist |

Related: [architecture.md](architecture.md) · [技术架构.md](技术架构.md) · [modules.md](modules.md)
