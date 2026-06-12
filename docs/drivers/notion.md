# `notion` — Notion API

**Code**: `internal/driver/notion/` → delegates to `http`

Notion REST preset: sets `Notion-Version` and Bearer token automatically.

## Capabilities

Same as `http` (depends on Notion API support for the configured paths).

## Server `options`

| Field | Default | Description |
|-------|---------|-------------|
| `endpoint` | `https://api.notion.com/v1` | API base URL |
| `notionVersion` | `2022-06-28` | `Notion-Version` header |
| `headers` | | Extra request headers |

## Credential

Any one of these fields (Bearer):

- `token`
- `integrationToken`
- `apiKey`

Or configure explicitly via `options.auth`.

## Table mapping

`remoteName` is a Notion API path (database id, page path, etc.) per Notion REST docs.

## Example

```yaml
servers:
  - name: notion_ws
    protocol: notion
    credential:
      token: ${NOTION_TOKEN}
    options:
      notionVersion: "2022-06-28"

tables:
  - server: notion_ws
    name: pages
    remoteName: databases/{database_id}/query
```

## Notes

- Complex Notion query bodies may need RPC `invoke` or `protocol: http` with a custom path.
- Server creation fails without a token.
