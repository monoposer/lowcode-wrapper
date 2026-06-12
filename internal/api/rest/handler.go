package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/engine"
	"github.com/monoposer/dataspan/internal/httpx"
	"github.com/monoposer/dataspan/internal/postgrest"
)

type Handler struct {
	Engine *engine.Engine
}

func New(e *engine.Engine) *Handler {
	return &Handler{Engine: e}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/v1/", h.handle)
	mux.HandleFunc("/rest/v1/", h.handle)
}

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		WriteError(w, r, postgrest.UnsupportedMethod(r.Method), postgrest.ErrorContext{})
		return
	}
	h.serveOpenAPI(w, r, "/rest/v1")
}

func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	if isOpenAPIPath(r.URL.Path) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			WriteError(w, r, postgrest.UnsupportedMethod(r.Method), postgrest.ErrorContext{})
			return
		}
		h.serveOpenAPI(w, r, openAPIBasePath(r.URL.Path))
		return
	}

	res, ok := parseDataAPIResource(r.URL.Path, r)
	if !ok {
		WriteError(w, r, postgrest.InvalidPath(), postgrest.ErrorContext{})
		return
	}
	if res.isRPC() {
		h.handleRPC(w, r, res.Schema, res.RPC)
		return
	}

	schema, table := res.Schema, res.Table
	ctx := postgrest.ErrorContext{Schema: schema, Table: table}
	q := postgrest.ParseQuery(r.URL.Query())
	if apiErr := postgrest.ValidateSelect(q.Select, r.URL.Query().Get("select")); apiErr != nil {
		WriteError(w, r, apiErr, ctx)
		return
	}
	prefer := postgrest.ParsePrefer(r.Header)

	switch r.Method {
	case http.MethodGet:
		h.handleSelect(w, r, schema, table, q, prefer, ctx)
	case http.MethodPost:
		h.handleInsert(w, r, schema, table, prefer, ctx)
	case http.MethodPatch:
		h.handleUpdate(w, r, schema, table, q, prefer, ctx)
	case http.MethodDelete:
		h.handleDelete(w, r, schema, table, q, prefer, ctx)
	default:
		WriteError(w, r, postgrest.UnsupportedMethod(r.Method), ctx)
	}
}

func (h *Handler) handleSelect(w http.ResponseWriter, r *http.Request, schema, table string, q postgrest.Query, prefer postgrest.Prefer, ctx postgrest.ErrorContext) {
	rows, err := h.Engine.Select(r.Context(), schema, table, q)
	if err != nil {
		WriteError(w, r, err, ctx)
		return
	}
	total := -1
	if prefer.CountExact {
		total, err = h.Engine.Count(r.Context(), schema, table, q)
		if err != nil {
			WriteError(w, r, err, ctx)
			return
		}
	}
	if err := writeSelectResult(w, r, rows, q, prefer, total); err != nil {
		if apiErr, ok := err.(*postgrest.APIError); ok {
			WriteError(w, r, apiErr, ctx)
			return
		}
		WriteError(w, r, err, ctx)
	}
}

func (h *Handler) handleInsert(w http.ResponseWriter, r *http.Request, schema, table string, prefer postgrest.Prefer, ctx postgrest.ErrorContext) {
	rows, single, err := decodeInsertBody(r)
	if err != nil {
		WriteError(w, r, err, ctx)
		return
	}
	if len(rows) == 0 {
		WriteError(w, r, postgrest.InvalidBody(nil), ctx)
		return
	}

	if !single {
		ret, err := h.Engine.InsertMany(r.Context(), schema, table, rows, prefer)
		if err != nil {
			WriteError(w, r, err, ctx)
			return
		}
		if prefer.Representation {
			httpx.WriteJSON(w, http.StatusCreated, ret)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	}

	ret, err := h.Engine.Insert(r.Context(), schema, table, rows[0], prefer)
	if err != nil {
		WriteError(w, r, err, ctx)
		return
	}
	if prefer.Representation && ret != nil {
		httpx.WriteJSON(w, http.StatusCreated, ret)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request, schema, table string, q postgrest.Query, prefer postgrest.Prefer, ctx postgrest.ErrorContext) {
	var row map[string]any
	if err := json.NewDecoder(r.Body).Decode(&row); err != nil {
		WriteError(w, r, err, ctx)
		return
	}
	_, err := h.Engine.Update(r.Context(), schema, table, q, row)
	if err != nil {
		WriteError(w, r, err, ctx)
		return
	}
	if prefer.Representation {
		rows, err := h.Engine.Select(r.Context(), schema, table, q)
		if err != nil {
			WriteError(w, r, err, ctx)
			return
		}
		writeMutationRows(w, rows)
		return
	}
	writeMutationNoContent(w)
}

func (h *Handler) handleDelete(w http.ResponseWriter, r *http.Request, schema, table string, q postgrest.Query, prefer postgrest.Prefer, ctx postgrest.ErrorContext) {
	_, err := h.Engine.Delete(r.Context(), schema, table, q)
	if err != nil {
		WriteError(w, r, err, ctx)
		return
	}
	if prefer.Representation {
		rows, err := h.Engine.Select(r.Context(), schema, table, q)
		if err != nil {
			WriteError(w, r, err, ctx)
			return
		}
		writeMutationRows(w, rows)
		return
	}
	writeMutationNoContent(w)
}

func (h *Handler) handleRPC(w http.ResponseWriter, r *http.Request, schema, name string) {
	ctx := postgrest.ErrorContext{Schema: schema, RPC: name}

	if r.Method != http.MethodPost {
		WriteError(w, r, postgrest.InvalidRPCMethod(), ctx)
		return
	}
	var body map[string]any
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			WriteError(w, r, err, ctx)
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
		WriteError(w, r, err, ctx)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) serveOpenAPI(w http.ResponseWriter, r *http.Request, basePath string) {
	schema := schemaFromRequest(r)
	spec, err := postgrest.BuildOpenAPI(r.Context(), h.Engine.Store, postgrest.OpenAPIOptions{
		Host:     r.Host,
		BasePath: basePath,
		Schema:   schemaFilterForOpenAPI(schema),
	})
	if err != nil {
		WriteError(w, r, err, postgrest.ErrorContext{})
		return
	}
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "application/openapi+json")
		w.WriteHeader(http.StatusOK)
		return
	}
	httpx.WriteJSONContentType(w, http.StatusOK, "application/openapi+json", spec)
}

func isOpenAPIPath(path string) bool {
	return path == "/rest/v1" || path == "/rest/v1/" || path == "/v1" || path == "/v1/"
}

func openAPIBasePath(path string) string {
	if strings.HasPrefix(path, "/v1") {
		return "/v1"
	}
	return "/rest/v1"
}

func schemaFilterForOpenAPI(profile string) string {
	if profile == "" || profile == defaultSchema {
		return ""
	}
	return profile
}

func stripDataAPIPrefix(path string) string {
	for _, prefix := range []string{"/rest/v1/", "/v1/"} {
		if strings.HasPrefix(path, prefix) {
			return strings.TrimPrefix(path, prefix)
		}
	}
	return path
}
