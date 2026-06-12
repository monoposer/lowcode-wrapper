package admin

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/httpx"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store"
	"github.com/monoposer/dataspan/internal/version"
)

// Prefix is the URL prefix for metadata CRUD and introspection.
const Prefix = "/admin/api"

type Handler struct {
	Store store.Store
	Mode  store.Mode
}

func New(s store.Store, mode store.Mode) *Handler {
	return &Handler{Store: s, Mode: mode}
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/health/ready", h.healthReady)
	p := Prefix
	mux.HandleFunc(p+"/credentials", h.credentials)
	mux.HandleFunc(p+"/credentials/", h.deleteCredential)
	mux.HandleFunc(p+"/servers", h.servers)
	mux.HandleFunc(p+"/servers/", h.serverByID)
	mux.HandleFunc(p+"/tables", h.tables)
	mux.HandleFunc(p+"/tables/", h.tableByID)
	mux.HandleFunc(p+"/columns/", h.columnByID)
	mux.HandleFunc(p+"/functions", h.functions)
	mux.HandleFunc(p+"/functions/", h.functionByID)
	mux.HandleFunc(p+"/import", h.importMetadata)
}

// adminWriteDenied blocks metadata mutations in file store mode (GET introspection remains available).
func (h *Handler) adminWriteDenied(w http.ResponseWriter) bool {
	if h.Mode != store.ModeFile {
		return false
	}
	httpx.WriteJSON(w, http.StatusMethodNotAllowed, map[string]string{
		"error": "admin write API is not available when DATASPAN_STORE_MODE=file; edit drivers.yaml instead",
	})
	return true
}

func (h *Handler) healthReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}
	if err := store.Ping(r.Context(), cfg); err != nil {
		httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "error",
			"store":  string(cfg.Mode),
			"error":  err.Error(),
		})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "ready",
		"store":   string(cfg.Mode),
		"version": version.Version,
	})
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": version.Version,
	})
}

func (h *Handler) credentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.adminWriteDenied(w) {
		return
	}
	var req models.CreateCredentialRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteAdminError(w, r, err)
		return
	}
	c, err := h.Store.CreateCredential(r.Context(), req.Name, req.Data)
	if err != nil {
		httpx.WriteAdminError(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, c)
}

func (h *Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.adminWriteDenied(w) {
		return
	}
	id, ok := parseAdminID(r.URL.Path, Prefix+"/credentials/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.Store.DeleteCredential(r.Context(), id); err != nil {
		httpx.WriteAdminError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) servers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Store.ListServers(r.Context())
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		if list == nil {
			list = []models.Server{}
		}
		httpx.WriteJSON(w, http.StatusOK, list)
	case http.MethodPost:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.CreateServerRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		srv, err := h.Store.CreateServer(r.Context(), req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, srv)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) serverByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(r.URL.Path, Prefix+"/servers/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		srv, err := h.Store.GetServerByID(r.Context(), id)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, srv)
	case http.MethodPatch:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.UpdateServerRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		srv, err := h.Store.UpdateServer(r.Context(), id, req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, srv)
	case http.MethodDelete:
		if h.adminWriteDenied(w) {
			return
		}
		if err := h.Store.DeleteServer(r.Context(), id); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) tables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Store.ListTables(r.Context())
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		if list == nil {
			list = []models.Table{}
		}
		httpx.WriteJSON(w, http.StatusOK, list)
	case http.MethodPost:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.CreateTableRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		tbl, cols, err := h.Store.CreateTable(r.Context(), req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, map[string]any{"table": tbl, "columns": cols})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) tableByID(w http.ResponseWriter, r *http.Request) {
	id, tail, ok := parseAdminIDTail(r.URL.Path, Prefix+"/tables/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if len(tail) == 1 && tail[0] == "columns" {
		h.tableColumns(w, r, id)
		return
	}
	if len(tail) != 0 {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		tbl, err := h.Store.GetTableByID(r.Context(), id)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, tbl)
	case http.MethodPatch:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.UpdateTableRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		tbl, err := h.Store.UpdateTable(r.Context(), id, req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, tbl)
	case http.MethodDelete:
		if h.adminWriteDenied(w) {
			return
		}
		if err := h.Store.DeleteTable(r.Context(), id); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) tableColumns(w http.ResponseWriter, r *http.Request, tableID uuid.UUID) {
	switch r.Method {
	case http.MethodGet:
		cols, err := h.Store.ListColumnsByTableID(r.Context(), tableID)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		if cols == nil {
			cols = []models.Column{}
		}
		httpx.WriteJSON(w, http.StatusOK, cols)
	case http.MethodPost:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.CreateColumnRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		col, err := h.Store.CreateColumn(r.Context(), tableID, req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, col)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) columnByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(r.URL.Path, Prefix+"/columns/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		col, err := h.Store.GetColumnByID(r.Context(), id)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, col)
	case http.MethodPatch:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.UpdateColumnRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		col, err := h.Store.UpdateColumn(r.Context(), id, req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, col)
	case http.MethodDelete:
		if h.adminWriteDenied(w) {
			return
		}
		if err := h.Store.DeleteColumn(r.Context(), id); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) functions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Store.ListFunctions(r.Context())
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		if list == nil {
			list = []models.Function{}
		}
		httpx.WriteJSON(w, http.StatusOK, list)
	case http.MethodPost:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.CreateFunctionRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		fn, err := h.Store.CreateFunction(r.Context(), req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusCreated, fn)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) functionByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(r.URL.Path, Prefix+"/functions/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		fn, err := h.Store.GetFunctionByID(r.Context(), id)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, fn)
	case http.MethodPatch:
		if h.adminWriteDenied(w) {
			return
		}
		var req models.UpdateFunctionRequest
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		fn, err := h.Store.UpdateFunction(r.Context(), id, req)
		if err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		httpx.WriteJSON(w, http.StatusOK, fn)
	case http.MethodDelete:
		if h.adminWriteDenied(w) {
			return
		}
		if err := h.Store.DeleteFunction(r.Context(), id); err != nil {
			httpx.WriteAdminError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseAdminID(path, prefix string) (uuid.UUID, bool) {
	id, tail, ok := parseAdminIDTail(path, prefix)
	if !ok || len(tail) != 0 {
		return uuid.Nil, false
	}
	return id, true
}

func parseAdminIDTail(path, prefix string) (uuid.UUID, []string, bool) {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return uuid.Nil, nil, false
	}
	parts := strings.Split(rest, "/")
	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, nil, false
	}
	return id, parts[1:], true
}
