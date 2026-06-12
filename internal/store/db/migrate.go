package db

import (
	"context"

	"github.com/monoposer/dataspan/internal/models"
	"gorm.io/gorm"
)

// AutoMigrate applies GORM schema migrations for Meta DB entity models.
func AutoMigrate(ctx context.Context, gdb *gorm.DB) error {
	return gdb.WithContext(ctx).AutoMigrate(models.MetaModels()...)
}
