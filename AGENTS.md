# dataspan — Agent entry

PostgREST-style sidecar: `/v1/{schema}/{table}` CRUD + `/v1/rpc/{fn}` across multiple protocol drivers.

## Quick start

1. `cp .env.example .env` → set `WRAPPER_MASTER_KEY` (`openssl rand -base64 32`)
2. **postgres mode**: `make postgres-up` → `make migrate` → `make run`
3. **file mode**: `WRAPPER_STORE_MODE=file` + `cp drivers.yaml.example drivers.yaml` → `make run`
4. http://localhost:3020/playground/ · `/swagger/` · `GET /health`

## Doc map (do not duplicate long content here)

| File | Purpose |
|------|---------|
| [`.cursor/rules/project.mdc`](.cursor/rules/project.mdc) | Agent constraints + links (alwaysApply) |
| [`.cursor/DEVELOPMENT.md`](.cursor/DEVELOPMENT.md) | env, make, curl examples |
| [`docs/`](docs/README.md) | Architecture, store, drivers, modules |
| [`deploy/`](deploy/README.md) | Production deployment |
| [`drivers.yaml.example`](drivers.yaml.example) | File-mode config sample |

For details, `@docs/drivers.md` or `@docs/store.md` — do not copy long sections into rules.

## Common touch points

- New protocol → driver package + `cmd/server` import + `init.up.sql` CHECK
- Metadata schema → `scripts/migrations/init.up.sql`
- Dual store modes → `internal/store/config.go`, `internal/store/file/declarative.go`
