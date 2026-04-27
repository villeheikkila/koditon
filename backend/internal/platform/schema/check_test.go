package schema

import (
	"context"
	"errors"
	"testing"
)

type fakeDB struct {
	version int32
	err     error
}

func (db fakeDB) GetSchemaVersion(context.Context) (int32, error) {
	return db.version, db.err
}

func TestCheckAcceptsRequiredVersion(t *testing.T) {
	t.Parallel()
	if err := Check(context.Background(), fakeDB{version: RequiredVersion}); err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
}

func TestCheckRejectsOldVersion(t *testing.T) {
	t.Parallel()
	if err := Check(context.Background(), fakeDB{version: RequiredVersion - 1}); err == nil {
		t.Fatal("expected old schema version to fail")
	}
}

func TestCheckWrapsReadError(t *testing.T) {
	t.Parallel()
	readErr := errors.New("boom")
	if err := Check(context.Background(), fakeDB{err: readErr}); !errors.Is(err, readErr) {
		t.Fatalf("Check error = %v, want wrapped read error", err)
	}
}
