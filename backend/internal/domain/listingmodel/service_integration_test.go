package listingmodel

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
)

func TestSourceListingAssignmentLifecycle(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("LOCAL_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL or LOCAL_DATABASE_URL to run DB-backed listing model tests")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	queries := db.New(tx)
	firstSeen := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	secondSeen := firstSeen.AddDate(-2, 0, 0)
	firstSourceID := insertShortcutListingFixture(t, ctx, tx, queries, 910000001, 50, 200000, firstSeen)
	firstResult := reconcileListingFixture(t, ctx, queries, firstSourceID)
	secondSourceID := insertShortcutListingFixture(t, ctx, tx, queries, 910000002, 50, 205000, secondSeen)
	secondResult := reconcileListingFixture(t, ctx, queries, secondSourceID)
	if firstResult.ListingID == secondResult.ListingID {
		t.Fatal("same unit observations from separate campaigns merged into one listing")
	}
	assertConfirmedAssignment(t, ctx, tx, firstSourceID, firstResult.ListingID)
	assertConfirmedAssignment(t, ctx, tx, secondSourceID, secondResult.ListingID)
	assertCandidateStatus(t, ctx, tx, firstSourceID, secondSourceID, "proposed", 1)
	updateShortcutListingArea(t, ctx, tx, 910000002, 51)
	upsertedSourceID, err := queries.UpsertShortcutAdSourceListing(ctx, int64Pointer(910000002))
	secondSourceID = requiredFixtureSourceID(t, upsertedSourceID, err)
	reconcileListingFixture(t, ctx, queries, secondSourceID)
	assertCandidateStatus(t, ctx, tx, firstSourceID, secondSourceID, "superseded", 1)
	updateShortcutListingArea(t, ctx, tx, 910000002, 50)
	upsertedSourceID, err = queries.UpsertShortcutAdSourceListing(ctx, int64Pointer(910000002))
	secondSourceID = requiredFixtureSourceID(t, upsertedSourceID, err)
	reconcileListingFixture(t, ctx, queries, secondSourceID)
	assertCandidateStatus(t, ctx, tx, firstSourceID, secondSourceID, "proposed", 1)
	if _, err := tx.Exec(ctx, `UPDATE public.source_listing_match_candidates SET match_status = 'accepted', decided_at = now() WHERE match_status = 'proposed' AND source_listing_id_a = LEAST($1::uuid, $2::uuid) AND source_listing_id_b = GREATEST($1::uuid, $2::uuid)`, firstSourceID, secondSourceID); err != nil {
		t.Fatalf("accept candidate: %v", err)
	}
	reconcileListingFixture(t, ctx, queries, secondSourceID)
	assertCandidateStatus(t, ctx, tx, firstSourceID, secondSourceID, "accepted", 1)
	if _, err := tx.Exec(ctx, `UPDATE public.source_listing_match_candidates SET match_status = 'rejected', decided_at = now() WHERE match_status = 'accepted' AND source_listing_id_a = LEAST($1::uuid, $2::uuid) AND source_listing_id_b = GREATEST($1::uuid, $2::uuid)`, firstSourceID, secondSourceID); err != nil {
		t.Fatalf("reject candidate: %v", err)
	}
	reconcileListingFixture(t, ctx, queries, secondSourceID)
	assertCandidateStatus(t, ctx, tx, firstSourceID, secondSourceID, "proposed", 0)
	if _, err := tx.Exec(ctx, `UPDATE public.target_sources SET target_id = $1, link_method = 'manual' WHERE target_type = 'listing' AND source_type = 'source_listing' AND source_id = $2 AND link_status = 'confirmed'`, firstResult.ListingID, secondSourceID); err != nil {
		t.Fatalf("merge source assignment: %v", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM public.listings WHERE listing_id = $1`, secondResult.ListingID); err != nil {
		t.Fatalf("remove obsolete standalone listing: %v", err)
	}
	replacementSourceID, err := queries.DeleteShortcutAdSourceListing(ctx, int64Pointer(910000001))
	if err != nil {
		t.Fatalf("delete primary source listing: %v", err)
	}
	if replacementSourceID == nil || *replacementSourceID != secondSourceID {
		t.Fatalf("replacement source = %v, want %s", replacementSourceID, secondSourceID)
	}
	var deletedAssignments int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM public.target_sources WHERE target_type = 'listing' AND source_type = 'source_listing' AND source_id = $1`, firstSourceID).Scan(&deletedAssignments); err != nil {
		t.Fatalf("count deleted source assignments: %v", err)
	}
	if deletedAssignments != 0 {
		t.Fatalf("deleted source assignments = %d, want 0", deletedAssignments)
	}
	reconcileListingFixture(t, ctx, queries, *replacementSourceID)
	var primarySourceID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT primary_source_listing_id FROM public.listings WHERE listing_id = $1`, firstResult.ListingID).Scan(&primarySourceID); err != nil {
		t.Fatalf("read reprojected listing: %v", err)
	}
	if primarySourceID != secondSourceID {
		t.Fatalf("primary source = %s, want %s", primarySourceID, secondSourceID)
	}
}

func insertShortcutListingFixture(t *testing.T, ctx context.Context, tx pgx.Tx, queries *db.Queries, adID int64, area int, price int64, seenAt time.Time) uuid.UUID {
	t.Helper()
	payload := shortcutListingPayload(area, price)
	if _, err := tx.Exec(ctx, `INSERT INTO origin.shortcut_ads (shortcut_ad_id, shortcut_ad_url, shortcut_ad_type, shortcut_ad_first_seen_at, shortcut_ad_last_seen_at, shortcut_ad_data, shortcut_ad_data_hash) VALUES ($1, $2, 'listing', $3, $3, $4, $5)`, adID, "https://example.invalid/listing/"+time.Unix(adID, 0).UTC().Format("20060102150405"), seenAt, payload, uuid.NewString()); err != nil {
		t.Fatalf("insert shortcut ad %d: %v", adID, err)
	}
	sourceListingID, err := queries.UpsertShortcutAdSourceListing(ctx, &adID)
	return requiredFixtureSourceID(t, sourceListingID, err)
}

func updateShortcutListingArea(t *testing.T, ctx context.Context, tx pgx.Tx, adID int64, area int) {
	t.Helper()
	if _, err := tx.Exec(ctx, `UPDATE origin.shortcut_ads SET shortcut_ad_data = $2, shortcut_ad_data_hash = $3, shortcut_ad_data_changed_at = now() WHERE shortcut_ad_id = $1`, adID, shortcutListingPayload(area, 205000), uuid.NewString()); err != nil {
		t.Fatalf("update shortcut ad area: %v", err)
	}
}

func shortcutListingPayload(area int, price int64) json.RawMessage {
	payload, _ := json.Marshal(map[string]any{
		"address":   map[string]any{"street": map[string]any{"name": "Testikatu"}, "streetNumber": "7", "buildingLetter": "A", "city": map[string]any{"name": "Helsinki"}, "zipCode": map[string]any{"value": "00100"}},
		"adData":    map[string]any{"size": area, "rooms": 2, "roomConfiguration": "2h+k"},
		"priceData": map[string]any{"priceSell": price},
	})
	return payload
}

func reconcileListingFixture(t *testing.T, ctx context.Context, queries *db.Queries, sourceListingID uuid.UUID) db.ReconcileSourceListingModelRow {
	t.Helper()
	result, err := queries.ReconcileSourceListingModel(ctx, sourceListingID)
	if err != nil {
		t.Fatalf("reconcile source listing %s: %v", sourceListingID, err)
	}
	return result
}

func requiredFixtureSourceID(t *testing.T, sourceListingID *uuid.UUID, err error) uuid.UUID {
	t.Helper()
	if err != nil {
		t.Fatalf("upsert source listing: %v", err)
	}
	if sourceListingID == nil {
		t.Fatal("upsert source listing returned no id")
	}
	return *sourceListingID
}

func assertConfirmedAssignment(t *testing.T, ctx context.Context, tx pgx.Tx, sourceListingID, listingID uuid.UUID) {
	t.Helper()
	var assignedListingID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT target_id FROM public.target_sources WHERE target_type = 'listing' AND source_type = 'source_listing' AND source_id = $1 AND link_status = 'confirmed'`, sourceListingID).Scan(&assignedListingID); err != nil {
		t.Fatalf("read source assignment: %v", err)
	}
	if assignedListingID != listingID {
		t.Fatalf("source %s assigned to %s, want %s", sourceListingID, assignedListingID, listingID)
	}
}

