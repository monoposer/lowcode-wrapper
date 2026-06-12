package errkind

import "fmt"

// OpNotSupported indicates a protocol driver cannot perform an operation.
type OpNotSupported struct {
	Operation string
	Protocol  string
}

func (e *OpNotSupported) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s is not supported for protocol %q", e.Operation, e.Protocol)
}

func NewOpNotSupported(op, protocol string) error {
	return &OpNotSupported{Operation: op, Protocol: protocol}
}
