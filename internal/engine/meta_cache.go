package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store"
)

type metaCache struct {
	mu    sync.RWMutex
	ttl   time.Duration
	rev   string
	tables map[string]metaCacheEntry[*models.ResolvedTable]
	funcs  map[string]metaCacheEntry[*models.ResolvedFunction]
}

type metaCacheEntry[T any] struct {
	value   T
	expires time.Time
}

func newMetaCacheFromEnv() *metaCache {
	ttl := metaCacheTTLFromEnv()
	if ttl <= 0 {
		return nil
	}
	return &metaCache{
		ttl:    ttl,
		tables: make(map[string]metaCacheEntry[*models.ResolvedTable]),
		funcs:  make(map[string]metaCacheEntry[*models.ResolvedFunction]),
	}
}

func metaCacheTTLFromEnv() time.Duration {
	raw := strings.TrimSpace(os.Getenv("DATASPAN_META_CACHE_TTL"))
	if raw == "" {
		return 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return 0
	}
	return d
}

func metaCacheKey(schema, name string) string {
	return schema + "\x00" + name
}

func (c *metaCache) syncRevision(ctx context.Context, s store.Store) error {
	rev, err := storeRevision(ctx, s)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rev != rev {
		c.rev = rev
		c.tables = make(map[string]metaCacheEntry[*models.ResolvedTable])
		c.funcs = make(map[string]metaCacheEntry[*models.ResolvedFunction])
	}
	return nil
}

func (c *metaCache) getTable(schema, table string) (*models.ResolvedTable, bool) {
	key := metaCacheKey(schema, table)
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.tables[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.value, true
}

func (c *metaCache) putTable(schema, table string, rt *models.ResolvedTable) {
	key := metaCacheKey(schema, table)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tables[key] = metaCacheEntry[*models.ResolvedTable]{
		value:   rt,
		expires: time.Now().Add(c.ttl),
	}
}

func (c *metaCache) getFunction(schema, name string) (*models.ResolvedFunction, bool) {
	key := metaCacheKey(schema, name)
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.funcs[key]
	if !ok || time.Now().After(entry.expires) {
		return nil, false
	}
	return entry.value, true
}

func (c *metaCache) putFunction(schema, name string, rf *models.ResolvedFunction) {
	key := metaCacheKey(schema, name)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.funcs[key] = metaCacheEntry[*models.ResolvedFunction]{
		value:   rf,
		expires: time.Now().Add(c.ttl),
	}
}

func storeRevision(ctx context.Context, s store.Store) (string, error) {
	servers, err := s.ListServers(ctx)
	if err != nil {
		return "", err
	}
	tables, err := s.ListTables(ctx)
	if err != nil {
		return "", err
	}
	functions, err := s.ListFunctions(ctx)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "s:%d|", len(servers))
	for _, srv := range servers {
		h.Write(srv.ID[:])
		_, _ = h.Write([]byte(srv.UpdatedAt.UTC().Format(time.RFC3339Nano)))
	}
	_, _ = fmt.Fprintf(h, "t:%d|", len(tables))
	for _, tbl := range tables {
		h.Write(tbl.ID[:])
	}
	_, _ = fmt.Fprintf(h, "f:%d|", len(functions))
	for _, fn := range functions {
		h.Write(fn.ID[:])
	}
	return hex.EncodeToString(h.Sum(nil)[:12]), nil
}

func (e *Engine) resolveTable(ctx context.Context, schema, table string) (*models.ResolvedTable, error) {
	if e.meta != nil {
		if err := e.meta.syncRevision(ctx, e.Store); err != nil {
			return nil, err
		}
		if rt, ok := e.meta.getTable(schema, table); ok {
			return rt, nil
		}
	}
	rt, err := e.Store.ResolveTable(ctx, schema, table)
	if err != nil {
		return nil, err
	}
	if e.meta != nil {
		e.meta.putTable(schema, table, rt)
	}
	return rt, nil
}

func (e *Engine) resolveFunction(ctx context.Context, schema, name string) (*models.ResolvedFunction, error) {
	if e.meta != nil {
		if err := e.meta.syncRevision(ctx, e.Store); err != nil {
			return nil, err
		}
		if rf, ok := e.meta.getFunction(schema, name); ok {
			return rf, nil
		}
	}
	rf, err := e.Store.ResolveFunction(ctx, schema, name)
	if err != nil {
		return nil, err
	}
	if e.meta != nil {
		e.meta.putFunction(schema, name, rf)
	}
	return rf, nil
}
