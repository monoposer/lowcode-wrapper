# lowcode-wrapper — Cursor / Agent 开发说明

PostgREST 风格 FDW Sidecar：元数据存 PG，凭据 AES 加密，对外 `/v1/{schema}/{table}` CRUD + `/v1/rpc/{fn}`，内部按 `http` / `postgres` / `mysql` / `file` 协议路由。

## 本地启动

1. `cp .env.example .env`
2. 生成主密钥：`openssl rand -base64 32` → 写入 `WRAPPER_MASTER_KEY`
3. `docker compose up -d postgres`（默认 `localhost:5433`）
4. `make run` → http://localhost:3020
5. 健康检查：`GET /health`

## 目录

| 路径 | 说明 |
|------|------|
| `cmd/server/` | HTTP 入口 |
| `internal/api/` | Admin API + PostgREST handlers |
| `internal/auth/` | AES-GCM 凭据 vault + HTTP auth |
| `internal/driver/` | 协议驱动（http / postgres / mysql / file） |
| `internal/postgrest/` | query 解析（filter / select / order） |
| `internal/service/` | Engine 编排（ResolveTable → Driver） |
| `internal/store/postgres/` | 元数据 store + 启动时 migrate |
| `scripts/migrations/` | SQL 参考（运行时 migrate 内联于 store） |

## 架构要点

- **Meta DB**（`DATABASE_URL`）：`wrapper_server`、`wrapper_table`、`wrapper_column`、`wrapper_function`、`wrapper_credential`
- **Foreign Server** ≈ FDW `CREATE SERVER`；**Foreign Table** ≈ `CREATE FOREIGN TABLE`
- 凭据仅存 ciphertext；Admin API **永不**返回明文 secret
- HTTP 驱动 auth：`NONE` / `BASIC` / `API_KEY` / `BEARER_TOKEN` / `CLIENT_CREDENTIALS` / `UNIVERSAL`（`internal/auth/httpauth.go`）
- file 驱动 Phase 1 仅 SELECT（csv / json / ndjson）

## 勿混淆

- 无 gRPC / protobuf；纯 `net/http` + JSON
- 非 PostgreSQL FDW 扩展，是 HTTP sidecar
- 数据 API 需先经 Admin API 注册 server / table / column
- 改元数据 schema：同步更新 `scripts/migrations/001_init.up.sql` 与 `internal/store/postgres/migrate.go` 内联 SQL

详见 [.cursor/DEVELOPMENT.md](.cursor/DEVELOPMENT.md)
