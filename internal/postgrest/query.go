package postgrest

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"lowcode-wrapper/internal/models"
)

type Op string

const (
	OpEq   Op = "eq"
	OpNeq  Op = "neq"
	OpGt   Op = "gt"
	OpGte  Op = "gte"
	OpLt   Op = "lt"
	OpLte  Op = "lte"
	OpLike Op = "like"
	OpIn   Op = "in"
	OpIs   Op = "is"
)

type Filter struct {
	Column string
	Op     Op
	Value  string
}

type OrderSpec struct {
	Column string
	Desc   bool
}

type Query struct {
	Select  []string
	Filters []Filter
	Order   []OrderSpec
	Limit   int
	Offset  int
}

type Prefer struct {
	Representation bool
	Upsert         bool
}

func ParseQuery(values url.Values) Query {
	q := Query{}
	if sel := strings.TrimSpace(values.Get("select")); sel != "" {
		for _, part := range strings.Split(sel, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				q.Select = append(q.Select, part)
			}
		}
	}
	if lim := strings.TrimSpace(values.Get("limit")); lim != "" {
		if n, err := strconv.Atoi(lim); err == nil && n >= 0 {
			q.Limit = n
		}
	}
	if off := strings.TrimSpace(values.Get("offset")); off != "" {
		if n, err := strconv.Atoi(off); err == nil && n >= 0 {
			q.Offset = n
		}
	}
	if ord := strings.TrimSpace(values.Get("order")); ord != "" {
		for _, part := range strings.Split(ord, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			spec := OrderSpec{Column: part}
			if strings.HasSuffix(part, ".desc") {
				spec.Column = strings.TrimSuffix(part, ".desc")
				spec.Desc = true
			} else if strings.HasSuffix(part, ".asc") {
				spec.Column = strings.TrimSuffix(part, ".asc")
			}
			spec.Column = strings.TrimSpace(spec.Column)
			if spec.Column != "" {
				q.Order = append(q.Order, spec)
			}
		}
	}
	reserved := map[string]bool{
		"select": true, "order": true, "limit": true, "offset": true,
	}
	for key, vals := range values {
		if reserved[key] || len(vals) == 0 {
			continue
		}
		if f := parseFilter(key, vals[0]); f != nil {
			q.Filters = append(q.Filters, *f)
		}
	}
	return q
}

func parseFilter(column, raw string) *Filter {
	dot := strings.Index(raw, ".")
	if dot <= 0 {
		return &Filter{Column: column, Op: OpEq, Value: raw}
	}
	op := Op(raw[:dot])
	val := raw[dot+1:]
	switch op {
	case OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpLike, OpIn, OpIs:
		return &Filter{Column: column, Op: op, Value: val}
	default:
		return &Filter{Column: column, Op: OpEq, Value: raw}
	}
}

func ParsePrefer(h http.Header) Prefer {
	p := Prefer{}
	for _, part := range strings.Split(h.Get("Prefer"), ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		switch part {
		case "return=representation":
			p.Representation = true
		case "resolution=merge-duplicates":
			p.Upsert = true
		}
	}
	return p
}

func FiltersToQueryValues(filters []Filter) url.Values {
	q := url.Values{}
	for _, f := range filters {
		q.Set(f.Column, string(f.Op)+"."+f.Value)
	}
	return q
}

func MapRowToRemote(row map[string]any, cols []models.Column) map[string]any {
	out := make(map[string]any, len(row))
	colMap := make(map[string]string, len(cols))
	for _, c := range cols {
		colMap[c.Name] = models.RemoteColumnName(c)
	}
	for k, v := range row {
		if remote, ok := colMap[k]; ok {
			out[remote] = v
		} else {
			out[k] = v
		}
	}
	return out
}

func MapFilters(filters []Filter, cols []models.Column) []Filter {
	if len(filters) == 0 {
		return filters
	}
	localToRemote := make(map[string]string, len(cols))
	for _, c := range cols {
		localToRemote[c.Name] = models.RemoteColumnName(c)
	}
	out := make([]Filter, len(filters))
	for i, f := range filters {
		col := f.Column
		if remote, ok := localToRemote[col]; ok {
			col = remote
		}
		out[i] = Filter{Column: col, Op: f.Op, Value: f.Value}
	}
	return out
}

func MapRowsFromRemote(rows []map[string]any, cols []models.Column) []map[string]any {
	remoteToLocal := make(map[string]string, len(cols))
	for _, c := range cols {
		remoteToLocal[models.RemoteColumnName(c)] = c.Name
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		mapped := make(map[string]any, len(row))
		for k, v := range row {
			if local, ok := remoteToLocal[k]; ok {
				mapped[local] = v
			} else {
				mapped[k] = v
			}
		}
		out[i] = mapped
	}
	return out
}
