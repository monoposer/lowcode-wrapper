package migrate_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/monoposer/dataspan/internal/migrate"
	"github.com/monoposer/dataspan/internal/store/db"
	"gorm.io/gorm"
)

func TestUpDownSQLite(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "meta.db")
	ctx := context.Background()

	if err := migrate.Up(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	gdb, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	m := gdb.Migrator()
	for _, table := range []string{"credentials", "servers", "foreign_tables", "foreign_columns", "foreign_functions"} {
		if !m.HasTable(table) {
			t.Fatalf("missing table %q", table)
		}
	}
	sqlDB, _ := gdb.DB()
	sqlDB.Close()

	if err := migrate.Down(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	gdb, err = db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		sqlDB, _ := gdb.DB()
		sqlDB.Close()
	}()
	if gdb.Migrator().HasTable("servers") {
		t.Fatal("servers table should be dropped")
	}
}

func TestUpIdempotent(t *testing.T) {
	dsn := filepath.Join(t.TempDir(), "meta.db")
	ctx := context.Background()
	if err := migrate.Up(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	if err := migrate.Up(ctx, dsn); err != nil {
		t.Fatal(err)
	}
	gdb, err := db.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := gdb.DB()
	defer sqlDB.Close()
	var n int64
	if err := gdb.Model(&struct{}{}).Table("servers").Count(&n).Error; err != nil && err != gorm.ErrRecordNotFound {
		t.Fatal(err)
	}
}
