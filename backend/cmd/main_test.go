package main

import (
	"context"
	"log/slog"
	"reflect"
	"testing"
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
