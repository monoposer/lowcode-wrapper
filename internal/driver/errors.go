package driver

import (
	"errors"

	"github.com/monoposer/dataspan/internal/errkind"
	"github.com/monoposer/dataspan/internal/models"
)

// ErrNotSupported indicates the driver does not implement the requested operation.
var ErrNotSupported = errors.New("operation not supported by this driver")

// Operation names supported operations for error reporting.
type Operation string

const (
	OpSelect Operation = "select"
	OpInsert Operation = "insert"
	OpUpdate Operation = "update"
	OpDelete Operation = "delete"
	OpUpsert Operation = "upsert"
	OpInvoke Operation = "invoke"
)

// OpNotSupported reports that a protocol does not implement an operation.
func OpNotSupported(op Operation, protocol models.Protocol) error {
	return errkind.NewOpNotSupported(string(op), string(protocol))
}

// WrapNotSupported converts ErrNotSupported into OpNotSupported.
func WrapNotSupported(op Operation, protocol models.Protocol, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrNotSupported) {
		return OpNotSupported(op, protocol)
	}
	var opErr *errkind.OpNotSupported
	if errors.As(err, &opErr) {
		return err
	}
	return err
}
