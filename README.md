# dataspan

A lightweight sidecar that exposes heterogeneous data sources through one PostgREST-compatible REST surface.

Register foreign servers and tables (PostgreSQL metadata or declarative YAML), then query remote HTTP, Postgres, MySQL, MongoDB, Redis, S3, or local files via `/v1/{schema}/{table}` CRUD and `/v1/rpc/{fn}`.

Repository: [github.com/monoposer/dataspan](https://github.com/monoposer/dataspan)

## Quick start

```bash
cp .env.example .env
# Set WRAPPER_MASTER_KEY: openssl rand -base64 32
make postgres-up
make migrate
make run
```

Playground UI: http://localhost:3020/playground/

API docs (OpenAPI + Swagger UI): http://localhost:3020/swagger/ — spec at `/openapi/openapi.yaml`

Health check: `GET /health` returns `{"status":"ok","version":"..."}`.

## Docker

```bash
# Build minimal server image (~15–20MB, distroless + static binary)
make docker-build IMAGE=monoposer/dataspan:latest

# Build convert CLI image
make docker-build-cli CLI_IMAGE=monoposer/dataspan-cli:latest

# Full local stack (postgres + server)
cp .env.example .env   # set WRAPPER_MASTER_KEY
make compose-up
```

Published images (on version tags): `monoposer/dataspan` and `monoposer/dataspan-cli` on Docker Hub.

## Admin API

- `POST /api/credentials` — store encrypted secrets
- `POST /api/servers` — register foreign server (`http`|`postgres`|`mysql`|`file`|`mongo`|`redis`|`s3`|`notion`|`firebase`|`airtable`|`sheets`)
- `POST /api/tables` — register foreign table + columns
- `POST /api/functions` — register RPC mapping

## Data API (PostgREST-like)

- `GET /v1/{schema}/{table}?col=eq.val&select=a,b&limit=10`
- `POST /v1/{schema}/{table}`
- `PATCH /v1/{schema}/{table}?id=eq.1`
- `DELETE /v1/{schema}/{table}?id=eq.1`
- `POST /v1/rpc/{function}`

## Documentation

| Doc | Description |
|-----|-------------|
| [docs/](docs/README.md) | Architecture, store modes, drivers, module map |
| [deploy/](deploy/README.md) | Production deployment examples |
| [.cursor/DEVELOPMENT.md](.cursor/DEVELOPMENT.md) | Local env, curl, make targets |
| [AGENTS.md](AGENTS.md) | Cursor agent entry point |
| [readme-zh.md](readme-zh.md) | 中文说明 |

## Versioning

Current release: see [VERSION](VERSION).

Local version helpers:

```bash
make version              # current
make version-next         # next CI release
make version-bump         # bump patch in VERSION
make version-bump BUMP=minor
make version-set VER=1.0.0
```

Merging to `main` automatically bumps the patch version, updates `VERSION`, and pushes a `v*` tag. That tag triggers:

- GitHub Release with `dataspan-convert` binaries
- Docker images `monoposer/dataspan` and `monoposer/dataspan-cli` on Docker Hub

Requires repository secrets: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`.

## License

[MIT](LICENSE)
