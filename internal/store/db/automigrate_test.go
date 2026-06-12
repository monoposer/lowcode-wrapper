package db_test

import (
	"testing"

	"github.com/monoposer/dataspan/internal/store/db"
)

func TestAutoMigrateEnabledFromEnv(t *testing.T) {
	t.Setenv("DATASPAN_AUTOMIGRATE", "")
	if !db.AutoMigrateEnabledFromEnv() {
		t.Fatal("expected default enabled")
	}
	t.Setenv("DATASPAN_AUTOMIGRATE", "0")
	if db.AutoMigrateEnabledFromEnv() {
		t.Fatal("expected disabled")
	}
	t.Setenv("DATASPAN_AUTOMIGRATE", "false")
	if db.AutoMigrateEnabledFromEnv() {
		t.Fatal("expected disabled")
	}
	t.Setenv("DATASPAN_AUTOMIGRATE", "1")
	if !db.AutoMigrateEnabledFromEnv() {
		t.Fatal("expected enabled")
	}
}
