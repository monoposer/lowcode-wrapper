package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/monoposer/dataspan/internal/logx"
	"github.com/monoposer/dataspan/internal/store"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
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
	if status == http.StatusNotFound {
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Prefer, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
