package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/api/admin"
	"github.com/monoposer/dataspan/internal/api/rest"
	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/httpx"
	"github.com/monoposer/dataspan/internal/postgrest"
)

// AdminAuth enforces a shared API key on /admin/api/* when DATASPAN_ADMIN_KEY is set.
func AdminAuth(a *auth.Admin, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isAdminPath(r.URL.Path) || !a.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		if err := a.Validate(auth.AdminAPIKeyFromRequest(r)); err != nil {
			if errors.Is(err, auth.ErrAdminAuthDisabled) {
				next.ServeHTTP(w, r)
				return
			}
			msg := "invalid admin api key"
			if errors.Is(err, auth.ErrAdminMissingKey) {
				msg = "missing admin api key"
			}
			httpx.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": msg})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// DataAuth enforces apikey + Bearer validation on /v1/ and /rest/v1/.
func DataAuth(g *auth.Gateway, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isDataPath(r.URL.Path) || !g.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		claims, err := g.Validate(r.Header.Get("apikey"), auth.BearerToken(r.Header.Get("Authorization")))
		if err != nil {
			if errors.Is(err, auth.ErrAuthDisabled) {
				next.ServeHTTP(w, r)
				return
			}
			rest.WriteError(w, r, mapGatewayAuthError(err), postgrest.ErrorContext{})
			return
		}
		if claims != nil {
			r = r.WithContext(auth.WithGatewayClaims(r.Context(), claims))
		}
		next.ServeHTTP(w, r)
	})
}

func isAdminPath(path string) bool {
	return path == admin.Prefix || strings.HasPrefix(path, admin.Prefix+"/")
}

func isDataPath(path string) bool {
	return strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/rest/v1/")
}

func mapGatewayAuthError(err error) *postgrest.APIError {
	switch {
	case errors.Is(err, auth.ErrMissingAPIKey):
		return postgrest.AuthError("No API key found in request")
	case errors.Is(err, auth.ErrInvalidAPIKey):
		return postgrest.AuthError("Invalid API key")
	case errors.Is(err, auth.ErrMissingBearer):
		return postgrest.AuthError("No authorization token was found")
	case errors.Is(err, auth.ErrInvalidJWT):
		msg := "JWT cryptographic operation failed"
		if strings.Contains(err.Error(), "expired") {
			msg = "JWT expired"
		}
		return postgrest.AuthError(msg)
	default:
		return postgrest.AuthError(err.Error())
	}
}
