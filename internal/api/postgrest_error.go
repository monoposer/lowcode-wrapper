package api

import (
	"log/slog"
	"net/http"

	"github.com/monoposer/dataspan/internal/logx"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func writePostgRESTError(w http.ResponseWriter, r *http.Request, err error, ctx postgrest.ErrorContext) {
	apiErr := postgrest.MapError(err, ctx)
	if apiErr == nil {
		apiErr = postgrest.InternalError(err)
	}

	level := slog.LevelWarn
	switch {
	case apiErr.Status >= 500:
		level = slog.LevelError
	case apiErr.Status == http.StatusNotFound:
		level = slog.LevelInfo
	}

	logx.Component("api").Log(r.Context(), level, "postgrest error",
		"method", r.Method,
		"path", r.URL.Path,
		"status", apiErr.Status,
		"code", apiErr.Code,
		"message", apiErr.Message,
	)

	writeJSON(w, apiErr.Status, apiErr)
}
