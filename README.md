# lowcode-wrapper

PostgREST-style FDW sidecar: register foreign servers/tables in PostgreSQL metadata, expose `/v1/{schema}/{table}` CRUD and `/v1/rpc/{fn}` against remote HTTP, Postgres, MySQL, or file sources.

## Quick start

```bash
cp .env.example .env
# Set WRAPPER_MASTER_KEY: openssl rand -base64 32
docker compose up -d postgres
make run
```

## Admin API

- `POST /api/credentials` — store encrypted secrets
- `POST /api/servers` — register foreign server (protocol: `http`|`postgres`|`mysql`|`file`)
- `POST /api/tables` — register foreign table + columns
- `POST /api/functions` — register RPC mapping

## Data API (PostgREST-like)

- `GET /v1/{schema}/{table}?col=eq.val&select=a,b&limit=10`
- `POST /v1/{schema}/{table}`
- `PATCH /v1/{schema}/{table}?id=eq.1`
- `DELETE /v1/{schema}/{table}?id=eq.1`
- `POST /v1/rpc/{function}`

## 文档

- [AGENTS.md](AGENTS.md) — Cursor / Agent 开发说明
- [.cursor/DEVELOPMENT.md](.cursor/DEVELOPMENT.md) — 本地调试、API 示例、协议配置
