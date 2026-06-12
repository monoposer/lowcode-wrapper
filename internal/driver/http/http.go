package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolHTTP, New)
}

type Driver struct {
	baseURL    string
	basePath   string
	auth       *models.HTTPAuth
	cred       map[string]any
	headers    map[string]string
	httpClient *http.Client
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.HTTPServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(opts.Endpoint) == "" {
		return nil, fmt.Errorf("http server %q requires options.endpoint", srv.Name)
	}
	authCfg := opts.Auth
	if authCfg != nil && cred != nil {
		merged := auth.MergeCredentialIntoOptions(authCfg.Options, cred)
		authCfg = &models.HTTPAuth{Type: authCfg.Type, Options: merged}
	} else if authCfg == nil && cred != nil {
		authCfg = &models.HTTPAuth{Type: models.AuthUniversal, Options: cred}
	}
	return &Driver{
		baseURL:    strings.TrimSuffix(strings.TrimSpace(opts.Endpoint), "/"),
		basePath:   strings.TrimSpace(opts.BasePath),
		auth:       authCfg,
		cred:       cred,
		headers:    opts.Headers,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (d *Driver) resourcePath(schema, table string) string {
	t := models.RemoteTableName(models.Table{TableName: table, RemoteName: table})
	if schema != "" && schema != "public" {
		t = schema + "/" + t
	}
	if d.basePath != "" {
		t = strings.Trim(d.basePath, "/") + "/" + strings.TrimPrefix(t, "/")
	}
	return "/" + strings.TrimPrefix(strings.TrimSpace(t), "/")
}

func (d *Driver) tableResourcePath(resolved *models.ResolvedTable) string {
	remote := models.RemoteTableName(resolved.Table)
	schema := resolved.Table.SchemaName
	if schema != "" && schema != "public" {
		remote = schema + "/" + remote
	}
	if d.basePath != "" {
		remote = strings.Trim(d.basePath, "/") + "/" + strings.TrimPrefix(remote, "/")
	}
	return "/" + strings.TrimPrefix(strings.TrimSpace(remote), "/")
}

func (d *Driver) applyAuth(ctx context.Context, req *http.Request) error {
	return auth.ApplyHTTPAuth(ctx, d.httpClient, req, d.auth)
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	path := d.tableResourcePath(req.Resolved)
	q := postgrest.FiltersToQueryValues(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	if len(req.Select) > 0 {
		q.Set("select", strings.Join(req.Select, ","))
	}
	for _, o := range req.Order {
		suffix := ".asc"
		if o.Desc {
			suffix = ".desc"
		}
		q.Add("order", o.Column+suffix)
	}
	if req.Limit > 0 {
		q.Set("limit", fmt.Sprint(req.Limit))
	}
	if req.Offset > 0 {
		q.Set("offset", fmt.Sprint(req.Offset))
	}
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, d.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	d.setRequestHeaders(httpReq, tableHeaderOptions(req.Resolved.Table.Options))
	httpReq.Header.Set("Accept", "application/json")
	if err := d.applyAuth(ctx, httpReq); err != nil {
		return nil, err
	}
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http select %s: %s %s", path, resp.Status, string(body))
	}
	rows, err := parseJSONRows(body)
	if err != nil {
		return nil, err
	}
	return postgrest.MapRowsFromRemote(rows, req.Resolved.Columns), nil
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	body, err := json.Marshal(row)
	if err != nil {
		return nil, err
	}
	path := d.tableResourcePath(req.Resolved)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	d.setRequestHeaders(httpReq, tableHeaderOptions(req.Resolved.Table.Options))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	if req.PreferRepresentation {
		httpReq.Header.Set("Prefer", "return=representation")
	}
	if err := d.applyAuth(ctx, httpReq); err != nil {
		return nil, err
	}
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http insert %s: %s %s", path, resp.Status, string(rb))
	}
	if !req.PreferRepresentation || len(rb) == 0 {
		return nil, nil
	}
	return parseOneRow(rb, req.Resolved.Columns)
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	keySet := keyColumnSet(req.Resolved.Table.KeyColumns)
	setRow := make(map[string]any)
	for k, v := range row {
		if !keySet[k] {
			setRow[k] = v
		}
	}
	body, err := json.Marshal(setRow)
	if err != nil {
		return 0, err
	}
	path := d.tableResourcePath(req.Resolved)
	q := postgrest.FiltersToQueryValues(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, d.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	d.setRequestHeaders(httpReq, tableHeaderOptions(req.Resolved.Table.Options))
	httpReq.Header.Set("Content-Type", "application/json")
	if err := d.applyAuth(ctx, httpReq); err != nil {
		return 0, err
	}
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("http update %s: %s %s", path, resp.Status, string(rb))
	}
	return 1, nil
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	row := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	body, err := json.Marshal(row)
	if err != nil {
		return false, nil, err
	}
	path := d.tableResourcePath(req.Resolved)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, d.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return false, nil, err
	}
	d.setRequestHeaders(httpReq, tableHeaderOptions(req.Resolved.Table.Options))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	prefer := "return=minimal,resolution=merge-duplicates"
	if req.PreferRepresentation {
		prefer = "return=representation,resolution=merge-duplicates"
	}
	httpReq.Header.Set("Prefer", prefer)
	if err := d.applyAuth(ctx, httpReq); err != nil {
		return false, nil, err
	}
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return false, nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		created := resp.StatusCode == http.StatusCreated
		if req.PreferRepresentation && len(rb) > 0 {
			ret, err := parseOneRow(rb, req.Resolved.Columns)
			return created, ret, err
		}
		return created, nil, nil
	}
	if !isConflictStatus(resp.StatusCode, rb) {
		return false, nil, fmt.Errorf("http upsert %s: %s %s", path, resp.Status, string(rb))
	}
	n, err := d.Update(ctx, req)
	return false, nil, errIfZero(n, err)
}

