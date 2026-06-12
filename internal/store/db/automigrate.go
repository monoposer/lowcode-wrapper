package db

import (
	"os"
	"strings"
)

// AutoMigrateEnabledFromEnv reports whether Meta DB schema migration runs on server startup.
// Default: enabled (unset / 1 / true). Set DATASPAN_AUTOMIGRATE=0 for multi-replica production.
func AutoMigrateEnabledFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DATASPAN_AUTOMIGRATE")))
	switch v {
	case "0", "false", "no", "off":
		return false
	default:
		return true
	}
}
