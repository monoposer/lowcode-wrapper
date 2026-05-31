package firebasedriver

import (
	"context"
	"fmt"
	"strings"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/driver/httpwrap"
	"lowcode-wrapper/internal/models"
)

func init() {
	driver.Register(models.ProtocolFirebase, New)
}

// New targets Firestore REST; table remote_name is the document path under /documents/.
func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.FirebaseServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimSpace(opts.Endpoint)
	if endpoint == "" {
		project := strings.TrimSpace(opts.ProjectID)
		if project == "" && cred != nil {
			project = strings.TrimSpace(fmt.Sprint(cred["projectId"]))
		}
		if project == "" {
			return nil, fmt.Errorf("firebase server %q requires options.projectId or credential.projectId", srv.Name)
		}
		db := strings.TrimSpace(opts.Database)
		if db == "" {
			db = "(default)"
		}
		endpoint = fmt.Sprintf(
			"https://firestore.googleapis.com/v1/projects/%s/databases/%s/documents",
			project, db,
		)
	}
	auth := httpwrap.BearerAuthFromCred(cred, "accessToken", "token", "idToken")
	defaults := models.HTTPServerOptions{
		Endpoint: endpoint,
		Auth:     auth,
	}
	if defaults.Auth == nil {
		return nil, fmt.Errorf("firebase server %q requires credential accessToken (or configure options.auth)", srv.Name)
	}
	return httpwrap.NewHTTPDriver(ctx, srv, cred, defaults)
}
