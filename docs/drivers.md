# 协议驱动（Driver）

Driver 是 wrapper 与**远程数据源**之间的适配层。Admin API 注册 `wrapper_server.protocol` 后，Engine 通过 `driver.New(protocol)` 实例化对应实现。

注册入口：`internal/driver/registry.go`（各驱动在 `init()` 中 `driver.Register`）。

## 元数据从哪来（Store 模式）

Driver **本身**在代码里注册；Foreign Server / Table **配置**由 Store 模式决定读取位置：

| `WRAPPER_STORE_MODE` | 配置来源 | 典型用途 |
|----------------------|----------|----------|
| `postgres`（默认） | Meta DB + Admin API | 生产、动态注册 |
| `file` | [`drivers.yaml`](../drivers.yaml.example) 声明式 YAML | 本地开发、GitOps 式配置 |

```bash
WRAPPER_STORE_MODE=file
WRAPPER_DRIVERS_FILE=./drivers.yaml
cp drivers.yaml.example drivers.yaml
```

YAML 中按 `servers[].protocol` 选择 driver。配置形态见 [store.md](store.md) 与 [`drivers.yaml.example`](../drivers.yaml.example)。

## 分组概览

```
┌─────────────────────────────────────────────────────────────┐
│  HTTP 通用层                                                 │
│  http — 任意 REST / PostgREST 风格上游                       │
│  （自定义服务、第三方 SaaS、内部微服务）                      │
└─────────────────────────────────────────────────────────────┘
         ▲ 委托
┌────────┴────────┐
│  HTTP 预设包装   │  notion、firebase（endpoint/auth/header 默认值）
└─────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  原生协议层                                                  │
│  postgres · mysql · mongo · redis                           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│  文件 / 对象层                                               │
│  file（本地目录）· s3（对象存储）                             │
└─────────────────────────────────────────────────────────────┘
```

## 1. HTTP 通用层 — `http`

**适用场景**

- 自建 REST API、内部微服务
- 任意第三方 HTTP API（Stripe、GitHub、企业 ERP 等）
- 上游已是 PostgREST 或 PostgREST 兼容接口

**为何优先用 `http`**

不必为每个 SaaS 写专用 driver。配置 `endpoint`、`auth`、`headers`、`remote_name` 列映射即可接入。

```json
{
  "name": "partner_api",
  "protocol": "http",
  ...
}
```

file 模式等价 YAML 见 `drivers.yaml.example` 中 `partner_api` 示例。

**能力**：Select / Insert / Update / Delete / Invoke（RPC）

**认证类型**（`options.auth.type`）：`NONE` · `BASIC` · `API_KEY` · `BEARER_TOKEN` · `CLIENT_CREDENTIALS` · `UNIVERSAL`

**实现**：`internal/driver/http/`

**测试**：`go test ./internal/driver/http/... -v`

---

## 2. HTTP 预设包装 — 委托 `httpwrap`

对常见 SaaS 预填 endpoint、header、Bearer 规则，底层仍走 `http`。

| protocol | 默认 endpoint | 凭据字段 | 代码 |
|----------|---------------|----------|------|
| `notion` | `https://api.notion.com/v1` | `token` / `integrationToken` | `notion/` |
| `firebase` | Firestore REST（由 `projectId` 推导） | `accessToken` / `token` | `firebase/` |
| `airtable` | `https://api.airtable.com/v0` | `token` / `personalAccessToken` | `airtable/` |
| `sheets` | `https://sheets.googleapis.com/v4` | `accessToken` / `token` | `sheets/` |

包装逻辑：`internal/driver/httpwrap/wrap.go` → `NewHTTPDriver()` 合并 defaults 后调用 `http.New()`。

若预设与实际上游不符，可直接改用 **`protocol: http`** 并手动配 endpoint。

---

## 3. 原生协议层

直连数据库或 KV，不经 HTTP 中转。

| protocol | 说明 | 主要 options | CRUD | RPC |
|----------|------|--------------|------|-----|
| `postgres` | PostgreSQL | `dsn`, `schema?` | ✓ | ✗ |
| `mysql` | MySQL | `dsn`, `database?` | ✓ | ✗ |
| `mongo` | MongoDB | `uri`, `database`；table: `collection` | ✓ | ✗ |
| `redis` | Redis | `addr`/`url`, `db?`；table: `keyPrefix`, `type` | ✓ | ✗ |

凭据字段见各 driver 源码或 [DEVELOPMENT.md](../.cursor/DEVELOPMENT.md)（postgres Admin 流程）。

---

## 4. 文件 / 对象层

只读或有限写入，适合静态数据集、对象文件。

| protocol | 说明 | 主要 options | Phase 1 能力 |
|----------|------|--------------|--------------|
| `file` | 本地目录 CSV / JSON / NDJSON / YAML / XLSX | `rootPath`；table: `format` | **仅 SELECT** |
| `s3` | AWS S3 对象 | `bucket`, `region?`；table: `prefix`, `format` | **仅 SELECT** |

本地 **YAML / Excel** 与 CSV 同属 tabular 只读场景，放在 `file` 驱动下通过 `format` 区分即可；**Google Sheets / Airtable** 是在线 API，用 `sheets` / `airtable` 预设（底层 `http`）。

**file 驱动映射约定**

| 字段 | 作用 |
|------|------|
| `schema` | 仅 API 路径 `/v1/{schema}/{table}`，与磁盘无关 |
| `name` | 逻辑表名；xlsx 时**等于 sheet 名** |
| `remoteName` | 磁盘文件路径（相对 `rootPath`）；省略时用 `name` |
| `options.format` | 文件格式；省略时按扩展名推断 |

大文件会全量读入内存，生产环境请加 `limit`。

---

## 选型建议

| 需求 | 推荐 |
|------|------|
| 对接任意 REST API | **`http`** |
| 对接 Notion / Firestore / Airtable / Google Sheets 且接受预设 | `notion` / `firebase` / `airtable` / `sheets` |
| 直连 PG / MySQL / Mongo / Redis | 对应原生 protocol |
| 读本地 CSV / YAML / Excel / S3 静态文件 | `file` / `s3` |
| 新 SaaS，无专用 driver | **`http`** + credential + column `remote_name` |

## 扩展新驱动

1. 在 `internal/driver/<name>/` 实现 `driver.Driver` 接口
2. `init()` 中 `driver.Register(models.ProtocolXxx, New)`
3. 在 `cmd/server/main.go` 空白导入 `_ "lowcode-wrapper/internal/driver/<name>"`
4. 若只是 REST 差异，**优先包装 `httpwrap`**，不必复制 CRUD 逻辑
5. 更新 `scripts/migrations/init.up.sql` 中 `wrapper_server.protocol` CHECK 约束
6. 在对应包下添加单测（参考 `http`、`file`）

## Driver 接口

```go
type Driver interface {
    Select(ctx, SelectRequest) ([]map[string]any, error)
    Insert(ctx, RowRequest) (map[string]any, error)
    Update(ctx, RowRequest) (int, error)
    Upsert(ctx, RowRequest) (bool, map[string]any, error)
    Delete(ctx, DeleteRequest) (int, error)
    Invoke(ctx, InvokeRequest) (any, error)
}
```

定义见 `internal/driver/driver.go`。
