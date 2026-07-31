package sourceprice

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"koditon/internal/db"
)

func TestObserveMaintainsPricePeriodsAndCanonicalCurrentPrice(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		databaseURL = os.Getenv("LOCAL_DATABASE_URL")
	}
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL or LOCAL_DATABASE_URL to run DB-backed source price tests")
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
	t.Cleanup(func() {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			t.Errorf("rollback transaction: %v", rollbackErr)
		}
	})
	queries := db.New(tx)
	adID := int64(920000001)
	payload, err := json.Marshal(map[string]any{"address": map[string]any{"street": map[string]any{"name": "Hintakatu"}, "streetNumber": "1", "city": map[string]any{"name": "Helsinki"}, "zipCode": map[string]any{"value": "00100"}}, "adData": map[string]any{"size": 50}, "priceData": map[string]any{"priceSell": 999000}})
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO origin.shortcut_ads (shortcut_ad_id, shortcut_ad_url, shortcut_ad_type, shortcut_ad_data, shortcut_ad_data_hash) VALUES ($1, $2, 'listing', $3, $4)`, adID, "https://example.invalid/price-history", payload, uuid.NewString()); err != nil {
		t.Fatalf("insert shortcut ad: %v", err)
	}
	sourceListingID, err := queries.UpsertShortcutAdSourceListing(ctx, &adID)
	if err != nil || sourceListingID == nil {
		t.Fatalf("upsert source listing: id=%v err=%v", sourceListingID, err)
	}
	firstHash := uuid.NewString()
	changed, err := Observe(ctx, queries, *sourceListingID, Observation{AskingPrice: intPtr(200000), SourcePayloadHash: &firstHash})
	if err != nil || !changed {
		t.Fatalf("first observation: changed=%v err=%v", changed, err)
	}
	secondHash := uuid.NewString()
	changed, err = Observe(ctx, queries, *sourceListingID, Observation{AskingPrice: intPtr(200000), SourcePayloadHash: &secondHash})
	if err != nil || changed {
		t.Fatalf("same-price observation: changed=%v err=%v", changed, err)
	}
	changed, err = Observe(ctx, queries, *sourceListingID, Observation{AskingPrice: intPtr(190000)})
	if err != nil || !changed {
		t.Fatalf("lower-price observation: changed=%v err=%v", changed, err)
	}
	changed, err = Observe(ctx, queries, *sourceListingID, Observation{AskingPrice: intPtr(200000)})
	if err != nil || !changed {
		t.Fatalf("restored-price observation: changed=%v err=%v", changed, err)
	}
	history, err := queries.ListSourceListingPriceHistory(ctx, sourceListingID)
	if err != nil {
		t.Fatalf("list history: %v", err)
	}
	if len(history) != 3 {
		t.Fatalf("history length = %d, want 3", len(history))
	}
	if history[0].AskingPrice == nil || *history[0].AskingPrice != 200000 || history[0].SupersededAt != nil {
		t.Fatalf("unexpected current period: %#v", history[0])
	}
	if history[1].AskingPrice == nil || *history[1].AskingPrice != 190000 || history[1].SupersededAt == nil {
		t.Fatalf("unexpected previous period: %#v", history[1])
	}
	if !history[1].LastObservedAt.Before(*history[1].SupersededAt) {
		t.Fatalf("old price was recorded as observed at its replacement boundary: %#v", history[1])
	}
	if history[2].SourcePayloadHash == nil || *history[2].SourcePayloadHash != secondHash || history[2].SupersededAt == nil {
		t.Fatalf("same-price refresh was not retained: %#v", history[2])
	}
	reconciled, err := queries.ReconcileSourceListingModel(ctx, *sourceListingID)
	if err != nil {
		t.Fatalf("reconcile source listing: %v", err)
	}
	var canonicalAskingPrice *int64
	if err := tx.QueryRow(ctx, `SELECT property_offering_asking_price FROM public.property_offerings WHERE property_offering_id = $1`, reconciled.PropertyOfferingID).Scan(&canonicalAskingPrice); err != nil {
		t.Fatalf("read canonical asking price: %v", err)
	}
	if canonicalAskingPrice == nil || *canonicalAskingPrice != 200000 {
		t.Fatalf("canonical asking price = %v, want current source price 200000", canonicalAskingPrice)
	}
	changed, err = Observe(ctx, queries, *sourceListingID, Observation{})
	if err != nil || !changed {
		t.Fatalf("unavailable-price observation: changed=%v err=%v", changed, err)
	}
	history, err = queries.ListSourceListingPriceHistory(ctx, sourceListingID)
	if err != nil {
		t.Fatalf("list history after unavailable price: %v", err)
	}
	if len(history) != 4 || history[0].AskingPrice != nil || history[0].SupersededAt != nil {
		t.Fatalf("unexpected unavailable current period: %#v", history)
	}
}
