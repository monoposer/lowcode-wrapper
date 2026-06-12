package postgrest

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/monoposer/dataspan/internal/models"
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
	Select   []string
	Filters  []Filter
	OrGroups [][]Filter // OR of AND groups (from `or=(a.eq.1,b.eq.2)`)
	Order    []OrderSpec
	Limit    int
	Offset   int
}

type Prefer struct {
	Representation bool
	Upsert         bool
	CountExact     bool
}

func ParseQuery(values url.Values) Query {
	q := Query{}
	if sel := strings.TrimSpace(values.Get("select")); sel != "" {
		for _, part := range strings.Split(sel, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.Contains(part, "(") {
				continue // validated separately via ValidateSelect
			}
			q.Select = append(q.Select, part)
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
		"select": true, "order": true, "limit": true, "offset": true, "or": true, "and": true,
	}
	if rawOr := strings.TrimSpace(values.Get("or")); rawOr != "" {
		if groups, err := parseOrGroups(rawOr); err == nil {
			q.OrGroups = groups
		}
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

// ValidateSelect rejects embedded resource syntax not supported by dataspan.
func ValidateSelect(selects []string, rawSelect string) *APIError {
	if strings.Contains(rawSelect, "(") {
		return EmbedNotSupported()
	}
	for _, s := range selects {
		if strings.Contains(s, "(") {
			return EmbedNotSupported()
		}
	}
	return nil
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
		case "count=exact", "count=exact;":
			p.CountExact = true
		default:
			if strings.HasPrefix(part, "count=exact") {
				p.CountExact = true
			}
		}
	}
	return p
}

func parseOrGroups(raw string) ([][]Filter, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "(") && strings.HasSuffix(raw, ")") {
		raw = strings.TrimSpace(raw[1 : len(raw)-1])
	}
	if raw == "" {
		return nil, nil
	}
	var groups [][]Filter
	for _, part := range splitTopLevel(raw, ',') {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		f, err := parseFilterExpr(part)
		if err != nil {
			return nil, err
		}
		groups = append(groups, []Filter{f})
	}
	return groups, nil
}

func parseFilterExpr(expr string) (Filter, error) {
	expr = strings.TrimSpace(expr)
	for _, op := range []Op{OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte, OpLike, OpIn, OpIs} {
		marker := "." + string(op) + "."
		if idx := strings.Index(expr, marker); idx > 0 {
			return Filter{
				Column: expr[:idx],
				Op:     op,
				Value:  expr[idx+len(marker):],
			}, nil
		}
	}
	return Filter{}, fmt.Errorf("invalid filter expression %q", expr)
}

func splitTopLevel(s string, sep rune) []string {
	var parts []string
	depth := 0
	start := 0
	for i, r := range s {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if r == sep && depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
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
