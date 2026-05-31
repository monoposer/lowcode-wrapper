package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"lowcode-wrapper/internal/api"
	"lowcode-wrapper/internal/models"
	"lowcode-wrapper/internal/service"
	store "lowcode-wrapper/internal/store/postgres"

	"github.com/google/uuid"

	_ "lowcode-wrapper/internal/driver/httpdriver"
)

type httpE2EEnv struct {
	baseURL   string
	tableName string
}

func setupHTTPE2E(t *testing.T, st *store.Store, serverOpts json.RawMessage, credRef *uuid.UUID) *httpE2EEnv {
	t.Helper()

	serverName := fmt.Sprintf("mock_http_%d", time.Now().UnixNano())
	tableName := fmt.Sprintf("orders_%d", time.Now().UnixNano())
	srv, err := st.CreateServer(context.Background(), models.CreateServerRequest{
		Name:          serverName,
		Protocol:      models.ProtocolHTTP,
		Options:       serverOpts,
		CredentialRef: credRef,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = st.CreateTable(context.Background(), models.CreateTableRequest{
		ServerName: srv.Name,
		SchemaName: "public",
		TableName:  tableName,
		RemoteName: "orders",
		KeyColumns: []string{"id"},
		Options:    json.RawMessage(`{"headers":{"X-Table":"orders"}}`),
		Columns: []models.ColumnInput{
			{Name: "id", DataType: "text"},
			{Name: "amount", DataType: "text", RemoteName: "total_amount"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	engine := service.NewEngine(st)
	mux := http.NewServeMux()
	api.NewPostgRESTHandler(engine).Register(mux)
	ts := httptest.NewServer(api.CORS(mux))
	t.Cleanup(ts.Close)

	return &httpE2EEnv{baseURL: ts.URL, tableName: tableName}
}

func (e *httpE2EEnv) tablePath() string {
	return "/v1/public/" + e.tableName
}

func TestHTTPDriverE2E(t *testing.T) {
	mockTS, mock := newPostgRESTMock()
	t.Cleanup(mockTS.Close)

	serverOpts, _ := json.Marshal(models.HTTPServerOptions{
		Endpoint: mockTS.URL,
		Headers:  map[string]string{"X-Tenant": "test-tenant"},
	})
	st := initIntegrationEnv(t)
	env := setupHTTPE2E(t, st, serverOpts, nil)

	t.Run("SelectList", func(t *testing.T) {
		resp := mustGET(t, env.baseURL+env.tablePath())
		defer resp.Body.Close()
		var rows []map[string]any
		decodeJSON(t, resp, &rows)
		if len(rows) != 2 {
			t.Fatalf("rows len = %d, want 2", len(rows))
		}
		if rows[0]["amount"] != "99" {
			t.Fatalf("local column mapping: %+v", rows[0])
		}
		last := mock.last()
		if last.Method != http.MethodGet || last.Path != "/orders" {
			t.Fatalf("upstream %s %s", last.Method, last.Path)
		}
		if last.Headers.Get("X-Tenant") != "test-tenant" || last.Headers.Get("X-Table") != "orders" {
			t.Fatalf("headers: %+v", last.Headers)
		}
	})

	t.Run("SelectFilterMapsRemoteColumn", func(t *testing.T) {
		resp := mustGET(t, env.baseURL+env.tablePath()+"?amount=eq.99")
		defer resp.Body.Close()
		var rows []map[string]any
		decodeJSON(t, resp, &rows)
		if len(rows) != 1 || rows[0]["id"] != "1" {
			t.Fatalf("rows: %+v", rows)
		}
		if mock.last().RawQuery != "total_amount=eq.99" {
			t.Fatalf("upstream query = %q", mock.last().RawQuery)
		}
	})

	t.Run("SelectWithSelectParam", func(t *testing.T) {
		resp := mustGET(t, env.baseURL+env.tablePath()+"?select=id,amount&limit=10")
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
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
		resp := mustPOST(t, env.baseURL+env.tablePath(), `{"id":"3","amount":"10"}`)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status %d body %s", resp.StatusCode, body)
		}
		last := mock.last()
		if last.Method != http.MethodPost {
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
		req, _ := http.NewRequest(http.MethodPatch, env.baseURL+env.tablePath()+"?amount=eq.50", bytes.NewBufferString(`{"amount":"55"}`))
		req.Header.Set("Content-Type", "application/json")
		resp := mustDo(t, req)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status %d body %s", resp.StatusCode, body)
		}
		last := mock.last()
		if last.Method != http.MethodPatch || last.RawQuery != "total_amount=eq.50" {
			t.Fatalf("upstream %s ?%s body %s", last.Method, last.RawQuery, last.Body)
		}
		var sent map[string]any
		_ = json.Unmarshal([]byte(last.Body), &sent)
		if sent["total_amount"] != "55" {
			t.Fatalf("patch body: %+v", sent)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodDelete, env.baseURL+env.tablePath()+"?id=eq.1", nil)
		resp := mustDo(t, req)
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status %d body %s", resp.StatusCode, body)
		}
		last := mock.last()
		if last.Method != http.MethodDelete || last.RawQuery != "id=eq.1" {
			t.Fatalf("upstream %s ?%s", last.Method, last.RawQuery)
		}
	})
}

func TestHTTPDriverE2E_APIKeyAuth(t *testing.T) {
	mockTS, mock := newPostgRESTMock()
	t.Cleanup(mockTS.Close)

	st := initIntegrationEnv(t)

	cred, err := st.CreateCredential(context.Background(), fmt.Sprintf("http_cred_%d", time.Now().UnixNano()), map[string]any{
		"apiKey": "secret-key-123",
	})
	if err != nil {
		t.Fatal(err)
	}

	serverOpts, _ := json.Marshal(models.HTTPServerOptions{
		Endpoint: mockTS.URL,
		Auth: &models.HTTPAuth{
			Type:    models.AuthAPIKey,
			Options: map[string]any{"label": "X-Api-Key"},
		},
	})
	env := setupHTTPE2E(t, st, serverOpts, &cred.ID)

	resp := mustGET(t, env.baseURL+env.tablePath()+"?id=eq.1")
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	if got := mock.last().Headers.Get("X-Api-Key"); got != "secret-key-123" {
		t.Fatalf("X-Api-Key = %q", got)
	}
}

func mustGET(t *testing.T, url string) *http.Response {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustPOST(t *testing.T, url, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func mustDo(t *testing.T, req *http.Request) *http.Response {
	t.Helper()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode >= 400 {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	if err := json.Unmarshal(body, dst); err != nil {
		t.Fatalf("decode: %v body %s", err, body)
	}
}
