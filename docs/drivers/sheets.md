# `sheets` — Google Sheets API

**Code**: `internal/driver/sheets/` → delegates to `http`

Google Sheets API v4 preset.

## Capabilities

Same as `http` (read/write depends on Sheets API path and HTTP method).

## Server `options`

| Field | Default | Description |
|-------|---------|-------------|
| `endpoint` | `https://sheets.googleapis.com/v4` | API base URL |
| `headers` | | Extra request headers |

## Credential

OAuth2 access token; any one of:

- `accessToken`
- `token`
- `oauthToken`

## Table mapping

`remoteName` is typically:

- `{spreadsheetId}/values/{range}`, or
- a path suffix compatible with the Sheets API

## Example

```yaml
servers:
  - name: gsheet
    protocol: sheets
    credential:
      accessToken: ${GOOGLE_ACCESS_TOKEN}

tables:
  - server: gsheet
    name: roster
    remoteName: spreadsheetId/Sheet1!A:Z
```

## Notes

- Token must include scopes such as `https://www.googleapis.com/auth/spreadsheets`.
- For local Excel files, use the [`file`](file.md) driver, not `sheets`.
