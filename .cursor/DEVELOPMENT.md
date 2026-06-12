# 本地开发

架构、driver 分组、store 模式见 [docs/](../docs/README.md)。本文只保留**操作步骤**。

## 环境变量

```bash
cp .env.example .env          # WRAPPER_MASTER_KEY、DATABASE_*
cp drivers.yaml.example drivers.yaml   # 仅 WRAPPER_STORE_MODE=file 时需要
```

| 变量 | 说明 |
|------|------|
| `WRAPPER_STORE_MODE` | `postgres`（默认）或 `file` |
| `WRAPPER_DRIVERS_FILE` | file 模式，默认 `./drivers.yaml` |
| `DATABASE_*` | postgres 模式 Meta DB（仅 `.env`） |
| `WRAPPER_MASTER_KEY` | 凭据加密主密钥 |

## 启动

```bash
docker compose up -d postgres   # postgres 模式
make migrate                    # postgres 模式，首次或 schema 变更
make run                        # :3020
make test
```

file 模式：`WRAPPER_STORE_MODE=file make run`（无需 postgres / migrate）。

Driver 单测：`go test ./internal/driver/http/... ./internal/driver/file/... -v`

## Admin API 示例（postgres 模式）

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

file 模式等价配置见 `drivers.yaml.example`（凭据 inline 在 `servers`）。

## 数据 API 示例

```bash
curl "$BASE/v1/public/orders?id=eq.1&select=id,amount"
curl -X POST "$BASE/v1/public/orders" -H 'Content-Type: application/json' -d '{"id":"1","amount":"99"}'
curl -X PATCH "$BASE/v1/public/orders?id=eq.1" -H 'Content-Type: application/json' -d '{"amount":"100"}'
```

## 迁移

- DDL：`scripts/migrations/init.up.sql`
- `make migrate` / `make migrate-down`（仅 postgres store）
- 服务启动不自动 migrate

## 常见陷阱

- 未设 `WRAPPER_MASTER_KEY` → 无法启动
- 未注册 table → `/v1/...` 404
- HTTP filter 用**本地**列名
- file driver 大文件全量读内存 → 生产加 `limit`
