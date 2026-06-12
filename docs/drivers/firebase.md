# `firebase` — Firestore REST

**Code**: `internal/driver/firebase/` → delegates to `http`

Google Firestore REST API (not the Admin SDK).

## Capabilities

Same as `http`.

## Server `options`

| Field | Description |
|-------|-------------|
| `projectId` | GCP project ID (used to build URL when `endpoint` is unset) |
| `database` | Firestore database id; default `(default)` |
| `endpoint` | Optional full base URL; overrides auto-built URL |

When `endpoint` is unset, base URL is:

`https://firestore.googleapis.com/v1/projects/{projectId}/databases/{database}/documents`

## Credential

Bearer token; any one of:

- `accessToken`
- `token`
- `idToken`

## Table mapping

`remoteName` is a document or collection path under `/documents/`.

## Example

```yaml
servers:
  - name: firestore
    protocol: firebase
    credential:
      accessToken: ${GOOGLE_ACCESS_TOKEN}
    options:
      projectId: my-gcp-project

tables:
  - server: firestore
    name: users
    remoteName: users
```

## Notes

- Requires a valid OAuth2 access token with Firestore scopes.
- Realtime Database is out of scope; use `http` pointed at its REST endpoint.
