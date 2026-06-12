package httpwrap

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"lowcode-wrapper/internal/driver"
	httpdriver "lowcode-wrapper/internal/driver/http"
	"lowcode-wrapper/internal/models"
)

// NewHTTPDriver merges defaults into server.options then uses the http driver.
// API-style protocols (notion, firebase) compose this instead of duplicating HTTP logic.
func NewHTTPDriver(ctx context.Context, srv models.Server, cred map[string]any, defaults models.HTTPServerOptions) (driver.Driver, error) {
	merged, err := MergeHTTPServerOptions(srv.Options, defaults)
	if err != nil {
		return nil, err
	}
	srv.Options = merged
	return httpdriver.New(ctx, srv, cred)
}

func MergeHTTPServerOptions(raw json.RawMessage, defaults models.HTTPServerOptions) (json.RawMessage, error) {
	var base models.HTTPServerOptions
	if len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &base); err != nil {
			return nil, err
		}
	}
	if base.Endpoint == "" {
		base.Endpoint = defaults.Endpoint
	}
	if base.BasePath == "" {
		base.BasePath = defaults.BasePath
	}
	if base.Auth == nil {
		base.Auth = defaults.Auth
	}
	base.Headers = mergeStringMaps(defaults.Headers, base.Headers)
	return json.Marshal(base)
}

func mergeStringMaps(layers ...map[string]string) map[string]string {
	out := make(map[string]string)
	for _, m := range layers {
		for k, v := range m {
			if k != "" && v != "" {
				out[k] = v
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func BearerAuthFromCred(cred map[string]any, keys ...string) *models.HTTPAuth {
	if cred == nil {
		return nil
	}
	for _, key := range keys {
		if v, ok := cred[key]; ok && strings.TrimSpace(fmt.Sprint(v)) != "" {
			return &models.HTTPAuth{
				Type:    models.AuthBearerToken,
				Options: map[string]any{"token": fmt.Sprint(v)},
			}
		}
	}
	return nil
}
