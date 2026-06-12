package postgrest

import (
	"net/http"
	"strings"
)

const (
	MediaObjectJSON = "application/vnd.pgrst.object+json"
	MediaArrayJSON  = "application/json"
)

// ObjectMode controls single-row Accept handling (.single / .maybeSingle).
type ObjectMode int

const (
	ObjectModeArray ObjectMode = iota
	ObjectModeSingle
	ObjectModeMaybeSingle
)

// ParseObjectMode inspects Accept and Prefer headers for PostgREST object responses.
func ParseObjectMode(h http.Header) ObjectMode {
	accept := strings.ToLower(h.Get("Accept"))
	if !strings.Contains(accept, MediaObjectJSON) {
		return ObjectModeArray
	}
	if strings.Contains(accept, "missing=return_null") ||
		strings.Contains(strings.ToLower(h.Get("Prefer")), "missing=return_null") {
		return ObjectModeMaybeSingle
	}
	return ObjectModeSingle
}
