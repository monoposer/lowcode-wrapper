# Deployment

All Docker-related files live in this directory. Build context is always the **repository root** (`..`).

| File | Purpose |
|------|---------|
| `Dockerfile` | Server image |
| `Dockerfile.cli` | CLI image (`import`, `generate types`) |
| `docker-compose.yml` | Postgres Meta DB + dataspan server (build from source) |

## Quick start

Meta DB only (run server on host with `make run`):

```bash
docker compose -f deploy/docker-compose.yml up -d postgres
make migrate
make run
```

Full stack in Docker:

```bash
docker compose -f deploy/docker-compose.yml up -d --build
```

Makefile shortcuts: `make postgres-up`, `make compose-up`.

## Build images without compose

```bash
make docker-build IMAGE=monoposer/dataspan:local
make docker-build-cli CLI_IMAGE=monoposer/dataspan-cli:local
```

Published images (`monoposer/dataspan`, `monoposer/dataspan-cli`) are built by GitHub Actions on `v*` tags. To use a pre-built server image instead of building from source, set `DATASPAN_IMAGE` in `.env` and remove the `build:` block from the compose service (or run the container directly).

## File metadata store (no Meta DB)

```bash
WRAPPER_STORE_MODE=file
WRAPPER_DRIVERS_FILE=/config/drivers.yaml
```

Uncomment the `volumes` section in `docker-compose.yml` to mount `drivers.yaml`.

## Required secrets

| Variable | Notes |
|----------|-------|
| `DATASPAN_VAULT_KEY` | 32-byte base64 AES key; **required** |
| `DATABASE_URL` or `DATABASE_DSN` | db store mode only |
| Foreign API keys | In `drivers.yaml` or via Admin API |

## Health check

```bash
curl -s http://localhost:3020/health
# {"status":"ok","version":"0.1.0"}
```
