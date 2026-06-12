package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/postgrest"
	"github.com/monoposer/dataspan/internal/service"
)

type PostgRESTHandler struct {
	Engine *service.Engine
}

func NewPostgRESTHandler(e *service.Engine) *PostgRESTHandler {
	return &PostgRESTHandler{Engine: e}
}

func (h *PostgRESTHandler) Register(mux *http.ServeMux) {
	// PostgREST layout: /{table} with Accept-Profile / Content-Profile for schema.
	mux.HandleFunc("/v1/", h.handle)
	mux.HandleFunc("/rest/v1/", h.handle)
}

func (h *PostgRESTHandler) handle(w http.ResponseWriter, r *http.Request) {
	res, ok := parseDataAPIResource(r.URL.Path, r)
	if !ok {
		writePostgRESTError(w, r, postgrest.InvalidPath(), postgrest.ErrorContext{})
		return
	}
	if res.isRPC() {
		h.handleRPC(w, r, res.Schema, res.RPC)
		return
	}

	schema, table := res.Schema, res.Table
	ctx := postgrest.ErrorContext{Schema: schema, Table: table}
	q := postgrest.ParseQuery(r.URL.Query())
	prefer := postgrest.ParsePrefer(r.Header)

	switch r.Method {
	case http.MethodGet:
		rows, err := h.Engine.Select(r.Context(), schema, table, q)
		if err != nil {
			writePostgRESTError(w, r, err, ctx)
			return
		}
		if rows == nil {
			rows = []map[string]any{}
		}
		writeJSON(w, http.StatusOK, rows)
	case http.MethodPost:
		var row map[string]any
		if err := json.NewDecoder(r.Body).Decode(&row); err != nil {
			writePostgRESTError(w, r, err, ctx)
			return
		}
		ret, err := h.Engine.Insert(r.Context(), schema, table, row, prefer)
		if err != nil {
			writePostgRESTError(w, r, err, ctx)
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
			writePostgRESTError(w, r, err, ctx)
			return
		}
		n, err := h.Engine.Update(r.Context(), schema, table, q, row)
		if err != nil {
			writePostgRESTError(w, r, err, ctx)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"affected": n})
	case http.MethodDelete:
		n, err := h.Engine.Delete(r.Context(), schema, table, q)
		if err != nil {
			writePostgRESTError(w, r, err, ctx)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"affected": n})
	default:
		writePostgRESTError(w, r, postgrest.UnsupportedMethod(r.Method), ctx)
	}
}

func (h *PostgRESTHandler) handleRPC(w http.ResponseWriter, r *http.Request, schema, name string) {
	ctx := postgrest.ErrorContext{Schema: schema, RPC: name}

	if r.Method != http.MethodPost {
		writePostgRESTError(w, r, postgrest.InvalidRPCMethod(), ctx)
		return
	}
	var body map[string]any
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writePostgRESTError(w, r, err, ctx)
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
		writePostgRESTError(w, r, err, ctx)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func stripDataAPIPrefix(path string) string {
	for _, prefix := range []string{"/rest/v1/", "/v1/"} {
		if strings.HasPrefix(path, prefix) {
			return strings.TrimPrefix(path, prefix)
		}
	}
	return path
}
