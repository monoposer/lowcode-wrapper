package postgrest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/monoposer/dataspan/internal/errkind"
	"github.com/monoposer/dataspan/internal/store/errs"
)

// APIError is the PostgREST / Supabase Data API error body.
// See https://postgrest.org/en/stable/references/errors.html
type APIError struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Details *string `json:"details"`
	Hint    *string `json:"hint"`
	Status  int     `json:"-"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

// ErrorContext carries request metadata for error mapping.
type ErrorContext struct {
	Schema string
	Table  string
	RPC    string
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TableNotFound(schema, table string) *APIError {
	schema = fallback(schema, "public")
	table = fallback(table, "unknown")
	return &APIError{
		Code:    "PGRST205",
		Message: fmt.Sprintf("Could not find the table '%s.%s' in the schema cache", schema, table),
		Details: nil,
		Hint:    nil,
		Status:  http.StatusNotFound,
	}
}

func RPCNotFound(schema, name string) *APIError {
	schema = fallback(schema, "public")
	return &APIError{
		Code:    "PGRST202",
		Message: fmt.Sprintf("Could not find the function %s.%s in the schema cache", schema, name),
		Details: nil,
		Hint:    nil,
		Status:  http.StatusNotFound,
	}
}

func InvalidBody(err error) *APIError {
	msg := "Invalid request body"
	if err != nil {
		msg = err.Error()
	}
	return &APIError{
		Code:    "PGRST102",
		Message: msg,
		Details: nil,
		Hint:    nil,
		Status:  http.StatusBadRequest,
	}
}

func UnsupportedMethod(method string) *APIError {
	return &APIError{
		Code:    "PGRST117",
		Message: fmt.Sprintf("Unsupported HTTP method: %s", method),
		Details: nil,
		Hint:    nil,
		Status:  http.StatusMethodNotAllowed,
	}
}

func InvalidRPCMethod() *APIError {
	return &APIError{
		Code:    "PGRST101",
		Message: "Functions only support GET and POST verbs",
		Details: nil,
		Hint:    nil,
		Status:  http.StatusMethodNotAllowed,
	}
}

func AuthError(message string) *APIError {
	return &APIError{
		Code:    "PGRST301",
		Message: message,
		Details: nil,
		Hint:    strPtr("Provide a valid apikey header and Authorization: Bearer <anon-key or JWT>"),
		Status:  http.StatusUnauthorized,
	}
}

func InvalidPath() *APIError {
	return &APIError{
		Code:    "PGRST125",
		Message: "Invalid path specified in request URL",
		Details: nil,
		Hint:    nil,
		Status:  http.StatusNotFound,
	}
}

func SingleNotFound() *APIError {
	return &APIError{
		Code:    "PGRST116",
		Message: "Cannot coerce the result to a single JSON object",
		Details: strPtr("The result contains 0 rows"),
		Hint:    nil,
		Status:  http.StatusNotFound,
	}
}

func MultipleRows() *APIError {
	return &APIError{
		Code:    "PGRST116",
		Message: "Cannot coerce the result to a single JSON object",
		Details: strPtr("The result contains more than 1 row"),
		Hint:    nil,
		Status:  http.StatusNotFound,
	}
}

func EmbedNotSupported() *APIError {
	return &APIError{
		Code:    "PGRST200",
		Message: "Embedded resources are not supported",
		Details: nil,
		Hint:    strPtr("Use flat column names in select= without nested ( ) syntax"),
		Status:  http.StatusBadRequest,
	}
}

func UnsupportedOperation(msg string) *APIError {
	return &APIError{
		Code:    "PGRST000",
		Message: msg,
		Details: nil,
		Hint:    nil,
		Status:  http.StatusBadRequest,
	}
}

// DriverOperationNotSupported maps unsupported driver operations to PostgREST-shaped errors.
func DriverOperationNotSupported(op, protocol string) *APIError {
	status := http.StatusBadRequest
	switch op {
	case "insert", "update", "delete", "upsert":
		status = http.StatusMethodNotAllowed
	}
	return &APIError{
		Code:    "PGRST000",
		Message: fmt.Sprintf("%s is not supported for protocol %q", op, protocol),
		Details: nil,
		Hint:    strPtr("See /rest/v1/ OpenAPI and docs/drivers/ for protocol capabilities"),
		Status:  status,
	}
}

func InternalError(err error) *APIError {
	msg := "Internal server error"
	if err != nil {
		msg = err.Error()
	}
	return &APIError{
		Code:    "PGRST000",
		Message: msg,
		Details: nil,
		Hint:    nil,
		Status:  http.StatusInternalServerError,
	}
}

// MapError converts errors into PostgREST-shaped API errors.
func MapError(err error, ctx ErrorContext) *APIError {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr
	}

	switch {
	case errors.Is(err, errs.ErrNotFound):
		if ctx.RPC != "" {
			return RPCNotFound(ctx.Schema, ctx.RPC)
		}
		return TableNotFound(ctx.Schema, ctx.Table)
	case isJSONSyntaxError(err):
		return InvalidBody(err)
	default:
		var opErr *errkind.OpNotSupported
		if errors.As(err, &opErr) {
			return DriverOperationNotSupported(opErr.Operation, opErr.Protocol)
		}
		if strings.Contains(err.Error(), "not supported") {
			return UnsupportedOperation(err.Error())
		}
		return &APIError{
			Code:    "PGRST000",
			Message: err.Error(),
			Details: nil,
			Hint:    nil,
			Status:  http.StatusBadRequest,
		}
	}
}

func isJSONSyntaxError(err error) bool {
	if err == nil {
		return false
	}
	var syn *json.SyntaxError
	var unmarshal *json.UnmarshalTypeError
	if errors.As(err, &syn) || errors.As(err, &unmarshal) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "json") || strings.Contains(msg, "unexpected end of json")
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
