package rest

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/monoposer/dataspan/internal/httpx"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func decodeInsertBody(r *http.Request) (rows []map[string]any, single bool, err error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, false, err
	}
	if len(body) == 0 {
		return []map[string]any{{}}, true, nil
	}
	if body[0] == '[' {
		if err := json.Unmarshal(body, &rows); err != nil {
			return nil, false, err
		}
		return rows, false, nil
	}
	var one map[string]any
	if err := json.Unmarshal(body, &one); err != nil {
		return nil, false, err
	}
	return []map[string]any{one}, true, nil
}

func writeSelectResult(w http.ResponseWriter, r *http.Request, rows []map[string]any, q postgrest.Query, prefer postgrest.Prefer, total int) error {
	if rows == nil {
		rows = []map[string]any{}
	}
	mode := postgrest.ParseObjectMode(r.Header)

	switch mode {
	case postgrest.ObjectModeSingle:
		if len(rows) == 0 {
			return postgrest.SingleNotFound()
		}
		if len(rows) > 1 {
			return postgrest.MultipleRows()
		}
	case postgrest.ObjectModeMaybeSingle:
		if len(rows) > 1 {
			return postgrest.MultipleRows()
		}
	}

	if prefer.CountExact {
		if total < 0 {
			total = len(rows)
		}
		end := q.Offset + len(rows) - 1
		if len(rows) == 0 {
			end = q.Offset
		}
		if end < q.Offset {
			end = q.Offset
		}
		w.Header().Set("Content-Range", postgrest.FormatContentRange(q.Offset, end, total))
	}

	w.WriteHeader(http.StatusOK)

	switch mode {
	case postgrest.ObjectModeSingle, postgrest.ObjectModeMaybeSingle:
		w.Header().Set("Content-Type", postgrest.MediaObjectJSON)
		if mode == postgrest.ObjectModeMaybeSingle && len(rows) == 0 {
			return json.NewEncoder(w).Encode(nil)
		}
		return json.NewEncoder(w).Encode(rows[0])
	default:
		w.Header().Set("Content-Type", postgrest.MediaArrayJSON)
		return json.NewEncoder(w).Encode(rows)
	}
}

func writeMutationNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func writeMutationRows(w http.ResponseWriter, rows []map[string]any) {
	if rows == nil {
		rows = []map[string]any{}
	}
	httpx.WriteJSON(w, http.StatusOK, rows)
}
