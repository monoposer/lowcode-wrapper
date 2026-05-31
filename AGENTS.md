# lowcode-wrapper — Cursor / Agent 开发说明

PostgREST 风格 FDW Sidecar：元数据存 PG，凭据 AES 加密，对外 `/v1/{schema}/{table}` CRUD + `/v1/rpc/{fn}`，内部按协议路由（`http` / `postgres` / `mysql` / `file` / `mongo` / `redis` / `s3` / `notion` / `firebase`）。

## 本地启动

1. `cp .env.example .env`
2. 生成主密钥：`openssl rand -base64 32` → 写入 `WRAPPER_MASTER_KEY`
3. 本地 Meta PG：`docker compose up -d postgres`（`localhost:5433`），或整栈：`docker compose up -d --build`（需 `.env` 中 `WRAPPER_MASTER_KEY`）
4. 非 compose 时：`make migrate` 再 `make run`
5. `make run` 或 compose 的 `wrapper` → http://localhost:3020
6. 测试 UI：`http://localhost:3020/playground/`
7. 健康检查：`GET /health`

## 目录

| 路径 | 说明 |
|------|------|
| `cmd/server/` | HTTP 入口 |
| `cmd/migrate/` | Meta DB 迁移 CLI（`up` / `down`） |
| `internal/api/` | Admin API + PostgREST handlers |
| `internal/auth/` | AES-GCM 凭据 vault + HTTP auth |
| `internal/driver/` | 协议驱动（http / postgres / mysql / file） |
| `internal/postgrest/` | query 解析（filter / select / order） |
| `internal/service/` | Engine 编排（ResolveTable → Driver） |
| `internal/store/postgres/` | 元数据 store |
| `internal/migrate/` | 读取 `scripts/migrations` 并执行 |
| `scripts/migrations/` | Meta DB DDL（唯一来源） |

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
- 改元数据 schema：在 `scripts/migrations/` 增加新的 `NNN_*.up.sql` / `.down.sql`，再 `make migrate`

详见 [.cursor/DEVELOPMENT.md](.cursor/DEVELOPMENT.md)
