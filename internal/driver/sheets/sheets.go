package sheets

import (
	"context"
	"fmt"
	"strings"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/driver/httpwrap"
	"github.com/monoposer/dataspan/internal/models"
)

func init() {
	driver.Register(models.ProtocolSheets, New)
}

// New wires Google Sheets API defaults then delegates all CRUD/RPC to the http driver.
// Table remote_name is typically {spreadsheetId}/{sheet}!{range} or values/{range} path suffix.
func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.SheetsServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		endpoint = "https://sheets.googleapis.com/v4"
	}
	headers := map[string]string{}
	for k, v := range opts.Headers {
		headers[k] = v
	}
	auth := httpwrap.BearerAuthFromCred(cred, "accessToken", "token", "oauthToken")
	defaults := models.HTTPServerOptions{
		Endpoint: endpoint,
		Auth:     auth,
		Headers:  headers,
	}
	if defaults.Auth == nil {
		return nil, fmt.Errorf("sheets server %q requires credential accessToken or options.auth", srv.Name)
	}
	return httpwrap.NewHTTPDriver(ctx, srv, cred, defaults)
}
