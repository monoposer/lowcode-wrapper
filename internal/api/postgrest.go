package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"lowcode-wrapper/internal/postgrest"
	"lowcode-wrapper/internal/service"
)

type PostgRESTHandler struct {
	Engine *service.Engine
}

func NewPostgRESTHandler(e *service.Engine) *PostgRESTHandler {
	return &PostgRESTHandler{Engine: e}
}

func (h *PostgRESTHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/v1/", h.handle)
}

func (h *PostgRESTHandler) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	if len(parts) == 2 && parts[0] == "rpc" {
		h.handleRPC(w, r, parts[1])
		return
	}
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}
	schema, table := parts[0], parts[1]
	q := postgrest.ParseQuery(r.URL.Query())
	prefer := postgrest.ParsePrefer(r.Header)

	switch r.Method {
	case http.MethodGet:
		rows, err := h.Engine.Select(r.Context(), schema, table, q)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, rows)
	case http.MethodPost:
		var row map[string]any
		if err := json.NewDecoder(r.Body).Decode(&row); err != nil {
			writeError(w, r, err)
			return
		}
		ret, err := h.Engine.Insert(r.Context(), schema, table, row, prefer)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if prefer.Representation && ret != nil {
			writeJSON(w, http.StatusCreated, ret)
			return
		}
		w.WriteHeader(http.StatusCreated)
	case http.MethodPatch:
		var row map[string]any
		if err := json.NewDecoder(r.Body).Decode(&row); err != nil {
			writeError(w, r, err)
			return
		}
		n, err := h.Engine.Update(r.Context(), schema, table, q, row)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"affected": n})
	case http.MethodDelete:
		n, err := h.Engine.Delete(r.Context(), schema, table, q)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"affected": n})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *PostgRESTHandler) handleRPC(w http.ResponseWriter, r *http.Request, name string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	schema := r.URL.Query().Get("schema")
	if schema == "" {
		schema = "public"
	}
	var body map[string]any
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, r, err)
			return
		}
	}
	query := make(map[string]string)
	for k, v := range r.URL.Query() {
		if k == "schema" || len(v) == 0 {
			continue
		}
		query[k] = v[0]
	}
	result, err := h.Engine.InvokeRPC(r.Context(), schema, name, body, query)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
