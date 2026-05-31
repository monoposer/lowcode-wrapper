# 本地开发（Cursor）

## 环境变量

```bash
# 元数据 PG（wrapper_server / wrapper_table / wrapper_credential 等）
DATABASE_URL=postgresql://wrapper:wrapper@localhost:5433/wrapper_meta?sslmode=disable

# AES-256-GCM 主密钥（32 bytes base64）
# 生成：openssl rand -base64 32
WRAPPER_MASTER_KEY=

PORT=3020
```

复制：`cp .env.example .env` 并填入 `WRAPPER_MASTER_KEY`。

## 启动与调试

```bash
docker compose up -d postgres   # 仅 Meta PG，5433 → wrapper_meta
# 或整栈（含 migrate + wrapper 镜像）：
# docker compose up -d --build
make migrate                    # 仅 postgres 服务时，首次或 schema 变更后
make run                        # 或 make build && ./bin/server
# Playground: http://localhost:3020/playground/

# 最小镜像：make docker-build
# Dockerfile 多阶段：golang:alpine 编译 → distroless/static-debian12:nonroot
make test
make check                      # 仅编译检查
```

集成测试（需 Meta PG）：

```bash
export DATABASE_URL='postgresql://wrapper:wrapper@localhost:5433/wrapper_meta?sslmode=disable'
export WRAPPER_MASTER_KEY="$(openssl rand -base64 32)"
make migrate
go test ./internal/integration/... -v
```

## 请求流

```
Client
  → /v1/{schema}/{table}  或  /v1/rpc/{fn}
  → internal/api/postgrest.go
  → internal/service/engine.go
  → store.ResolveTable / ResolveFunction
  → driver.New(protocol) → Select / Insert / Update / Delete / Invoke
  → 远程 HTTP / Postgres / MySQL / 本地文件
```

Admin 注册流（必须先于数据 API）：

```
POST /api/credentials   → wrapper_credential（加密 payload）
POST /api/servers       → wrapper_server（protocol + options + credentialRef）
POST /api/tables        → wrapper_table + wrapper_column
POST /api/functions     → wrapper_function（RPC 映射，可选）
```

## Admin API 示例

```bash
BASE=http://localhost:3020

# 1. 凭据
curl -s -X POST "$BASE/api/credentials" \
  -H 'Content-Type: application/json' \
  -d '{"name":"partner","data":{"username":"u","password":"p"}}'

# 2. Foreign Server（HTTP）
curl -s -X POST "$BASE/api/servers" \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"partner_api",
    "protocol":"http",
    "credentialRef":"<uuid>",
    "options":{"endpoint":"https://api.example.com","auth":{"type":"BASIC"}}
  }'

# 3. Foreign Table
curl -s -X POST "$BASE/api/tables" \
  -H 'Content-Type: application/json' \
  -d '{
    "serverName":"partner_api",
    "schemaName":"public",
    "tableName":"orders",
    "remoteName":"orders",
    "keyColumns":["id"],
    "columns":[
      {"name":"id","dataType":"text"},
      {"name":"amount","dataType":"text","remoteName":"total_amount"}
    ]
  }'

# 4. RPC
curl -s -X POST "$BASE/api/functions" \
  -H 'Content-Type: application/json' \
  -d '{
    "serverName":"partner_api",
    "name":"create_order",
    "operation":"invoke",
    "remotePath":"/v2/orders/create",
    "method":"POST"
  }'
```

## PostgREST 数据 API 示例

```bash
# SELECT
curl "$BASE/v1/public/orders?id=eq.1&select=id,amount&limit=10"

# INSERT
curl -X POST "$BASE/v1/public/orders" \
  -H 'Content-Type: application/json' \
  -H 'Prefer: return=representation' \
  -d '{"id":"2","amount":"99"}'

# UPDATE
curl -X PATCH "$BASE/v1/public/orders?id=eq.2" \
  -H 'Content-Type: application/json' \
  -d '{"amount":"100"}'

# RPC
curl -X POST "$BASE/v1/rpc/create_order" \
  -H 'Content-Type: application/json' \
  -d '{"sku":"ABC","qty":1}'
```

## 协议驱动配置

| protocol | server.options | credential.data |
|----------|----------------|-----------------|
| `http` | `endpoint`, `basePath?`, `headers?`, `auth.type` + `auth.options` | username/password/token 等 |
| http table.options | `headers?`（与 server 合并，table 同名 key 覆盖） | — |
| http function.options | `headers?`（Invoke / RPC） | — |
| `postgres` | `dsn?`, `schema?` | `dsn` 或连接字段 |
| `mysql` | `dsn?`, `database?` | `dsn` 或 username/password/host/database |
| `file` | `rootPath` | 通常不需要 |
| table.options | file: `format` = csv/json/ndjson | — |

## Migration

- SQL 唯一来源：[`scripts/migrations/`](../scripts/migrations/)（`NNN_*.up.sql` / `NNN_*.down.sql`）
- 应用：`make migrate` 或 `go run ./cmd/migrate up`（需 `DATABASE_URL`）
- 回滚：`make migrate-down` 或 `go run ./cmd/migrate down`
- 服务启动**不会**自动 migrate

## 常见陷阱

- 未设置 `WRAPPER_MASTER_KEY` 时服务无法启动
- 未注册 table 就访问 `/v1/...` 会 404
- HTTP 驱动 filter 列名用**本地** column name，内部会映射到 `remote_name`
- file 驱动大文件全量读入内存，生产应加 `limit`
- `.env` 含密钥，已在 `.gitignore` 中排除
