package httpdriver

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"lowcode-wrapper/internal/driver"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/postgrest"
)

func testResolvedTable() *models.ResolvedTable {
	return &models.ResolvedTable{
		Table: models.Table{
			SchemaName: "public",
			RemoteName: "orders",
			KeyColumns: []string{"id"},
			Options:    json.RawMessage(`{"headers":{"X-Table":"orders"}}`),
		},
		Columns: []models.Column{
			{Name: "id", DataType: "text"},
			{Name: "amount", DataType: "text", RemoteName: "total_amount"},
		},
	}
}

func newTestDriver(t *testing.T, endpoint string, cred map[string]any, auth *models.HTTPAuth) *Driver {
	t.Helper()
	opts, err := json.Marshal(models.HTTPServerOptions{
		Endpoint: endpoint,
		Headers:  map[string]string{"X-Tenant": "test-tenant"},
		Auth:     auth,
	})
	if err != nil {
		t.Fatal(err)
	}
	drv, err := New(context.Background(), models.Server{
		Name:     "test",
		Protocol: models.ProtocolHTTP,
		Options:  opts,
	}, cred)
	if err != nil {
		t.Fatal(err)
	}
	return drv.(*Driver)
}

func TestDriverCRUD(t *testing.T) {
	mockTS, mock := newPostgRESTMock()
	t.Cleanup(mockTS.Close)

	drv := newTestDriver(t, mockTS.URL, nil, nil)
	resolved := testResolvedTable()
	ctx := context.Background()

	t.Run("SelectList", func(t *testing.T) {
		rows, err := drv.Select(ctx, driver.SelectRequest{Resolved: resolved})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("rows len = %d, want 2", len(rows))
		}
		if rows[0]["amount"] != "99" {
			t.Fatalf("local column mapping: %+v", rows[0])
		}
		last := mock.last()
		if last.Method != "GET" || last.Path != "/orders" {
			t.Fatalf("upstream %s %s", last.Method, last.Path)
		}
		if last.Headers.Get("X-Tenant") != "test-tenant" || last.Headers.Get("X-Table") != "orders" {
			t.Fatalf("headers: %+v", last.Headers)
		}
	})

	t.Run("SelectFilterMapsRemoteColumn", func(t *testing.T) {
		rows, err := drv.Select(ctx, driver.SelectRequest{
			Resolved: resolved,
			Filters:  []postgrest.Filter{{Column: "amount", Op: postgrest.OpEq, Value: "99"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0]["id"] != "1" {
			t.Fatalf("rows: %+v", rows)
		}
		if mock.last().RawQuery != "total_amount=eq.99" {
			t.Fatalf("upstream query = %q", mock.last().RawQuery)
		}
	})

	t.Run("SelectWithSelectParam", func(t *testing.T) {
		_, err := drv.Select(ctx, driver.SelectRequest{
			Resolved: resolved,
			Select:   []string{"id", "amount"},
			Limit:    10,
		})
		if err != nil {
			t.Fatal(err)
		}
		q, err := url.ParseQuery(mock.last().RawQuery)
		if err != nil {
			t.Fatal(err)
		}
		if q.Get("select") != "id,amount" || q.Get("limit") != "10" {
			t.Fatalf("upstream query = %v", q)
		}
	})

	t.Run("Insert", func(t *testing.T) {
		_, err := drv.Insert(ctx, driver.RowRequest{
			Resolved: resolved,
			Row:      map[string]any{"id": "3", "amount": "10"},
		})
		if err != nil {
			t.Fatal(err)
		}
		last := mock.last()
		if last.Method != "POST" {
			t.Fatalf("method %s", last.Method)
		}
		var sent map[string]any
		if err := json.Unmarshal([]byte(last.Body), &sent); err != nil {
			t.Fatal(err)
		}
		if sent["total_amount"] != "10" {
			t.Fatalf("remote body: %+v", sent)
		}
	})

	t.Run("Update", func(t *testing.T) {
		n, err := drv.Update(ctx, driver.RowRequest{
			Resolved: resolved,
			Row:      map[string]any{"amount": "55"},
			Filters:  []postgrest.Filter{{Column: "amount", Op: postgrest.OpEq, Value: "50"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("affected = %d", n)
		}
		last := mock.last()
		if last.Method != "PATCH" || last.RawQuery != "total_amount=eq.50" {
			t.Fatalf("upstream %s ?%s body %s", last.Method, last.RawQuery, last.Body)
		}
		var sent map[string]any
		_ = json.Unmarshal([]byte(last.Body), &sent)
		if sent["total_amount"] != "55" {
			t.Fatalf("patch body: %+v", sent)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		n, err := drv.Delete(ctx, driver.DeleteRequest{
			Resolved: resolved,
			Filters:  []postgrest.Filter{{Column: "id", Op: postgrest.OpEq, Value: "1"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("affected = %d", n)
		}
		last := mock.last()
		if last.Method != "DELETE" || last.RawQuery != "id=eq.1" {
			t.Fatalf("upstream %s ?%s", last.Method, last.RawQuery)
		}
	})
}

func TestDriverAPIKeyAuth(t *testing.T) {
	mockTS, mock := newPostgRESTMock()
	t.Cleanup(mockTS.Close)

	drv := newTestDriver(t, mockTS.URL, map[string]any{"apiKey": "secret-key-123"}, &models.HTTPAuth{
		Type:    models.AuthAPIKey,
		Options: map[string]any{"label": "X-Api-Key"},
	})
	resolved := testResolvedTable()

	_, err := drv.Select(context.Background(), driver.SelectRequest{
		Resolved: resolved,
		Filters:  []postgrest.Filter{{Column: "id", Op: postgrest.OpEq, Value: "1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := mock.last().Headers.Get("X-Api-Key"); got != "secret-key-123" {
		t.Fatalf("X-Api-Key = %q", got)
	}
}
