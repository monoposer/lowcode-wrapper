package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
)

type driverPool struct {
	mu    sync.Mutex
	items map[uuid.UUID]*pooledDriver
}

type pooledDriver struct {
	version string
	drv     driver.Driver
}

func newDriverPool() *driverPool {
	return &driverPool{items: make(map[uuid.UUID]*pooledDriver)}
}

func (p *driverPool) Get(
	ctx context.Context,
	srv models.Server,
	cred map[string]any,
	create func(context.Context, models.Server, map[string]any) (driver.Driver, error),
) (driver.Driver, bool, error) {
	version := driverConfigVersion(srv, cred)

	p.mu.Lock()
	if item, ok := p.items[srv.ID]; ok && item.version == version {
		drv := item.drv
		p.mu.Unlock()
		return drv, false, nil
	}
	p.mu.Unlock()

	drv, err := create(ctx, srv, cred)
	if err != nil {
		return nil, false, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if item, ok := p.items[srv.ID]; ok && item.version == version {
		driver.Close(drv)
		return item.drv, false, nil
	}
	if item, ok := p.items[srv.ID]; ok {
		driver.Close(item.drv)
	}
	p.items[srv.ID] = &pooledDriver{version: version, drv: drv}
	return drv, true, nil
}

func (p *driverPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for id, item := range p.items {
		driver.Close(item.drv)
		delete(p.items, id)
	}
}

func driverConfigVersion(srv models.Server, cred map[string]any) string {
	h := sha256.New()
	h.Write([]byte(srv.UpdatedAt.UTC().Format(time.RFC3339Nano)))
	h.Write([]byte(srv.Protocol))
	h.Write(srv.Options)
	if srv.CredentialRef != nil {
		h.Write(srv.CredentialRef[:])
	}
	hashCredential(h, cred)
	return hex.EncodeToString(h.Sum(nil)[:16])
}

func hashCredential(h interface {
	Write([]byte) (int, error)
}, cred map[string]any) {
	if len(cred) == 0 {
		return
	}
	keys := make([]string, 0, len(cred))
	for k := range cred {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte(fmt.Sprint(cred[k])))
	}
}
