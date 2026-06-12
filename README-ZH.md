# dataspan

**用一套 PostgREST 兼容的 REST 接口，访问多种异构数据源。**

注册 Foreign Server / Table（PostgreSQL 元数据库或 `drivers.yaml`）后，前端即可用熟悉的 PostgREST 语法查询远程 HTTP、数据库、对象存储或本地文件 — 过滤器（`id=eq.1`）、`select`、`order`、`limit`、RPC，以及标准错误码。

```
客户端  →  /v1/{schema}/{table}  →  driver  →  远程数据源
```

## 快速启动

```bash
cp .env.example .env          # DATASPAN_VAULT_KEY=$(openssl rand -base64 32)
make postgres-up && make migrate && make run
```

- Swagger：http://localhost:3020/swagger/

Docker 全栈部署见 [deploy/](deploy/README.md)（`make compose-up`）。

## 数据 API

| 方法 | 路径 | 示例 |
|------|------|------|
| GET | `/v1/{schema}/{table}` | `?id=eq.1&select=id,name&limit=10` |
| POST | `/v1/{schema}/{table}` | 插入行 |
| PATCH | `/v1/{schema}/{table}` | `?id=eq.1` + JSON body |
| DELETE | `/v1/{schema}/{table}` | `?id=eq.1` |
| POST | `/v1/rpc/{function}` | 远程过程调用 |

`/rest/v1/` 为路径别名，兼容 [Supabase JS](https://supabase.com/docs/reference/javascript) 等 PostgREST 客户端。

错误体与 PostgREST 一致（`code`、`message`、`details`、`hint`），例如未注册表时返回 `PGRST205`。

设置 `DATASPAN_ANON_KEY` 后，数据面要求 `apikey` + `Authorization: Bearer`（anon key 或 `DATASPAN_JWT_SECRET` 签发的 JWT）。不设则本地开放访问。不做 RLS。

## Admin API

查询数据前需先注册元数据：

- `POST /api/credentials` — 加密存储凭据
- `POST /api/servers` — Foreign Server + 协议
- `POST /api/tables` — 表与列映射（`name` → `remote_name`）
- `POST /api/functions` — RPC 映射

协议与选型见 [docs/drivers.md](docs/drivers.md)。

## 更多文档

| | |
|-|-|
| [docs/](docs/README.md) | 架构、store 模式、drivers |
| [deploy/](deploy/README.md) | Docker 镜像与 compose |