func errIfZero(n int, err error) error {
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("upsert conflict but update affected 0 rows")
	}
	return nil
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	path := d.tableResourcePath(req.Resolved)
	q := postgrest.FiltersToQueryValues(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, d.baseURL+path, nil)
	if err != nil {
		return 0, err
	}
	d.setRequestHeaders(httpReq, tableHeaderOptions(req.Resolved.Table.Options))
	if err := d.applyAuth(ctx, httpReq); err != nil {
		return 0, err
	}
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("http delete %s: %s %s", path, resp.Status, string(rb))
	}
	return 1, nil
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	path := req.Resolved.Function.RemotePath
	if path == "" {
		return nil, fmt.Errorf("function remote_path required")
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if d.basePath != "" {
		path = "/" + strings.Trim(d.basePath, "/") + path
	}
	method := strings.ToUpper(strings.TrimSpace(req.Resolved.Function.Method))
	if method == "" {
		method = http.MethodPost
	}
	var bodyReader io.Reader
	if req.Body != nil && method != http.MethodGet && method != http.MethodHead {
		b, err := json.Marshal(req.Body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}
	fullURL := d.baseURL + path
	if len(req.Query) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, err
		}
		q := u.Query()
		for k, v := range req.Query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}
	d.setRequestHeaders(httpReq, functionHeaderOptions(req.Resolved.Function.Options))
	if bodyReader != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")
	if err := d.applyAuth(ctx, httpReq); err != nil {
		return nil, err
	}
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("http invoke %s: %s %s", path, resp.Status, string(rb))
	}
	if len(rb) == 0 {
		return nil, nil
	}
	var parsed any
	if err := json.Unmarshal(rb, &parsed); err != nil {
		return string(rb), nil
	}
	return parsed, nil
}

func parseJSONRows(body []byte) ([]map[string]any, error) {
	var arr []map[string]any
	if err := json.Unmarshal(body, &arr); err == nil {
		return arr, nil
	}
	var one map[string]any
	if err := json.Unmarshal(body, &one); err == nil {
		return []map[string]any{one}, nil
	}
	return nil, fmt.Errorf("invalid json response")
}

func parseOneRow(body []byte, cols []models.Column) (map[string]any, error) {
	rows, err := parseJSONRows(body)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, nil
	}
	mapped := postgrest.MapRowsFromRemote(rows, cols)
	return mapped[0], nil
}

func keyColumnSet(keys []string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out
}

func isConflictStatus(code int, body []byte) bool {
	if code == http.StatusConflict || code == http.StatusUnprocessableEntity {
		return true
	}
	s := strings.ToLower(string(body))
	return strings.Contains(s, "duplicate") || strings.Contains(s, "unique") || strings.Contains(s, "already exists")
}
