package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Dir returns the migrations directory (scripts/migrations), overridable via MIGRATIONS_DIR.
func Dir() (string, error) {
	if d := os.Getenv("MIGRATIONS_DIR"); d != "" {
		return d, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, "scripts", "migrations")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("scripts/migrations not found (set MIGRATIONS_DIR or run from repo root)")
}

func Up(ctx context.Context, databaseURL string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.up.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no *.up.sql files in %s", dir)
	}
	sort.Strings(files)
	return apply(ctx, databaseURL, files)
}

func Down(ctx context.Context, databaseURL string) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.down.sql"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no *.down.sql files in %s", dir)
	}
	sort.Strings(files)
	slices.Reverse(files)
	return apply(ctx, databaseURL, files)
}

func apply(ctx context.Context, databaseURL string, files []string) error {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	for _, path := range files {
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		body := strings.TrimSpace(string(sql))
		if body == "" {
			continue
		}
		if _, err := pool.Exec(ctx, body); err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
	}
	return nil
}
