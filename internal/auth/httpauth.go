package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/monoposer/dataspan/internal/models"
)

func ApplyHTTPAuth(ctx context.Context, client *http.Client, req *http.Request, auth *models.HTTPAuth) error {
	if auth == nil {
		return nil
	}
	opts := auth.Options
	if opts == nil {
		opts = map[string]any{}
	}

	switch auth.Type {
	case "", models.AuthNone:
		return nil
	case models.AuthBasic:
		user := strOpt(opts, "username")
		pass := strOpt(opts, "password")
		token := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
		req.Header.Set("Authorization", "Basic "+token)
	case models.AuthAPIKey:
		key := strOpt(opts, "apiKey")
		label := strOpt(opts, "label")
		if label == "" {
			label = "X-API-Key"
		}
		req.Header.Set(label, key)
	case models.AuthBearerToken:
		req.Header.Set("Authorization", "Bearer "+strOpt(opts, "token"))
	case models.AuthClientCredentials:
		token, err := fetchClientCredentialsToken(ctx, client, opts)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	case models.AuthUniversal:
		for k, v := range opts {
			if k == "legacyPreset" {
				continue
			}
			req.Header.Set(k, fmt.Sprint(v))
		}
	default:
		return fmt.Errorf("unsupported auth type %q", auth.Type)
	}
	return nil
}

func strOpt(opts map[string]any, key string) string {
	if v, ok := opts[key]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

func fetchClientCredentialsToken(ctx context.Context, client *http.Client, opts map[string]any) (string, error) {
	tokenURL := strOpt(opts, "tokenUrl")
	if tokenURL == "" {
		return "", fmt.Errorf("client_credentials: tokenUrl required")
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", strOpt(opts, "clientId"))
	form.Set("client_secret", strOpt(opts, "clientSecret"))
	if scope := strOpt(opts, "scope"); scope != "" {
		form.Set("scope", scope)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, string(raw))
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", fmt.Errorf("token response missing access_token")
	}
	return parsed.AccessToken, nil
}
