# 内部模块速查

## `internal/api`

Admin + PostgREST 数据面 + Playground + Swagger。

| 路径 | 说明 |
|------|------|
| `/health` | 健康检查 |
| `/api/credentials` `/api/servers` `/api/tables` `/api/functions` | Admin CRUD |
| `/v1/{schema}/{table}` | GET/POST/PATCH/DELETE |
| `/v1/rpc/{name}` | RPC |
| `/playground/` `/swagger/` | UI |

主要文件：`admin.go`、`postgrest.go`、`logging.go`。curl 示例见 [.cursor/DEVELOPMENT.md](../.cursor/DEVELOPMENT.md)。

## `internal/auth`

- `WRAPPER_MASTER_KEY` → AES-GCM Vault
- Admin 写入加密 payload；出站解密；Admin **不**返回明文
- HTTP auth（`httpauth.go`）：`NONE` / `BASIC` / `API_KEY` / `BEARER_TOKEN` / `CLIENT_CREDENTIALS` / `UNIVERSAL`

## `internal/service`

Engine：`ResolveTable` → `driver.New` → CRUD / RPC。依赖 `store.Store` 接口。

## `internal/postgrest`

解析 `id=eq.1`、`select`、`order`、`limit`；`MapFilters` / `MapRowToRemote` 做列名映射。

## `internal/migrate` + `scripts/migrations`

仅 **postgres store 模式**。DDL：`init.up.sql` / `init.down.sql`。`make migrate`；启动不自动 migrate。

## `cmd/`

| 入口 | 说明 |
|------|------|
| `cmd/server` | Vault → `store.NewFromEnv` → HTTP handlers |
| `cmd/migrate` | `up` / `down`，需 `DATABASE_URL` |

Driver 在 `cmd/server/main.go` blank import 注册。
