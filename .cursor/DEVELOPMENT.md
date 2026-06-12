# Local development

Architecture, driver groups, and store modes: [docs/](../docs/README.md). This file covers **operations only**.

## Environment variables

```bash
cp .env.example .env          # DATASPAN_VAULT_KEY, DATABASE_*
cp drivers.yaml.example drivers.yaml   # only when WRAPPER_STORE_MODE=file
```

| Variable | Description |
|----------|-------------|
| `WRAPPER_STORE_MODE` | `postgres` (default) or `file` |
| `WRAPPER_DRIVERS_FILE` | file mode path, default `./drivers.yaml` |
| `DATABASE_*` | postgres mode Meta DB (`.env` only) |
| `DATASPAN_VAULT_KEY` | credential encryption master key |

## Start

```bash
make postgres-up                # postgres mode (deploy/docker-compose.yml)
make migrate                    # postgres mode, first run or schema change
make run                        # :3020
make test
```

File mode: `WRAPPER_STORE_MODE=file make run` (no postgres / migrate).

Driver tests: `go test ./internal/driver/http/... ./internal/driver/file/... -v`

## Admin API examples (postgres mode)

```bash
BASE=http://localhost:3020

curl -s -X POST "$BASE/api/credentials" \
  -H 'Content-Type: application/json' \
  -d '{"name":"partner","data":{"apiKey":"secret"}}'

curl -s -X POST "$BASE/api/servers" \
  -H 'Content-Type: application/json' \
  -d '{"name":"partner_api","protocol":"http","credentialRef":"<uuid>","options":{"endpoint":"https://api.example.com","auth":{"type":"API_KEY","options":{"label":"X-Api-Key"}}}}'

curl -s -X POST "$BASE/api/tables" \
  -H 'Content-Type: application/json' \
  -d '{"serverName":"partner_api","tableName":"orders","remoteName":"orders","keyColumns":["id"],"columns":[{"name":"id"},{"name":"amount","remoteName":"total_amount"}]}'
```

File-mode equivalent: `drivers.yaml.example` (credentials inline on `servers`).

## PostgREST errors (data API)

`/v1/` and `/rest/v1/` return PostgREST-shaped errors for Supabase client compatibility:

```json
{
  "code": "PGRST205",
  "message": "Could not find the table 'public.orders' in the schema cache",
  "details": null,
  "hint": null
}
```

Admin API (`/api/*`) still uses `{"error":"..."}`.

## Supabase JS client

Point the client at dataspan with the `/rest/v1` path (now supported):

```javascript
import { createClient } from '@supabase/supabase-js'

const supabase = createClient('http://localhost:3020', 'any-anon-key', {
  db: { schema: 'public' },
})
```

Set `DATASPAN_ANON_KEY` (and optionally `DATASPAN_SERVICE_KEY`, `DATASPAN_JWT_SECRET`) to enable gateway auth on `/v1/` and `/rest/v1/`. Requests need `apikey` + `Authorization: Bearer <anon-key or HS256 JWT>`. No RLS — valid credentials access all registered tables. Unset `DATASPAN_ANON_KEY` for open local dev.

## Data API examples

```bash
curl "$BASE/v1/public/orders?id=eq.1&select=id,amount"
curl -X POST "$BASE/v1/public/orders" -H 'Content-Type: application/json' -d '{"id":"1","amount":"99"}'
curl -X PATCH "$BASE/v1/public/orders?id=eq.1" -H 'Content-Type: application/json' -d '{"amount":"100"}'
```

## Version

```bash
make version              # current VERSION file
make version-next         # next release (same logic as CI on merge)
make version-bump         # bump patch (default)
make version-bump BUMP=minor
make version-set VER=1.0.0
./scripts/version.sh --help
```

## Migrations

- DDL: `scripts/migrations/init.up.sql`
- `make migrate` / `make migrate-down` (postgres store only)
- Server does not auto-migrate on startup

## Common pitfalls

- Missing `DATASPAN_VAULT_KEY` → server won't start
- Unregistered table → `/v1/...` 404
- HTTP filters use **local** column names
- File driver loads whole files into memory → add `limit` in production
