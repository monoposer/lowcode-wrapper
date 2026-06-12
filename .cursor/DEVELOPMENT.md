# Local development

Architecture, driver groups, and store modes: [docs/](../docs/README.md). This file covers **operations only**.

## Environment variables

```bash
cp .env.example .env          # DATASPAN_VAULT_KEY, DATABASE_URL
cp drivers.yaml.example drivers.yaml   # only when WRAPPER_STORE_MODE=file
```

| Variable | Description |
|----------|-------------|
| `WRAPPER_STORE_MODE` | `db` (default) or `file` (`postgres` is legacy alias for `db`) |
| `WRAPPER_DRIVERS_FILE` | file mode path, default `./drivers.yaml` |
| `DATABASE_URL` / `DATABASE_DSN` | db mode Meta DB DSN (postgres / mysql / sqlite) |
| `DATASPAN_VAULT_KEY` | credential encryption master key |

## Start

```bash
make postgres-up                # db mode with compose postgres service
make migrate                    # db mode, GORM AutoMigrate
make run                        # :3020
make test
```

File mode: `WRAPPER_STORE_MODE=file make run` (no postgres / migrate).

Driver tests: `go test ./internal/driver/http/... ./internal/driver/file/... -v`

## Admin API examples (db mode)

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
curl "$BASE/rest/v1/orders?id=eq.1&select=id,amount" -H 'Accept-Profile: public'
curl -X POST "$BASE/rest/v1/orders" -H 'Content-Type: application/json' -H 'Content-Profile: public' -d '{"id":"1","amount":"99"}'
curl -X PATCH "$BASE/rest/v1/orders?id=eq.1" -H 'Content-Type: application/json' -H 'Content-Profile: public' -d '{"amount":"100"}'
```

## Version

```bash
make version              # current VERSION file
make version-next         # same as VERSION (tag CI uses on merge to main)
make version-bump         # bump VERSION before merge (patch default)
make version-bump BUMP=minor
make version-set VER=1.0.0
./scripts/version.sh --help

Merging to `main` tags `v$(cat VERSION)` and triggers GitHub Release (convert CLI) + Docker publish. Bump `VERSION` in the PR when the tag does not exist yet.
```

## Migrations

- Schema: GORM models in `internal/models/entities.go`
- `make migrate` / `make migrate-down` (db store only; requires `DATABASE_URL` or `DATABASE_DSN`)
- Server does not auto-migrate on startup

## Common pitfalls

- Missing `DATASPAN_VAULT_KEY` → server won't start
- Unregistered table → `/v1/...` 404
- HTTP filters use **local** column names
- File driver loads whole files into memory → add `limit` in production
