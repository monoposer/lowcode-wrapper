package postgres

import "testing"

func TestResolveDSNFromFields(t *testing.T) {
	dsn, err := Config{
		Host:     "localhost",
		Port:     5433,
		Username: "wrapper",
		Password: "secret",
		Database: "wrapper_meta",
		SSLMode:  "disable",
	}.ResolveDSN()
	if err != nil {
		t.Fatal(err)
	}
	if dsn != "postgresql://wrapper:secret@localhost:5433/wrapper_meta?sslmode=disable" {
		t.Fatalf("dsn = %q", dsn)
	}
}

func TestResolveDSNExplicit(t *testing.T) {
	want := "postgresql://u:p@db.example.com:5432/app?sslmode=require"
	dsn, err := Config{DSN: want}.ResolveDSN()
	if err != nil {
		t.Fatal(err)
	}
	if dsn != want {
		t.Fatalf("dsn = %q", dsn)
	}
}

func TestResolveDSNMissingFields(t *testing.T) {
	_, err := Config{Host: "localhost"}.ResolveDSN()
	if err == nil {
		t.Fatal("expected error")
	}
}
