# `airtable` — Airtable REST

**Code**: `internal/driver/airtable/` → delegates to `http`

Airtable Web API v0 preset.

## Capabilities

Same as `http`.

## Server `options`

| Field | Default | Description |
|-------|---------|-------------|
| `endpoint` | `https://api.airtable.com/v0` | API base URL |
| `headers` | | Extra request headers |

## Credential

Personal Access Token / API key; any one of:

- `token`
- `personalAccessToken`
- `apiKey`
- `accessToken`

## Table mapping

`remoteName` is usually `{baseId}/{tableNameOrId}`.

## Example

```yaml
servers:
  - name: airtable_ws
    protocol: airtable
    credential:
      token: ${AIRTABLE_TOKEN}

tables:
  - server: airtable_ws
    name: contacts
    remoteName: appXXXXX/Contacts
    keyColumns: [id]
```

## Notes

- Airtable rate limits and pagination follow upstream responses; use `limit` for large datasets.
- Align field names via `columns[].remoteName` to Airtable field ids/names.
