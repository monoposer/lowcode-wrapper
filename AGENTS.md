# lowcode-wrapper — Agent 入口

PostgREST 风格 sidecar：`/v1/{schema}/{table}` CRUD + `/v1/rpc/{fn}`，多协议 driver。

## 快速启动

1. `cp .env.example .env` → 设置 `WRAPPER_MASTER_KEY`（`openssl rand -base64 32`）
2. **postgres 模式**：`docker compose up -d postgres` → `make migrate` → `make run`
3. **file 模式**：`WRAPPER_STORE_MODE=file` + `cp drivers.yaml.example drivers.yaml` → `make run`
4. http://localhost:3020/playground/ · `/swagger/` · `GET /health`

## 文档分工（勿重复展开）

| 文件 | 用途 |
|------|------|
| [`.cursor/rules/project.mdc`](.cursor/rules/project.mdc) | Agent 硬约定 + 链接（alwaysApply） |
| [`.cursor/DEVELOPMENT.md`](.cursor/DEVELOPMENT.md) | env、make、curl 示例 |
| [`docs/`](docs/README.md) | 架构、store、drivers、模块速查 |
| [`drivers.yaml.example`](drivers.yaml.example) | file 模式配置样例 |

需要细节时 `@docs/drivers.md` 或 `@docs/store.md`，不要从 rule 里复制长文。

## 改代码时常见触点

- 新 protocol → driver 包 + `cmd/server` import + `init.up.sql` CHECK
- 元数据 schema → `scripts/migrations/init.up.sql`
- store 双模式 → `internal/store/config.go`、`internal/store/file/declarative.go`
