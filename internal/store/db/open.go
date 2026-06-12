package db

import (
	"fmt"
	"strings"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func Open(dsn string) (*gorm.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("database DSN is required")
	}
	dialector, err := dialectorFor(dsn)
	if err != nil {
		return nil, err
	}
	gdb, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, err
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return gdb, nil
}

func dialectorFor(dsn string) (gorm.Dialector, error) {
	lower := strings.ToLower(dsn)
	switch {
	case strings.HasPrefix(lower, "postgres://"), strings.HasPrefix(lower, "postgresql://"):
		return postgres.Open(dsn), nil
	case strings.HasPrefix(lower, "mysql://"):
		return mysql.Open(dsn), nil
	case strings.HasPrefix(lower, "file:"), strings.HasPrefix(lower, "sqlite:"),
		strings.HasSuffix(lower, ".db"), strings.HasSuffix(lower, ".sqlite"), strings.HasSuffix(lower, ".sqlite3"):
		return sqlite.Open(dsn), nil
	default:
		return nil, fmt.Errorf("unsupported database DSN %q (use postgres, mysql, or sqlite)", dsn)
	}
}
