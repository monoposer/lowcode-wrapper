package httpdriver

import (
	"net/http"
	"testing"
)

func TestApplyHeaderMaps(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	applyHeaderMaps(req,
		map[string]string{"X-Tenant": "a", "": "skip"},
		map[string]string{"X-Tenant": "b", "X-Trace": "1"},
	)
	if got := req.Header.Get("X-Tenant"); got != "b" {
		t.Fatalf("X-Tenant = %q, want b (later layer wins)", got)
	}
	if got := req.Header.Get("X-Trace"); got != "1" {
		t.Fatalf("X-Trace = %q, want 1", got)
	}
}
