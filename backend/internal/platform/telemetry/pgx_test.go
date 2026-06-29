package telemetry

import "testing"

func TestDefaultSpanNameUsesSQLCQueryName(t *testing.T) {
	t.Parallel()
	got := defaultSpanName("-- name: ListUsers :many\nSELECT * FROM users")
	if got != "ListUsers" {
		t.Fatalf("defaultSpanName = %q, want ListUsers", got)
	}
}

func TestDefaultSpanNameFallsBackToSQLVerb(t *testing.T) {
	t.Parallel()
	got := defaultSpanName("-- generated\nINSERT INTO users(id) VALUES($1)")
	if got != "insert" {
		t.Fatalf("defaultSpanName = %q, want insert", got)
	}
}
