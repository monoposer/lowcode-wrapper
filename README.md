# dataspan

**One PostgREST-compatible API over heterogeneous data sources.**

Register foreign servers and tables once (PostgreSQL metadata or `drivers.yaml`), then query HTTP APIs, databases, object storage, or local files through the same REST surface your frontend already knows — filters (`id=eq.1`), `select`, `order`, `limit`, RPC, and PostgREST error codes.

```
Client  →  /rest/v1/{table}  →  driver  →  remote source
```

## Quick start

```bash
cp .env.example .env          # DATASPAN_VAULT_KEY=$(openssl rand -base64 32)
make postgres-up && make migrate && make run
```

Open http://localhost:3020/swagger/

Run everything in Docker: [deploy/](deploy/README.md) (`make compose-up`).

## Data API

| Method | Path | Example |
|--------|------|---------|
| GET | `/rest/v1/{table}` | `Accept-Profile: public` · `?id=eq.1&select=id,name` |
| POST | `/rest/v1/{table}` | `Content-Profile: public` · JSON body |
| PATCH | `/rest/v1/{table}` | `Content-Profile: public` · `?id=eq.1` + body |
| DELETE | `/rest/v1/{table}` | `Content-Profile: public` · `?id=eq.1` |
| POST | `/v1/rpc/{function}` | remote procedure |

`/rest/v1/` is an alias for [Supabase JS](https://supabase.com/docs/reference/javascript) and other PostgREST clients.

Set `DATASPAN_ANON_KEY` to require `apikey` + `Authorization: Bearer` on the data API (optional JWT via `DATASPAN_JWT_SECRET`). No row-level security — gateway auth only.

Errors follow the PostgREST shape (`code`, `message`, `details`, `hint`) — e.g. `PGRST205` when a table is not registered, `PGRST301` for auth failures.

## Admin API

Register metadata before querying the data API:

- `POST /api/credentials` — encrypted secrets
- `POST /api/servers` — foreign server + protocol
- `POST /api/tables` — table + column mapping (`name` → `remote_name`)
- `POST /api/functions` — RPC mapping

Protocols: `http`, `postgres`, `mysql`, `mongo`, `redis`, `s3`, `file`, and SaaS presets (`notion`, `firebase`, `airtable`, `sheets`). See [docs/drivers.md](docs/drivers.md).

## Learn more

| | |
|-|-|
| [docs/](docs/README.md) | Architecture, store modes, drivers |
| [deploy/](deploy/README.md) | Docker images & compose |

## License

[MIT](LICENSE)
