package file_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"lowcode-wrapper/internal/store/file"
)

func TestDeclarativeInlineCredential(t *testing.T) {
	t.Setenv("PARTNER_API_KEY", "secret-from-env")
	path := filepath.Join(t.TempDir(), "drivers.yaml")
	content := `
servers:
  - name: api
    protocol: http
    credential:
      apiKey: ${PARTNER_API_KEY}
    options:
      endpoint: https://example.com
tables:
  - server: api
    name: orders
    keyColumns: [id]
    columns:
      - name: id
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	st, err := file.New(testVault(t), file.Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	srvs, err := st.ListServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(srvs) != 1 || srvs[0].Name != "api" {
		t.Fatalf("servers: %+v", srvs)
	}
	if srvs[0].CredentialRef == nil {
		t.Fatal("expected inline credential to produce credentialRef")
	}

	cred, err := st.ResolveCredential(context.Background(), *srvs[0].CredentialRef)
	if err != nil {
		t.Fatal(err)
	}
	if cred["apiKey"] != "secret-from-env" {
		t.Fatalf("cred: %+v", cred)
	}
}

func TestDeclarativeYAMLLoadLegacyNamedCredential(t *testing.T) {
	t.Setenv("PARTNER_API_KEY", "secret-from-env")
	path := filepath.Join(t.TempDir(), "drivers.yaml")
	content := `
credentials:
  - name: partner
    data:
      apiKey: ${PARTNER_API_KEY}
servers:
  - name: api
    protocol: http
    credential: partner
    options:
      endpoint: https://example.com
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	st, err := file.New(testVault(t), file.Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)

	srvs, err := st.ListServers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(srvs) != 1 {
		t.Fatalf("servers: %+v", srvs)
	}
}
