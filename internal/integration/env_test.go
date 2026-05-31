package integration_test

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"lowcode-wrapper/internal/auth"
	"lowcode-wrapper/internal/migrate"
	store "lowcode-wrapper/internal/store/postgres"
)

func initIntegrationEnv(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set")
	}
	key := make([]byte, 32)
	for i := range key {
		key[i] = 0xAB
	}
	os.Setenv("WRAPPER_MASTER_KEY", base64.StdEncoding.EncodeToString(key))

	if err := migrate.Up(context.Background(), dsn); err != nil {
		t.Fatal(err)
	}
	vault, err := auth.NewVaultFromEnv()
	if err != nil {
		t.Fatal(err)
	}
	st, err := store.NewFromEnv(vault)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(st.Close)
	return st
}
