package driver

import "context"

// Counter is implemented by drivers that can count rows efficiently.
type Counter interface {
	Count(ctx context.Context, req SelectRequest) (int, error)
}

// Count returns the number of rows matching req, using a native COUNT when available.
func Count(ctx context.Context, drv Driver, req SelectRequest) (int, error) {
	if c, ok := drv.(Counter); ok {
		return c.Count(ctx, req)
	}
	req2 := req
	req2.Limit = 0
	req2.Offset = 0
	req2.Select = nil
	rows, err := drv.Select(ctx, req2)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}
