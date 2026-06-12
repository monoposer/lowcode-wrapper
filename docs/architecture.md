# 架构

PostgREST 风格 **sidecar**（非 PG FDW 扩展）：元数据 + 凭据加密 + 统一 `/v1/...` 数据面。

## 请求流

```
Client → /v1/{schema}/{table} | /v1/rpc/{fn}
  → internal/api/postgrest.go
  → internal/postgrest/query.go
  → internal/service/engine.go
  → internal/store (ResolveTable)
  → internal/driver/*
```

## 元数据注册（postgres 模式 / Admin API）

```
POST /api/servers → POST /api/tables → （可选）POST /api/functions
```

file 模式：编辑 [`drivers.yaml`](../drivers.yaml.example)，见 [store.md](store.md)。

## 包地图

| 层 | 包 |
|----|-----|
| 入口 | `cmd/server`、`cmd/migrate` |
| HTTP | `internal/api` |
| 编排 | `internal/service` |
| Query | `internal/postgrest` |
| 元数据 | `internal/store` |
| 驱动 | `internal/driver` |
| 凭据 | `internal/auth` |

## 列名映射

API 用**本地**列名；Driver 出站用 `remote_name`。Filter / PATCH body 同样按本地名传入。
