// Package e2e runs black-box HTTP tests against the dataspan API.
// Cases are defined in e2e/testdata/cases.yaml; data in e2e/testdata/data/.
package e2e_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/monoposer/dataspan/internal/api"
	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/service"
	"github.com/monoposer/dataspan/internal/store"

	_ "github.com/monoposer/dataspan/internal/driver/file"
)

type caseSuite struct {
	Cases     []httpCase `yaml:"cases"`
	AuthCases []httpCase `yaml:"auth_cases"`
}

type httpCase struct {
	Name       string         `yaml:"name"`
	Method     string         `yaml:"method"`
	Path       string         `yaml:"path"`
	Query      string         `yaml:"query"`
	Body       string         `yaml:"body"`
	Status     int            `yaml:"status"`
	JSON       any            `yaml:"json"`
	JSONHas    map[string]any `yaml:"json_has"`
	JSONMinLen int            `yaml:"json_min_len"`
	JSONCode   string         `yaml:"json_code"`
	APIKey     string         `yaml:"apikey"`
	Bearer     string         `yaml:"bearer"`
}

func TestE2E(t *testing.T) {
	dir := e2eDir(t)
	suite := loadSuite(t, filepath.Join(dir, "testdata/cases.yaml"))

	srv := startServer(t, serverOpts{fixtureRoot: filepath.Join(dir, "testdata")})
	defer srv.Close()

	for _, c := range suite.Cases {
		t.Run(c.Name, func(t *testing.T) {
			doCase(t, srv, c)
		})
	}
}

func TestE2EAuth(t *testing.T) {
	dir := e2eDir(t)
	suite := loadSuite(t, filepath.Join(dir, "testdata/cases.yaml"))

	srv := startServer(t, serverOpts{
		fixtureRoot: filepath.Join(dir, "testdata"),
		anonKey:     "test-anon-key",
	})
	defer srv.Close()

	for _, c := range suite.AuthCases {
		t.Run(c.Name, func(t *testing.T) {
			doCase(t, srv, c)
		})
	}
}

type serverOpts struct {
	fixtureRoot string
	anonKey     string
}

func startServer(t *testing.T, opts serverOpts) *httptest.Server {
	t.Helper()

	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAB
	}
	t.Setenv("DATASPAN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	t.Setenv("WRAPPER_STORE_MODE", "file")
	if opts.anonKey != "" {
		t.Setenv("DATASPAN_ANON_KEY", opts.anonKey)
	} else {
		t.Setenv("DATASPAN_ANON_KEY", "")
	}

	dataDir := filepath.Join(opts.fixtureRoot, "data")
	driversPath := filepath.Join(t.TempDir(), "drivers.yaml")
	if err := os.WriteFile(driversPath, []byte(strings.TrimSpace(`
servers:
  - name: local_files
    protocol: file
    enabled: true
    options:
      rootPath: `+jsonString(dataDir)+`

tables:
  - server: local_files
    schema: public
    name: items
    remoteName: items.csv
    keyColumns: [id]
    options:
      format: csv
    columns:
      - name: id
        dataType: text
      - name: name
        dataType: text
`)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WRAPPER_DRIVERS_FILE", driversPath)

	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.NewFromEnv(vault)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	engine := service.NewEngine(st)
	gateway := auth.NewGatewayFromEnv()

	mux := http.NewServeMux()
	api.NewAdminHandler(st).Register(mux)
	api.NewPostgRESTHandler(engine).Register(mux)

	handler := api.CORS(api.DataAPIAuth(gateway, mux))
	return httptest.NewServer(handler)
}

func doCase(t *testing.T, srv *httptest.Server, c httpCase) {
	t.Helper()

	method := c.Method
	if method == "" {
		method = http.MethodGet
	}
	url := srv.URL + c.Path
	if q := strings.TrimSpace(c.Query); q != "" {
		if !strings.HasPrefix(q, "?") {
			q = "?" + q
		}
		url += q
	}

	var body io.Reader
	if strings.TrimSpace(c.Body) != "" {
		body = strings.NewReader(c.Body)
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")
	if c.Body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.APIKey != "" {
		req.Header.Set("apikey", c.APIKey)
	}
	if c.Bearer != "" {
		req.Header.Set("Authorization", "Bearer "+c.Bearer)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != c.Status {
		t.Fatalf("status=%d want=%d body=%s", res.StatusCode, c.Status, string(raw))
	}

	if c.JSONCode != "" {
		var errBody struct {
			Code string `json:"code"`
		}
		if err := json.Unmarshal(raw, &errBody); err != nil {
			t.Fatalf("decode error body: %v raw=%s", err, string(raw))
		}
		if errBody.Code != c.JSONCode {
			t.Fatalf("code=%q want=%q body=%s", errBody.Code, c.JSONCode, string(raw))
		}
		return
	}

	if len(c.JSONHas) > 0 {
		var got map[string]any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode response: %v raw=%s", err, string(raw))
		}
		for k, want := range c.JSONHas {
			if fmt.Sprint(got[k]) != fmt.Sprint(want) {
				t.Fatalf("json_has[%q]=%v want=%v body=%s", k, got[k], want, string(raw))
			}
		}
	}

	if c.JSON != nil {
		want, err := json.Marshal(c.JSON)
		if err != nil {
			t.Fatal(err)
		}
		var got, expect any
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("decode response: %v raw=%s", err, string(raw))
		}
		if err := json.Unmarshal(want, &expect); err != nil {
			t.Fatal(err)
		}
		gotNorm, _ := json.Marshal(got)
		expectNorm, _ := json.Marshal(expect)
		if !bytes.Equal(gotNorm, expectNorm) {
			t.Fatalf("body mismatch\ngot:  %s\nwant: %s", prettyJSON(gotNorm), prettyJSON(expectNorm))
		}
	}

	if c.JSONMinLen > 0 {
		var arr []any
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Fatalf("expected JSON array: %v raw=%s", err, string(raw))
		}
		if len(arr) < c.JSONMinLen {
			t.Fatalf("len=%d want>=%d body=%s", len(arr), c.JSONMinLen, string(raw))
		}
	}
}

func loadSuite(t *testing.T, path string) caseSuite {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var suite caseSuite
	if err := yaml.Unmarshal(raw, &suite); err != nil {
		t.Fatal(err)
	}
	return suite
}

func e2eDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func prettyJSON(b []byte) string {
	var buf bytes.Buffer
	_ = json.Indent(&buf, b, "", "  ")
	return buf.String()
}
