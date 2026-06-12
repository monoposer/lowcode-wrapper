# Meta Store

`WRAPPER_STORE_MODE` 决定 **Foreign Server / Table 元数据从哪读**。Meta DB 连接只在 `.env`（`DATABASE_*`），不在 YAML 里配库。

| 模式 | 元数据来源 | 配置方式 |
|------|------------|----------|
| `postgres`（默认） | PostgreSQL | Admin API + `make migrate` |
| `file` | `drivers.yaml` | 声明式 YAML（凭据 inline 在 `servers`） |

```bash
WRAPPER_STORE_MODE=postgres|file
WRAPPER_DRIVERS_FILE=./drivers.yaml   # file 模式，默认 ./drivers.yaml
```

Postgres 模式：`.env` 中 `DATABASE_HOST` / `DATABASE_PASSWORD` 等，或 `DATABASE_URL`。

## drivers.yaml（file 模式）

磁盘形态与 postgres **拆表不同**；`compileDeclarative` 后内存与 postgres store **相同**（`Server` + `CredentialRef` + 加密 payload）。

```yaml
servers:
  - name: partner_api
    protocol: http
    credential:
      apiKey: ${PARTNER_API_KEY}
    options:
      endpoint: https://api.example.com

tables:
  - server: partner_api
    name: orders
    columns: [...]
```

| 持久化 | 磁盘 | 内存 |
|--------|------|------|
| postgres | `wrapper_credential` + `wrapper_server` + … | `models.Server` + `CredentialRef` |
| file | `servers` + `tables` + `functions` | 同上 |

多 server 共享凭据时，才用顶层 `credentials` + `credential: 名称`（见 [`drivers.yaml.example`](../drivers.yaml.example)）。

## 安全

| 内容 | 放哪 |
|------|------|
| Meta DB 密码 | `.env` `DATABASE_*` |
| Foreign 凭据 | `servers[].credential` 或 Admin API；优先 `${ENV_VAR}` |
| `WRAPPER_MASTER_KEY` | 仅 `.env` |

实现：`internal/store/postgres/`、`internal/store/file/`。
