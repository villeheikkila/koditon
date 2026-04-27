package app

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"koditon/internal/platform/config"
)

func TestLifecycleCleanupRunsInReverseOrder(t *testing.T) {
	t.Parallel()
	lifecycle := newLifecycle(slog.Default())
	var calls []string
	lifecycle.Defer("first", func(context.Context) error {
		calls = append(calls, "first")
		return nil
	})
	lifecycle.Defer("second", func(context.Context) error {
		calls = append(calls, "second")
		return nil
	})
	if err := lifecycle.Cleanup(context.Background()); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	want := []string{"second", "first"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
	if err := lifecycle.Cleanup(context.Background()); err != nil {
		t.Fatalf("second Cleanup returned error: %v", err)
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("second cleanup changed calls: %v", calls)
	}
}

func TestNewDatabasePoolAppliesConfiguredPoolSettings(t *testing.T) {
	t.Parallel()
	cfg := config.Config{
		DatabaseURL: "postgres://postgres:postgres@localhost:5432/koditon?sslmode=disable",
		Database: config.DatabaseConfig{
			MaxConns:          10,
			MinConns:          2,
			MaxConnLifetime:   30 * time.Minute,
			MaxConnIdleTime:   5 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
	}
	pool, err := newDatabasePool(context.Background(), cfg)
	if err != nil {
		t.Fatalf("newDatabasePool returned error: %v", err)
	}
	defer pool.Close()
	poolCfg := pool.Config()
	if poolCfg.MaxConns != 10 {
		t.Fatalf("MaxConns = %d, want 10", poolCfg.MaxConns)
	}
	if poolCfg.MinConns != 2 {
		t.Fatalf("MinConns = %d, want 2", poolCfg.MinConns)
	}
	if poolCfg.MaxConnLifetime != 30*time.Minute {
		t.Fatalf("MaxConnLifetime = %v, want 30m", poolCfg.MaxConnLifetime)
	}
}
