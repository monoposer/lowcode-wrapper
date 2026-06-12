# dataspan — Agent entry

PostgREST-style sidecar: `/rest/v1/{table}` CRUD + `/rest/v1/rpc/{fn}` across multiple protocol drivers.

## Quick start

1. `cp .env.example .env` → set `DATASPAN_VAULT_KEY` (`openssl rand -base64 32`)
2. **db mode**: `make postgres-up` → `make migrate` → `make run`
3. **file mode**: `WRAPPER_STORE_MODE=file` + `cp drivers.yaml.example drivers.yaml` → `make run`
4. http://localhost:3020/swagger/（动态 OpenAPI）· `GET /health`

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

- New protocol → driver package + `cmd/server` import + `models.Protocol` constant
- Metadata schema → `internal/models/entities.go` + `make migrate`
- Dual store modes → `internal/store/config.go`, `internal/store/file/declarative.go`
