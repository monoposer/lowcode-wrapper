package driver

// Closer releases protocol resources (pools, clients). Optional on Driver implementations.
type Closer interface {
	Close() error
}

// Close releases driver resources when the implementation supports it.
func Close(d Driver) {
	if d == nil {
		return
	}
	if c, ok := d.(Closer); ok {
		_ = c.Close()
	}
}
