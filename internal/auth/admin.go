package auth

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"
)

var (
	ErrAdminAuthDisabled = errors.New("admin auth disabled")
	ErrAdminMissingKey   = errors.New("missing admin api key")
	ErrAdminInvalidKey   = errors.New("invalid admin api key")
)

// Admin validates a single shared API key for /admin/api/*.
// When DATASPAN_ADMIN_KEY is unset, admin routes are open (local dev).
type Admin struct {
	Key     string
	Enabled bool
}

func NewAdminFromEnv() *Admin {
	key := strings.TrimSpace(os.Getenv("DATASPAN_ADMIN_KEY"))
	return &Admin{Key: key, Enabled: key != ""}
}

func (a *Admin) Validate(key string) error {
	if !a.Enabled {
		return ErrAdminAuthDisabled
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return ErrAdminMissingKey
	}
	if subtle.ConstantTimeCompare([]byte(key), []byte(a.Key)) != 1 {
		return ErrAdminInvalidKey
	}
	return nil
}

// AdminAPIKeyFromRequest reads the admin key from X-Api-Key or Authorization: Bearer.
func AdminAPIKeyFromRequest(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Api-Key")); v != "" {
		return v
	}
	return BearerToken(r.Header.Get("Authorization"))
}
