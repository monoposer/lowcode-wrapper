package engine

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/models"
)

func TestMetaCacheReusesResolve(t *testing.T) {
	cache := &metaCache{
		ttl:    30 * time.Second,
		tables: make(map[string]metaCacheEntry[*models.ResolvedTable]),
		funcs:  make(map[string]metaCacheEntry[*models.ResolvedFunction]),
	}
	srv := models.Server{ID: uuid.New(), UpdatedAt: time.Now()}
	rt := &models.ResolvedTable{
		Table:  models.Table{SchemaName: "public", TableName: "orders"},
		Server: srv,
	}
	stub := &metaStoreStub{
		servers: []models.Server{srv},
		tables:  []models.Table{rt.Table},
	}

	if err := cache.syncRevision(context.Background(), stub); err != nil {
		t.Fatal(err)
	}
	cache.putTable("public", "orders", rt)
	firstLists := stub.listCalls
	if _, ok := cache.getTable("public", "orders"); !ok {
		t.Fatal("expected cached table")
	}
	if stub.listCalls != firstLists {
		t.Fatalf("getTable should not list store, calls=%d", stub.listCalls)
	}
}

func TestMetaCacheInvalidatesOnRevisionChange(t *testing.T) {
	cache := &metaCache{
		ttl:    30 * time.Second,
		tables: make(map[string]metaCacheEntry[*models.ResolvedTable]),
	}
	srv := models.Server{ID: uuid.New(), UpdatedAt: time.Now()}
	stub := &metaStoreStub{
		servers: []models.Server{srv},
		tables:  []models.Table{{ID: uuid.New(), SchemaName: "public", TableName: "orders"}},
	}
	if err := cache.syncRevision(context.Background(), stub); err != nil {
		t.Fatal(err)
	}
	cache.putTable("public", "orders", &models.ResolvedTable{Server: srv})

	srv.UpdatedAt = srv.UpdatedAt.Add(time.Second)
	stub.servers = []models.Server{srv}
	if err := cache.syncRevision(context.Background(), stub); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.getTable("public", "orders"); ok {
		t.Fatal("expected cache flush after revision change")
	}
}

func TestMetaCacheTTLFromEnv(t *testing.T) {
	t.Setenv("DATASPAN_META_CACHE_TTL", "")
	if got := metaCacheTTLFromEnv(); got != 0 {
		t.Fatalf("got=%v", got)
	}
	t.Setenv("DATASPAN_META_CACHE_TTL", "5s")
	if got := metaCacheTTLFromEnv(); got != 5*time.Second {
		t.Fatalf("got=%v", got)
	}
}

func TestNewMetaCacheFromEnvDisabled(t *testing.T) {
	t.Setenv("DATASPAN_META_CACHE_TTL", "")
	if c := newMetaCacheFromEnv(); c != nil {
		t.Fatal("expected nil cache")
	}
}
