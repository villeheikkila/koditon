package consumers

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDecodeDimensionLayerListingPayloadCarriesExpectedDirtyAt(t *testing.T) {
	listingID := uuid.New()
	expectedDirtyAt := time.Date(2026, 5, 9, 12, 30, 0, 0, time.UTC)
	raw, err := json.Marshal(dimensionLayerListingPayload{SaleListingID: listingID.String(), Reason: "dirty_target", ExpectedDirtyAt: &expectedDirtyAt})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	payload, err := decodeDimensionLayerListingPayload(syncJobEnvelope{SyncJobPayload: raw})
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ExpectedDirtyAt == nil {
		t.Fatal("expected dirty lease timestamp to be preserved")
	}
	if !payload.ExpectedDirtyAt.Equal(expectedDirtyAt) {
		t.Fatalf("expected dirty_at %s, got %s", expectedDirtyAt, payload.ExpectedDirtyAt)
	}
}

func TestDecodeDirtyDimensionTargetPayloadCarriesExpectedDirtyAt(t *testing.T) {
	targetID := uuid.New()
	expectedDirtyAt := time.Date(2026, 5, 9, 12, 45, 0, 0, time.UTC)
	raw, err := json.Marshal(dirtyDimensionTargetPayload{TargetType: "building", TargetID: targetID.String(), ExpectedDirtyAt: &expectedDirtyAt})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	payload, err := decodeDirtyDimensionTargetPayload(syncJobEnvelope{SyncJobPayload: raw})
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ExpectedDirtyAt == nil {
		t.Fatal("expected dirty lease timestamp to be preserved")
	}
	if !payload.ExpectedDirtyAt.Equal(expectedDirtyAt) {
		t.Fatalf("expected dirty_at %s, got %s", expectedDirtyAt, payload.ExpectedDirtyAt)
	}
}

func TestClearPropertyDimensionTargetDirtyHonorsExpectedDirtyAt(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("LOCAL_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL or LOCAL_DATABASE_URL to run DB-backed dirty lease test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()
	targetID := uuid.New()
	staleDirtyAt := time.Now().UTC().Add(-time.Hour)
	freshDirtyAt := time.Now().UTC()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM public.property_dimension_dirty_targets WHERE target_type = 'listing' AND target_id = $1`, targetID)
	})
	if _, err := pool.Exec(ctx, `INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at) VALUES ('listing', $1, ARRAY['test'], $2)`, targetID, freshDirtyAt); err != nil {
		t.Fatalf("insert dirty target: %v", err)
	}
	var cleared int32
	if err := pool.QueryRow(ctx, `SELECT public.fnc__clear_property_dimension_target_dirty('listing', $1, $2)`, targetID, staleDirtyAt).Scan(&cleared); err != nil {
		t.Fatalf("clear stale lease: %v", err)
	}
	if cleared != 0 {
		t.Fatalf("stale lease cleared %d rows, want 0", cleared)
	}
	var resolvedAt *time.Time
	if err := pool.QueryRow(ctx, `SELECT resolved_at FROM public.property_dimension_dirty_targets WHERE target_type = 'listing' AND target_id = $1`, targetID).Scan(&resolvedAt); err != nil {
		t.Fatalf("read dirty target: %v", err)
	}
	if resolvedAt != nil {
		t.Fatalf("stale lease set resolved_at = %s", resolvedAt)
	}
	if err := pool.QueryRow(ctx, `SELECT public.fnc__clear_property_dimension_target_dirty('listing', $1, $2)`, targetID, freshDirtyAt).Scan(&cleared); err != nil {
		t.Fatalf("clear current lease: %v", err)
	}
	if cleared != 1 {
		t.Fatalf("current lease cleared %d rows, want 1", cleared)
	}
}
