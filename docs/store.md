# Meta store

`WRAPPER_STORE_MODE` controls **where Foreign Server / Table metadata is read from**. Meta DB connection settings live only in `.env` (`DATABASE_*`), not in YAML.

| Mode | Metadata source | Configuration |
|------|-----------------|---------------|
| `postgres` (default) | PostgreSQL | Admin API + `make migrate` |
| `file` | `drivers.yaml` | Declarative YAML (credentials inline on `servers`) |

```bash
WRAPPER_STORE_MODE=postgres|file
WRAPPER_DRIVERS_FILE=./drivers.yaml   # file mode, default ./drivers.yaml
```

Postgres mode: `DATABASE_HOST` / `DATABASE_PASSWORD` in `.env`, or `DATABASE_URL`.

## drivers.yaml (file mode)

On disk the shape differs from postgres tables; after `compileDeclarative` the in-memory model matches the postgres store (`Server` + `CredentialRef` + encrypted payload).

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
| postgres | `wrapper_credential` + `wrapper_server` + … | `models.Server` + `CredentialRef` |
| file | `servers` + `tables` + `functions` | same as postgres |

Use top-level `credentials` + `credential: name` only when multiple servers share one secret (see [`drivers.yaml.example`](../drivers.yaml.example)).

## Security

| Secret | Location |
|--------|----------|
| Meta DB password | `.env` `DATABASE_*` |
| Foreign credentials | `servers[].credential` or Admin API; prefer `${ENV_VAR}` |
| `WRAPPER_MASTER_KEY` | `.env` only |

Implementation: `internal/store/postgres/`, `internal/store/file/`.
