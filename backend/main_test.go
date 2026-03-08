package main

import (
	"testing"

	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/app/analytics"
	"github.com/CloudNativeDevelopmentTeamH/analytics/backend/config"
)

func TestBuildStore_InMemory(t *testing.T) {
	cfg := config.Config{
		Storage: config.StorageConfig{Backend: "in-memory"},
	}

	store, closeFn, err := buildStore(cfg)
	if err != nil {
		t.Fatalf("buildStore returned error: %v", err)
	}

	if closeFn != nil {
		t.Fatalf("expected nil close function for in-memory store")
	}

	if _, ok := store.(*analytics.Store); !ok {
		t.Fatalf("expected *analytics.Store, got %T", store)
	}
}

func TestBuildStore_PostgresWithoutDSNReturnsError(t *testing.T) {
	cfg := config.Config{
		Storage: config.StorageConfig{
			Backend: "postgres",
			Postgres: config.StoragePostgresConfig{
				DSN: "",
			},
		},
	}

	store, closeFn, err := buildStore(cfg)
	if err == nil {
		t.Fatalf("expected error for empty dsn")
	}
	if store != nil {
		t.Fatalf("expected nil store on error")
	}
	if closeFn != nil {
		t.Fatalf("expected nil close function on error")
	}
}

func TestBuildStore_UnsupportedBackend(t *testing.T) {
	cfg := config.Config{
		Storage: config.StorageConfig{Backend: "redis"},
	}

	store, closeFn, err := buildStore(cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported backend")
	}
	if store != nil {
		t.Fatalf("expected nil store on error")
	}
	if closeFn != nil {
		t.Fatalf("expected nil close function on error")
	}
}
