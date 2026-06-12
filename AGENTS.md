# dataspan — Agent entry

PostgREST-style sidecar: `/rest/v1/{table}` CRUD + `/rest/v1/rpc/{fn}` across multiple protocol drivers.

## Quick start

1. `cp .env.example .env` → set `DATASPAN_VAULT_KEY` (`openssl rand -base64 32`)
2. **db mode**: `make postgres-up` → `make run`（启动时 GORM AutoMigrate）
3. **file mode**: `DATASPAN_STORE_MODE=file` + `cp drivers.yaml.example drivers.yaml` → `make run`
4. http://localhost:3020/rest/v1/（OpenAPI JSON）· `GET /health`

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

- HTTP 入口 → `internal/api/auth.go`（中间件）、`internal/api/admin/`（管理面）、`internal/api/rest/`（数据面）、`internal/httpx/`（CORS/JSON）
- 编排 → `internal/engine/`（Engine、Driver 池、元数据缓存）
- New protocol → driver package + `cmd/server` import + `models.Protocol` constant
- Metadata schema → `internal/models/entities.go`（db 模式 server 启动 AutoMigrate，`DATASPAN_AUTOMIGRATE=0` 可跳过）
- Dual store modes → `internal/store/config.go`, `internal/store/file/declarative.go`
