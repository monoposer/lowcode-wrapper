package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/store"
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
	mux.HandleFunc("/api/tables/", h.tableColumns)
	mux.HandleFunc("/api/functions", h.functions)
}

func (h *AdminHandler) health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	idStr := strings.TrimPrefix(r.URL.Path, "/api/credentials/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, r, err)
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
	idStr := strings.TrimPrefix(r.URL.Path, "/api/servers/")
	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, r, err)
		return
	}
	switch r.Method {
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

func (h *AdminHandler) tableColumns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/tables/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 3 || parts[2] != "columns" {
		http.NotFound(w, r)
		return
	}
	cols, err := h.Store.ListColumns(r.Context(), parts[0], parts[1])
	if err != nil {
		writeError(w, r, err)
		return
	}
	if cols == nil {
		cols = []models.Column{}
	}
	writeJSON(w, http.StatusOK, cols)
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
