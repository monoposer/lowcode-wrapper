package httpwrap

import (
	"encoding/json"
	"testing"

	"github.com/monoposer/dataspan/internal/models"
)

func TestMergeHTTPServerOptions(t *testing.T) {
	raw, err := MergeHTTPServerOptions(
		json.RawMessage(`{"endpoint":"https://custom.example","headers":{"X-A":"1"}}`),
		models.HTTPServerOptions{
			Endpoint: "https://default.example",
			Headers:  map[string]string{"Notion-Version": "2022-06-28"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var got models.HTTPServerOptions
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Endpoint != "https://custom.example" {
		t.Fatalf("endpoint = %q", got.Endpoint)
	}
	if got.Headers["Notion-Version"] != "2022-06-28" || got.Headers["X-A"] != "1" {
		t.Fatalf("headers = %#v", got.Headers)
	}
}
