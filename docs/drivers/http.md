# `http` — Generic REST driver

**Code**: `internal/driver/http/`

Preferred driver for any HTTP/REST upstream: microservices, third-party SaaS, or an existing PostgREST instance.

## Capabilities

| Select | Insert | Update | Delete | Upsert | Invoke |
|--------|--------|--------|--------|--------|--------|
| ✓ | ✓ | ✓ | ✓ | ✓ | ✓ |

Outbound requests append PostgREST-style query params (`id=eq.1`, `select`, `order`, `limit`); writes use a JSON body.

## Server `options`

| Field | Required | Description |
|-------|----------|-------------|
| `endpoint` | ✓ | Base URL, e.g. `https://api.example.com` |
| `basePath` | | Path prefix, e.g. `/v1` |
| `headers` | | Default request headers (overridable per table/function via `options.headers`) |
| `auth` | | Outbound auth; see table below |

### `auth.type`

| type | Common `options` fields |
|------|-------------------------|
| `NONE` | — |
| `BASIC` | `username`, `password` |
| `API_KEY` | `apiKey`, `label` (default `X-API-Key`) |
| `BEARER_TOKEN` | `token` |
| `CLIENT_CREDENTIALS` | OAuth2 client-credentials fields |
| `UNIVERSAL` | Credential fields merged into options |

When `auth` is omitted, credential fields are merged as `UNIVERSAL`.

## Table / Function `options`

```yaml
tables:
  - server: partner_api
    name: orders
    remoteName: orders          # remote path segment; defaults to name
    options:
      headers:                  # extra headers for this table only
        X-Table: orders
```

Function `remotePath` + `method` are used for `Invoke` (RPC).

## Example

```yaml
servers:
  - name: partner_api
    protocol: http
    credential:
      apiKey: ${PARTNER_API_KEY}
    options:
      endpoint: https://api.example.com
      basePath: /v1
      auth:
        type: API_KEY
        options:
          label: X-Api-Key
```

```bash
curl "$BASE/rest/v1/orders?id=eq.1" -H 'Accept-Profile: public'
```

## Notes

- The upstream must understand the REST semantics dataspan forwards; use a dedicated driver or an adapter layer for non-REST APIs.
- Tests: `go test ./internal/driver/http/... -v`
