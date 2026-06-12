package postgrest_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/monoposer/dataspan/internal/auth"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/postgrest"
	"github.com/monoposer/dataspan/internal/store/file"
)

func testFileStore(t *testing.T) *file.Store {
	t.Helper()
	key := make([]byte, 32)
	t.Setenv("DATASPAN_VAULT_KEY", base64.StdEncoding.EncodeToString(key))
	v, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st, err := file.New(v, file.Config{Path: filepath.Join(t.TempDir(), "meta.yaml")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}

func TestBuildOpenAPI(t *testing.T) {
	st := testFileStore(t)
	_, err := st.CreateServer(context.Background(), models.CreateServerRequest{
		Name:     "api",
		Protocol: models.ProtocolHTTP,
		Options:  json.RawMessage(`{"endpoint":"https://example.com"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = st.CreateTable(context.Background(), models.CreateTableRequest{
		ServerName: "api",
		SchemaName: "public",
		TableName:  "orders",
		KeyColumns: []string{"id"},
		Columns: []models.ColumnInput{
			{Name: "id", DataType: "text"},
			{Name: "amount", DataType: "numeric", RemoteName: "total"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.CreateFunction(context.Background(), models.CreateFunctionRequest{
		ServerName: "api",
		Name:       "ship",
		Operation:  "invoke",
		Method:     "POST",
	})
	if err != nil {
		t.Fatal(err)
	}

	spec, err := postgrest.BuildOpenAPI(context.Background(), st, postgrest.OpenAPIOptions{
		Host:     "localhost:3020",
		BasePath: "/rest/v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if spec["openapi"] != "3.1.0" {
		t.Fatalf("openapi=%v", spec["openapi"])
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatal("missing paths")
	}
	if _, ok := paths["/orders"]; !ok {
		t.Fatalf("missing /orders path: %v", paths)
	}
	if _, ok := paths["/rpc/ship"]; !ok {
		t.Fatalf("missing rpc path: %v", paths)
	}
	components, ok := spec["components"].(map[string]any)
	if !ok {
		t.Fatal("missing components")
	}
	schemas, ok := components["schemas"].(map[string]any)
	if !ok {
		t.Fatal("missing components.schemas")
	}
	orders, ok := schemas["orders"].(map[string]any)
	if !ok {
		t.Fatal("missing orders schema")
	}
	props, ok := orders["properties"].(map[string]any)
	if !ok || props["amount"] == nil {
		t.Fatalf("orders properties=%v", orders)
	}
}
