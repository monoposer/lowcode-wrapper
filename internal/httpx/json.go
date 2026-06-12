package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/monoposer/dataspan/internal/logx"
	"github.com/monoposer/dataspan/internal/store"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	WriteJSONContentType(w, status, "application/json", v)
}

func WriteJSONContentType(w http.ResponseWriter, status int, contentType string, v any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteAdminError(w http.ResponseWriter, r *http.Request, err error) {
	level := slog.LevelWarn
	status := http.StatusBadRequest
	if errors.Is(err, store.ErrNotFound) {
		level = slog.LevelInfo
		status = http.StatusNotFound
	}
	logx.Component("api").Log(r.Context(), level, "handler error",
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"err", err.Error(),
	)
	WriteJSON(w, status, map[string]string{"error": err.Error()})
}

func DecodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
