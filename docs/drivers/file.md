# `file` — Local files

**Code**: `internal/driver/file/`

Reads structured files from a local directory; suited for development, fixtures, and static datasets.

## Capabilities

| Select | Insert | Update | Delete | Upsert | Invoke |
|--------|--------|--------|--------|--------|--------|
| ✓ | ✗ | ✗ | ✗ | ✗ | ✗ |

Writes return `not supported` (phase 1).

## Server `options`

| Field | Required | Description |
|-------|----------|-------------|
| `rootPath` | ✓ | Root directory (must exist and be a directory) |

No credential required.

## Table mapping

| Field | Description |
|-------|-------------|
| `name` | Data API table name; **for xlsx, equals the sheet name** |
| `remoteName` | File path relative to `rootPath`; defaults to `name` |
| `options.format` | `csv` · `json` · `ndjson` · `yaml` · `xlsx`; inferred from extension when omitted |

Filters, `select`, `limit`, and `offset` run in memory on loaded rows.

## Example

```yaml
servers:
  - name: local_files
    protocol: file
    options:
      rootPath: /data/fixtures

tables:
  - server: local_files
    name: items
    remoteName: items.csv
    keyColumns: [id]
    options:
      format: csv
    columns:
      - name: id
      - name: name
```

Multiple xlsx sheets: several tables share `remoteName: sales.xlsx` with different `name` values (sheet names).

```bash
curl "$BASE/rest/v1/items" -H 'Accept-Profile: public'
```

## Notes

- Entire file is loaded into memory; use `limit` for large files.
- Tests: `go test ./internal/driver/file/... -v`
