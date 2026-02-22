package tui

import (
	"testing"
	"time"

	"koditon-go/internal/prices"
)

func TestParseSortMode(t *testing.T) {
	if got := parseSortMode("date_desc"); got != "date_desc" {
		t.Fatalf("expected date_desc, got %q", got)
	}
	if got := parseSortMode("unknown"); got != "price_asc" {
		t.Fatalf("expected default price_asc, got %q", got)
	}
}

func TestFilterRowsByArea(t *testing.T) {
	min := 35.0
	max := 45.0
	rows := []prices.SearchTransactionsRow{
		{Area: 30.0},
		{Area: 40.0},
		{Area: 50.0},
	}
	filtered := filterRowsByArea(rows, &min, &max)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 row, got %d", len(filtered))
	}
	if filtered[0].Area != 40.0 {
		t.Fatalf("expected area 40.0, got %.1f", filtered[0].Area)
	}
}

func TestSortRows(t *testing.T) {
	day1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	rows := []prices.SearchTransactionsRow{
		{Price: 500000, CreatedAt: day1},
		{Price: 300000, CreatedAt: day2},
		{Price: 400000, CreatedAt: day1},
	}
	sortRows(rows, "price_asc")
	if rows[0].Price != 300000 || rows[1].Price != 400000 || rows[2].Price != 500000 {
		t.Fatalf("unexpected price_asc order: %+v", rows)
	}
	sortRows(rows, "date_desc")
	if !rows[0].CreatedAt.Equal(day2) {
		t.Fatalf("expected newest first for date_desc")
	}
}
