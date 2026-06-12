package api

import (
	"io"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/importer"
	"github.com/monoposer/dataspan/internal/store"
)

type importRequest struct {
	Kind       string `json:"kind"`
	Content    string `json:"content"`
	Dialect    string `json:"dialect,omitempty"`
	Mode       string `json:"mode,omitempty"`
	ServerName string `json:"serverName,omitempty"`
	Endpoint   string `json:"endpoint,omitempty"`
	Schema     string `json:"schema,omitempty"`
}

func (h *AdminHandler) importMetadata(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg, err := store.LoadConfig()
	if err != nil {
		writeError(w, r, err)
		return
	}
	if cfg.Mode != store.ModeDB {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "import API is only available when WRAPPER_STORE_MODE=db",
		})
		return
	}

	var req importRequest
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	switch {
	case strings.HasPrefix(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeError(w, r, err)
			return
		}
		req.Kind = r.FormValue("kind")
		req.Dialect = r.FormValue("dialect")
		req.Mode = r.FormValue("mode")
		req.ServerName = r.FormValue("serverName")
		req.Endpoint = r.FormValue("endpoint")
		req.Schema = r.FormValue("schema")
		file, _, err := r.FormFile("file")
		if err != nil {
			writeError(w, r, err)
			return
		}
		defer file.Close()
		raw, err := io.ReadAll(file)
		if err != nil {
			writeError(w, r, err)
			return
		}
		req.Content = string(raw)
	case strings.HasPrefix(ct, "application/yaml"), strings.HasPrefix(ct, "text/yaml"):
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, err)
			return
		}
		req.Kind = r.URL.Query().Get("kind")
		if req.Kind == "" {
			req.Kind = "yaml"
		}
		req.Dialect = r.URL.Query().Get("dialect")
		req.Mode = r.URL.Query().Get("mode")
		req.ServerName = r.URL.Query().Get("serverName")
		req.Endpoint = r.URL.Query().Get("endpoint")
		req.Schema = r.URL.Query().Get("schema")
		req.Content = string(raw)
	default:
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, r, err)
			return
		}
	}

	if strings.TrimSpace(req.Content) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "content is required"})
		return
	}

	kind := importer.Kind(strings.TrimSpace(req.Kind))
	if kind == "" {
		kind = importer.DetectKind("input", []byte(req.Content))
	}
	conv := importer.ConvertOptions{Kind: kind, Input: []byte(req.Content)}
	switch kind {
	case importer.KindOpenAPI:
		conv.OpenAPI = importer.OpenAPIOptions{
			ServerName: req.ServerName,
			Endpoint:   req.Endpoint,
		}
	case importer.KindSQL:
		d, err := importer.ParseSQLDialect(req.Dialect)
		if err != nil {
			writeError(w, r, err)
			return
		}
		schema := strings.TrimSpace(req.Schema)
		if schema == "" {
			schema = "public"
		}
		conv.SQL = importer.SQLOptions{
			ServerName: req.ServerName,
			Schema:     schema,
			Dialect:    d,
		}
	}

	yamlOut, err := importer.Convert(conv)
	if err != nil {
		writeError(w, r, err)
		return
	}

	mode := importer.ImportMode(strings.TrimSpace(req.Mode))
	if mode == "" {
		mode = importer.ModeReplace
	}
	result, err := store.ImportYAML(r.Context(), h.Store, yamlOut, mode)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"imported": result,
		"kind":     kind,
		"mode":     mode,
	})
}
