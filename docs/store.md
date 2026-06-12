# Meta store

`WRAPPER_STORE_MODE` controls **where Foreign Server / Table metadata is read from**. Meta DB connection is **DSN only** in `.env` (`DATABASE_URL` or `DATABASE_DSN`), not in YAML.

| Mode | Metadata source | Configuration |
|------|-----------------|---------------|
| `db` (default) | SQL database (postgres / mysql / sqlite) | Admin API + `make migrate` (GORM AutoMigrate) |
| `file` | `drivers.yaml` | Declarative YAML (credentials inline on `servers`) |

```bash
WRAPPER_STORE_MODE=db|file
WRAPPER_DRIVERS_FILE=./drivers.yaml   # file mode, default ./drivers.yaml
```

Legacy alias: `WRAPPER_STORE_MODE=postgres` → `db`.

DB mode examples:

```bash
DATABASE_URL=postgresql://user:pass@localhost:5432/dataspan_meta?sslmode=disable
# DATABASE_DSN=mysql://user:pass@tcp(localhost:3306)/dataspan_meta
# DATABASE_DSN=file:./dataspan_meta.db
```

## Meta DB tables (db mode)

GORM models in `internal/models/entities.go` drive schema via `make migrate`:

| Entity | Table |
|--------|-------|
| `MetaCredential` | `credentials` |
| `MetaServer` | `servers` |
| `MetaForeignTable` | `foreign_tables` |
| `MetaForeignColumn` | `foreign_columns` |
| `MetaForeignFunction` | `foreign_functions` |

## drivers.yaml (file mode)

On disk the shape differs from DB tables; after `compileDeclarative` the in-memory model matches db store (`Server` + `CredentialRef` + encrypted payload).

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
| db | `credentials` + `servers` + `foreign_*` | `models.Server` + `CredentialRef` |
| file | `servers` + `tables` + `functions` | same as db |

Use top-level `credentials` + `credential: name` only when multiple servers share one secret (see [`drivers.yaml.example`](../drivers.yaml.example)).

## Security

| Secret | Location |
|--------|----------|
| Meta DB DSN | `.env` `DATABASE_URL` / `DATABASE_DSN` |
| Foreign credentials | `servers[].credential` or Admin API; prefer `${ENV_VAR}` |
| `DATASPAN_VAULT_KEY` | `.env` only |

Implementation: `internal/store/db/` (GORM), `internal/store/file/`.
