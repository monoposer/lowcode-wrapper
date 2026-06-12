# lowcode-wrapper

PostgREST-style FDW sidecar: register foreign servers/tables in PostgreSQL metadata, expose `/v1/{schema}/{table}` CRUD and `/v1/rpc/{fn}` against remote HTTP, Postgres, MySQL, or file sources.

## Quick start

```bash
cp .env.example .env
# Set WRAPPER_MASTER_KEY: openssl rand -base64 32
docker compose up -d postgres
make migrate
make run
```

Playground UI: http://localhost:3020/playground/

API docs (OpenAPI + Swagger UI): http://localhost:3020/swagger/ — spec at `/openapi/openapi.yaml`

## Docker

```bash
# Build minimal image (~15–20MB, distroless + static binary)
make docker-build IMAGE=lowcode-wrapper:latest

# Full stack (postgres + migrate + server)
cp .env.example .env   # set WRAPPER_MASTER_KEY
docker compose up -d --build
```

Migrations run once via the `migrate` service; the server image only contains `/server`.

## Admin API

- `POST /api/credentials` — store encrypted secrets
- `POST /api/servers` — register foreign server (`http`|`postgres`|`mysql`|`file`|`mongo`|`redis`|`s3`|`notion`|`firebase`)
- `POST /api/tables` — register foreign table + columns
- `POST /api/functions` — register RPC mapping

## Data API (PostgREST-like)

- `GET /v1/{schema}/{table}?col=eq.val&select=a,b&limit=10`
- `POST /v1/{schema}/{table}`
- `PATCH /v1/{schema}/{table}?id=eq.1`
- `DELETE /v1/{schema}/{table}?id=eq.1`
- `POST /v1/rpc/{function}`

## 文档

| 文档 | 说明 |
|------|------|
| [docs/](docs/README.md) | 架构、store、drivers、模块速查 |
| [.cursor/DEVELOPMENT.md](.cursor/DEVELOPMENT.md) | 本地 env、curl、make |
| [AGENTS.md](AGENTS.md) | Cursor Agent 入口 |

## License

[MIT](LICENSE)
