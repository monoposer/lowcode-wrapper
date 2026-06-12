package api

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/monoposer/dataspan/internal/logx"
)

type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (r *responseRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// Logging records HTTP requests (skips /playground/ static assets).
func Logging(next http.Handler) http.Handler {
	log := logx.Component("http")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if skipRequestLog(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote", r.RemoteAddr,
		}
		if q := r.URL.RawQuery; q != "" {
			attrs = append(attrs, "query", q)
		}

		switch {
		case status >= 500:
			log.Log(r.Context(), slog.LevelError, "request", attrs...)
		case status >= 400:
			log.Log(r.Context(), slog.LevelWarn, "request", attrs...)
		default:
			log.Log(r.Context(), slog.LevelInfo, "request", attrs...)
		}
	})
}

func skipRequestLog(path string) bool {
	return strings.HasPrefix(path, "/playground/") ||
		strings.HasPrefix(path, "/openapi/") ||
		path == "/swagger/" || path == "/swagger" || path == "/docs"
}
