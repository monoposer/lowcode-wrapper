package notion

import (
	"context"
	"fmt"
	"strings"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/driver/httpwrap"
	"lowcode-wrapper/internal/models"
)

func init() {
	driver.Register(models.ProtocolNotion, New)
}

// New wires Notion REST defaults then delegates all CRUD/RPC to the http driver.
func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.NotionServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	version := strings.TrimSpace(opts.NotionVersion)
	if version == "" {
		version = "2022-06-28"
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.notion.com/v1"
	}
	headers := map[string]string{"Notion-Version": version}
	for k, v := range opts.Headers {
		headers[k] = v
	}
	auth := httpwrap.BearerAuthFromCred(cred, "token", "integrationToken", "apiKey")
	defaults := models.HTTPServerOptions{
		Endpoint: endpoint,
		Auth:     auth,
		Headers:  headers,
	}
	if defaults.Auth == nil {
		return nil, fmt.Errorf("notion server %q requires credential token or options.auth", srv.Name)
	}
	return httpwrap.NewHTTPDriver(ctx, srv, cred, defaults)
}
