package gentypes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestMetaClientFetch(t *testing.T) {
	tableID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/admin/api/tables":
			_, _ = w.Write([]byte(`[{"id":"11111111-1111-1111-1111-111111111111","schemaName":"public","tableName":"orders","keyColumns":["id"],"serverName":"api"}]`))
		case "/admin/api/tables/" + tableID.String() + "/columns":
			_, _ = w.Write([]byte(`[{"name":"id","dataType":"text","nullable":false,"position":1}]`))
		case "/admin/api/functions":
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := &MetaClient{BaseURL: srv.URL}
	snap, err := client.Fetch(context.Background(), []string{"public"})
	if err != nil {
		t.Fatal(err)
	}
	st := snap.Schemas["public"]
	if len(st.Tables) != 1 || len(st.Tables["orders"].Columns) != 1 {
		t.Fatalf("snap=%+v", snap)
	}
	raw, _ := json.Marshal(st.Tables["orders"].Columns[0])
	if string(raw) == "" {
		t.Fatal("expected column")
	}
}
