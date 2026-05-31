package driver

import (
	"context"
	"fmt"
	"sync"

	"lowcode-wrapper/internal/models"
)

var (
	mu        sync.RWMutex
	factories = map[models.Protocol]Factory{}
)

func Register(protocol models.Protocol, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[protocol] = factory
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (Driver, error) {
	mu.RLock()
	factory, ok := factories[srv.Protocol]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unsupported protocol %q", srv.Protocol)
	}
	return factory(ctx, srv, cred)
}
