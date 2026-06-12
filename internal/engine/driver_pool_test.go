package engine

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
)

type stubDriver struct {
	id int
}

func (stubDriver) Select(context.Context, driver.SelectRequest) ([]map[string]any, error) {
	return nil, nil
}
func (stubDriver) Insert(context.Context, driver.RowRequest) (map[string]any, error) {
	return nil, nil
}
func (stubDriver) Update(context.Context, driver.RowRequest) (int, error) { return 0, nil }
func (stubDriver) Upsert(context.Context, driver.RowRequest) (bool, map[string]any, error) {
	return false, nil, nil
}
func (stubDriver) Delete(context.Context, driver.DeleteRequest) (int, error) { return 0, nil }
func (stubDriver) Invoke(context.Context, driver.InvokeRequest) (any, error) { return nil, nil }

func TestDriverPoolReusesDriver(t *testing.T) {
	t.Parallel()

	pool := newDriverPool()
	srv := models.Server{
		ID:        uuid.New(),
		Name:      "pg",
		Protocol:  models.ProtocolPostgres,
		Options:   json.RawMessage(`{"dsn":"postgres://localhost/test"}`),
		UpdatedAt: time.Now(),
	}
	var created int32
	factory := func(context.Context, models.Server, map[string]any) (driver.Driver, error) {
		n := atomic.AddInt32(&created, 1)
		return stubDriver{id: int(n)}, nil
	}

	d1, first, err := pool.Get(context.Background(), srv, nil, factory)
	if err != nil || !first {
		t.Fatalf("first get: drv=%v created=%v err=%v", d1, first, err)
	}
	d2, again, err := pool.Get(context.Background(), srv, nil, factory)
	if err != nil || again {
		t.Fatalf("second get: drv=%v created=%v err=%v", d2, again, err)
	}
	if d1.(stubDriver).id != d2.(stubDriver).id {
		t.Fatalf("expected same driver instance")
	}
	if atomic.LoadInt32(&created) != 1 {
		t.Fatalf("factory called %d times", created)
	}
}

func TestDriverPoolRecreatesOnConfigChange(t *testing.T) {
	t.Parallel()

	pool := newDriverPool()
	srv := models.Server{
		ID:        uuid.New(),
		Name:      "pg",
		Protocol:  models.ProtocolPostgres,
		Options:   json.RawMessage(`{"dsn":"postgres://localhost/test"}`),
		UpdatedAt: time.Now(),
	}
	var created int32
	factory := func(context.Context, models.Server, map[string]any) (driver.Driver, error) {
		n := atomic.AddInt32(&created, 1)
		return stubDriver{id: int(n)}, nil
	}

	d1, _, err := pool.Get(context.Background(), srv, nil, factory)
	if err != nil {
		t.Fatal(err)
	}

	srv.UpdatedAt = srv.UpdatedAt.Add(time.Second)
	d2, recreated, err := pool.Get(context.Background(), srv, nil, factory)
	if err != nil || !recreated {
		t.Fatalf("recreated=%v err=%v", recreated, err)
	}
	if d1.(stubDriver).id == d2.(stubDriver).id {
		t.Fatal("expected new driver after config change")
	}
	if atomic.LoadInt32(&created) != 2 {
		t.Fatalf("factory called %d times", created)
	}
}
