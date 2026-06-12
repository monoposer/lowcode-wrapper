package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store"
	"github.com/monoposer/dataspan/internal/version"
)

type AdminHandler struct {
	Store store.Store
}

func NewAdminHandler(s store.Store) *AdminHandler {
	return &AdminHandler{Store: s}
}

func (h *AdminHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/api/credentials", h.credentials)
	mux.HandleFunc("/api/credentials/", h.deleteCredential)
	mux.HandleFunc("/api/servers", h.servers)
	mux.HandleFunc("/api/servers/", h.serverByID)
	mux.HandleFunc("/api/tables", h.tables)
	mux.HandleFunc("/api/tables/", h.tableByID)
	mux.HandleFunc("/api/columns/", h.columnByID)
	mux.HandleFunc("/api/functions", h.functions)
	mux.HandleFunc("/api/functions/", h.functionByID)
	mux.HandleFunc("/api/import", h.importMetadata)
}

func (h *AdminHandler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"version": version.Version,
	})
}

func (h *AdminHandler) credentials(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req models.CreateCredentialRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	c, err := h.Store.CreateCredential(r.Context(), req.Name, req.Data)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (h *AdminHandler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseAdminID(r.URL.Path, "/api/credentials/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := h.Store.DeleteCredential(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AdminHandler) servers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Store.ListServers(r.Context())
		if err != nil {
			writeError(w, r, err)
			return
		}
		if list == nil {
			list = []models.Server{}
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req models.CreateServerRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		srv, err := h.Store.CreateServer(r.Context(), req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, srv)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) serverByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(r.URL.Path, "/api/servers/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		srv, err := h.Store.GetServerByID(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, srv)
	case http.MethodPatch:
		var req models.UpdateServerRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		srv, err := h.Store.UpdateServer(r.Context(), id, req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, srv)
	case http.MethodDelete:
		if err := h.Store.DeleteServer(r.Context(), id); err != nil {
			writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) tables(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Store.ListTables(r.Context())
		if err != nil {
			writeError(w, r, err)
			return
		}
		if list == nil {
			list = []models.Table{}
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req models.CreateTableRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		tbl, cols, err := h.Store.CreateTable(r.Context(), req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"table": tbl, "columns": cols})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) tableByID(w http.ResponseWriter, r *http.Request) {
	id, tail, ok := parseAdminIDTail(r.URL.Path, "/api/tables/")
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
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, tbl)
	case http.MethodPatch:
		var req models.UpdateTableRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		tbl, err := h.Store.UpdateTable(r.Context(), id, req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, tbl)
	case http.MethodDelete:
		if err := h.Store.DeleteTable(r.Context(), id); err != nil {
			writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) tableColumns(w http.ResponseWriter, r *http.Request, tableID uuid.UUID) {
	switch r.Method {
	case http.MethodGet:
		cols, err := h.Store.ListColumnsByTableID(r.Context(), tableID)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if cols == nil {
			cols = []models.Column{}
		}
		writeJSON(w, http.StatusOK, cols)
	case http.MethodPost:
		var req models.CreateColumnRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		col, err := h.Store.CreateColumn(r.Context(), tableID, req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, col)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) columnByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(r.URL.Path, "/api/columns/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		col, err := h.Store.GetColumnByID(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, col)
	case http.MethodPatch:
		var req models.UpdateColumnRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		col, err := h.Store.UpdateColumn(r.Context(), id, req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, col)
	case http.MethodDelete:
		if err := h.Store.DeleteColumn(r.Context(), id); err != nil {
			writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) functions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		list, err := h.Store.ListFunctions(r.Context())
		if err != nil {
			writeError(w, r, err)
			return
		}
		if list == nil {
			list = []models.Function{}
		}
		writeJSON(w, http.StatusOK, list)
	case http.MethodPost:
		var req models.CreateFunctionRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		fn, err := h.Store.CreateFunction(r.Context(), req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusCreated, fn)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *AdminHandler) functionByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parseAdminID(r.URL.Path, "/api/functions/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		fn, err := h.Store.GetFunctionByID(r.Context(), id)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, fn)
	case http.MethodPatch:
		var req models.UpdateFunctionRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		fn, err := h.Store.UpdateFunction(r.Context(), id, req)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, fn)
	case http.MethodDelete:
		if err := h.Store.DeleteFunction(r.Context(), id); err != nil {
			writeError(w, r, err)
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
