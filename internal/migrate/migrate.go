package migrate

import (
	"context"
	"fmt"

	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/store/db"
	"gorm.io/gorm"
)

// Up applies model-driven schema migrations (GORM AutoMigrate).
// Server startup runs the same logic via db.New; this helper is for tests and tooling.
func Up(ctx context.Context, dsn string) error {
	gdb, err := db.Open(dsn)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return db.AutoMigrate(ctx, gdb)
}

// Down drops all Meta DB tables (destructive). For tests only.
func Down(ctx context.Context, dsn string) error {
	gdb, err := db.Open(dsn)
	if err != nil {
		return err
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	m := gdb.WithContext(ctx).Migrator()
	for _, model := range []any{
		&models.MetaForeignFunction{},
		&models.MetaForeignColumn{},
		&models.MetaForeignTable{},
		&models.MetaServer{},
		&models.MetaCredential{},
	} {
		if err := dropIfExists(m, model); err != nil {
			return fmt.Errorf("drop %T: %w", model, err)
		}
	}
	return nil
}

func dropIfExists(m gorm.Migrator, model any) error {
	if !m.HasTable(model) {
		return nil
	}
	return m.DropTable(model)
}
