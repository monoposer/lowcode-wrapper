package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/models"
)

func (d *Driver) setRequestHeaders(req *http.Request, extra ...map[string]string) {
	applyHeaderMaps(req, append([]map[string]string{d.headers}, extra...)...)
}

func applyHeaderMaps(req *http.Request, layers ...map[string]string) {
	for _, m := range layers {
		for k, v := range m {
			k = strings.TrimSpace(k)
			v = strings.TrimSpace(v)
			if k == "" || v == "" {
				continue
			}
			req.Header.Set(k, v)
		}
	}
}

func tableHeaderOptions(raw json.RawMessage) map[string]string {
	opts, err := models.ParseServerOptions[models.HTTPTableOptions](raw)
	if err != nil {
		return nil
	}
	return opts.Headers
}

func functionHeaderOptions(raw json.RawMessage) map[string]string {
	opts, err := models.ParseServerOptions[models.HTTPFunctionOptions](raw)
	if err != nil {
		return nil
	}
	return opts.Headers
}