func assertCandidateStatus(t *testing.T, ctx context.Context, tx pgx.Tx, sourceListingID1, sourceListingID2 uuid.UUID, status string, expected int) {
	t.Helper()
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM public.source_listing_match_candidates WHERE source_listing_id_a = LEAST($1::uuid, $2::uuid) AND source_listing_id_b = GREATEST($1::uuid, $2::uuid) AND match_status = $3`, sourceListingID1, sourceListingID2, status).Scan(&count); err != nil {
		t.Fatalf("count %s candidates: %v", status, err)
	}
	if count != expected {
		var facts string
		var candidates string
		var compatiblePairs int
		_ = tx.QueryRow(ctx, `SELECT COALESCE(string_agg(concat_ws('|', source_listing_id::text, postal_norm, street_norm, house_norm, stair_norm, apartment_norm, area_tenths::text), ', '), '') FROM public.source_listing_match_facts WHERE source_listing_id IN ($1, $2)`, sourceListingID1, sourceListingID2).Scan(&facts)
		_ = tx.QueryRow(ctx, `SELECT COALESCE(string_agg(concat_ws('|', match_method, match_status, match_score::text), ', '), '') FROM public.source_listing_match_candidates WHERE source_listing_id_a = LEAST($1::uuid, $2::uuid) AND source_listing_id_b = GREATEST($1::uuid, $2::uuid)`, sourceListingID1, sourceListingID2).Scan(&candidates)
		_ = tx.QueryRow(ctx, `SELECT count(*) FROM public.source_listing_match_facts a JOIN public.source_listing_match_facts b ON b.source_listing_id > a.source_listing_id AND b.postal_norm = a.postal_norm AND b.street_norm = a.street_norm AND b.house_norm = a.house_norm AND b.area_tenths = a.area_tenths WHERE a.source_listing_id IN ($1, $2) AND b.source_listing_id IN ($1, $2) AND (a.stair_norm IS NULL OR b.stair_norm IS NULL OR a.stair_norm = b.stair_norm) AND (a.apartment_norm IS NULL OR b.apartment_norm IS NULL OR a.apartment_norm = b.apartment_norm)`, sourceListingID1, sourceListingID2).Scan(&compatiblePairs)
		t.Fatalf("%s candidate count = %d, want %d; compatible pairs: %d; facts: %s; candidates: %s", status, count, expected, compatiblePairs, facts, candidates)
	}
}

func int64Pointer(value int64) *int64 {
	return &value
}
