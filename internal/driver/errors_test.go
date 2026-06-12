package driver_test

import (
	"errors"
	"testing"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/errkind"
	"github.com/monoposer/dataspan/internal/models"
)

func TestWrapNotSupported(t *testing.T) {
	err := driver.WrapNotSupported(driver.OpUpsert, models.ProtocolFile, driver.ErrNotSupported)
	var opErr *errkind.OpNotSupported
	if !errors.As(err, &opErr) {
		t.Fatalf("expected OpNotSupported, got %T", err)
	}
	if opErr.Operation != "upsert" || opErr.Protocol != "file" {
		t.Fatalf("opErr=%+v", opErr)
	}
}

func TestWrapNotSupportedPassthrough(t *testing.T) {
	want := errors.New("connection refused")
	if got := driver.WrapNotSupported(driver.OpInsert, models.ProtocolHTTP, want); !errors.Is(got, want) {
		t.Fatalf("got=%v", got)
	}
}
