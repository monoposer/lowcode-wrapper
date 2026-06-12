package api

import (
	"net/http"
	"strings"
)

const defaultSchema = "public"

// dataAPIResource is a resolved table or RPC target from the request path.
type dataAPIResource struct {
	Schema string
	Table  string
	RPC    string
}

func (r dataAPIResource) isRPC() bool {
	return r.RPC != ""
}

// schemaFromRequest reads the active schema per PostgREST conventions.
// GET/HEAD use Accept-Profile; other methods use Content-Profile. Defaults to public.
func schemaFromRequest(r *http.Request) string {
	var profile string
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		profile = strings.TrimSpace(r.Header.Get("Accept-Profile"))
	default:
		profile = strings.TrimSpace(r.Header.Get("Content-Profile"))
	}
	if profile == "" {
		return defaultSchema
	}
	return profile
}

// parseDataAPIResource resolves PostgREST-style paths:
//   - /{table}           — schema from Accept-Profile / Content-Profile (default public)
//   - /rpc/{function}    — RPC; schema from ?schema=, profile headers, or public
//   - /{schema}/{table}  — legacy dataspan layout (still supported)
func parseDataAPIResource(path string, r *http.Request) (dataAPIResource, bool) {
	tail := strings.Trim(stripDataAPIPrefix(path), "/")
	if tail == "" {
		return dataAPIResource{}, false
	}
	parts := strings.Split(tail, "/")

	switch len(parts) {
	case 1:
		if parts[0] == "rpc" {
			return dataAPIResource{}, false
		}
		return dataAPIResource{Schema: schemaFromRequest(r), Table: parts[0]}, true
	case 2:
		if parts[0] == "rpc" {
			schema := strings.TrimSpace(r.URL.Query().Get("schema"))
			if schema == "" {
				schema = schemaFromRequest(r)
			}
			return dataAPIResource{Schema: schema, RPC: parts[1]}, true
		}
		// Legacy: /{schema}/{table}
		return dataAPIResource{Schema: parts[0], Table: parts[1]}, true
	default:
		return dataAPIResource{}, false
	}
}
