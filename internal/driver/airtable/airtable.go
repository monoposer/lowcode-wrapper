package airtable

import (
	"context"
	"fmt"
	"strings"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/driver/httpwrap"
	"lowcode-wrapper/internal/models"
)

func init() {
	driver.Register(models.ProtocolAirtable, New)
}

// New wires Airtable REST defaults then delegates all CRUD/RPC to the http driver.
// Table remote_name is typically {baseId}/{tableNameOrId}.
func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.AirtableServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.airtable.com/v0"
	}
	headers := map[string]string{}
	for k, v := range opts.Headers {
		headers[k] = v
	}
	auth := httpwrap.BearerAuthFromCred(cred, "token", "personalAccessToken", "apiKey", "accessToken")
	defaults := models.HTTPServerOptions{
		Endpoint: endpoint,
		Auth:     auth,
		Headers:  headers,
	}
	if defaults.Auth == nil {
		return nil, fmt.Errorf("airtable server %q requires credential token or options.auth", srv.Name)
	}
	return httpwrap.NewHTTPDriver(ctx, srv, cred, defaults)
}
