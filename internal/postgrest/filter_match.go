package postgrest

import (
	"fmt"
	"strings"
)

// MatchQuery returns true when row satisfies AND filters and any OR group.
func MatchQuery(row map[string]any, filters []Filter, orGroups [][]Filter) bool {
	if len(orGroups) > 0 {
		for _, group := range orGroups {
			if MatchFilters(row, group) {
				return true
			}
		}
		return false
	}
	return MatchFilters(row, filters)
}

// MatchFilters requires all filters to match (AND).
func MatchFilters(row map[string]any, filters []Filter) bool {
	for _, f := range filters {
		if !matchFilter(row, f) {
			return false
		}
	}
	return true
}

func matchFilter(row map[string]any, f Filter) bool {
	val, ok := row[f.Column]
	sval := fmt.Sprint(val)
	switch f.Op {
	case OpEq:
		return ok && sval == f.Value
	case OpNeq:
		return !ok || sval != f.Value
	case OpGt:
		return ok && sval > f.Value
	case OpGte:
		return ok && sval >= f.Value
	case OpLt:
		return ok && sval < f.Value
	case OpLte:
		return ok && sval <= f.Value
	case OpLike:
		return ok && strings.Contains(sval, strings.Trim(f.Value, "%"))
	case OpIn:
		for _, part := range strings.Split(f.Value, ",") {
			if ok && sval == strings.TrimSpace(part) {
				return true
			}
		}
		return false
	case OpIs:
		if f.Value == "null" {
			return !ok || val == nil
		}
		return ok && sval == f.Value
	default:
		return ok && sval == f.Value
	}
}
