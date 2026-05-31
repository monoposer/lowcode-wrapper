package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
)

// postgRESTMock simulates a minimal PostgREST-style HTTP API for http driver tests.
type postgRESTMock struct {
	mu       sync.Mutex
	requests []recordedHTTP
}

type recordedHTTP struct {
	Method   string
	Path     string
	RawQuery string
	Headers  http.Header
	Body     string
}

func newPostgRESTMock() (*httptest.Server, *postgRESTMock) {
	m := &postgRESTMock{}
	ts := httptest.NewServer(http.HandlerFunc(m.serveHTTP))
	return ts, m
}

func (m *postgRESTMock) last() recordedHTTP {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		return recordedHTTP{}
	}
	return m.requests[len(m.requests)-1]
}

func (m *postgRESTMock) record(r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()
	if len(body) > 0 {
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}
	m.mu.Lock()
	m.requests = append(m.requests, recordedHTTP{
		Method:   r.Method,
		Path:     r.URL.Path,
		RawQuery: r.URL.RawQuery,
		Headers:  r.Header.Clone(),
		Body:     string(body),
	})
	m.mu.Unlock()
}

func (m *postgRESTMock) serveHTTP(w http.ResponseWriter, r *http.Request) {
	m.record(r)
	w.Header().Set("Content-Type", "application/json")

	if r.URL.Path != "/orders" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		m.handleGet(w, r)
	case http.MethodPost:
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"3","total_amount":"10"}`))
	case http.MethodPatch:
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"affected":1}`))
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *postgRESTMock) handleGet(w http.ResponseWriter, r *http.Request) {
	all := []map[string]any{
		{"id": "1", "total_amount": "99"},
		{"id": "2", "total_amount": "50"},
	}
	q := r.URL.Query()
	if q.Get("id") == "eq.1" {
		writeJSON(w, []map[string]any{all[0]})
		return
	}
	if q.Get("total_amount") == "eq.99" {
		writeJSON(w, []map[string]any{all[0]})
		return
	}
	if q.Get("select") == "id,total_amount" {
		writeJSON(w, all)
		return
	}
	writeJSON(w, all)
}

func writeJSON(w http.ResponseWriter, v any) {
	_ = json.NewEncoder(w).Encode(v)
}
