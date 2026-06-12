package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/postgrest"
)

// DataAPIAuth enforces apikey + Bearer validation on /v1/ and /rest/v1/.
// When DATASPAN_ANON_KEY is unset the gateway is disabled (local dev).
func DataAPIAuth(g *auth.Gateway, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isDataAPIPath(r.URL.Path) || !g.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		claims, err := g.Validate(r.Header.Get("apikey"), auth.BearerToken(r.Header.Get("Authorization")))
		if err != nil {
			if errors.Is(err, auth.ErrAuthDisabled) {
				next.ServeHTTP(w, r)
				return
			}
			writeGatewayAuthError(w, r, mapGatewayAuthError(err))
			return
		}
		if claims != nil {
			r = r.WithContext(auth.WithGatewayClaims(r.Context(), claims))
		}
		next.ServeHTTP(w, r)
	})
}

func isDataAPIPath(path string) bool {
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

func writeGatewayAuthError(w http.ResponseWriter, r *http.Request, apiErr *postgrest.APIError) {
	writePostgRESTError(w, r, apiErr, postgrest.ErrorContext{})
}
