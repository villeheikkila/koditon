package shortcut

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/joho/godotenv"

	client "koditon/internal/clients/shortcut"
	"koditon/internal/platform/httpratelimit"
)

func TestLiveShortcutAdPayloadV1Validates(t *testing.T) {
	loadLiveEnv()
	if os.Getenv("SHORTCUT_LIVE_API_TESTS") != "1" {
		t.Skip("set SHORTCUT_LIVE_API_TESTS=1 to run live external API tests")
	}
	baseURL := os.Getenv("SHORTCUT_BASE_URL")
	adBaseURL := os.Getenv("SHORTCUT_AD_BASE_URL")
	userAgent := os.Getenv("SHORTCUT_USER_AGENT")
	rawID := os.Getenv("SHORTCUT_LIVE_AD_ID")
	if baseURL == "" || adBaseURL == "" || userAgent == "" || rawID == "" {
		t.Skip("set SHORTCUT_BASE_URL, SHORTCUT_AD_BASE_URL, SHORTCUT_USER_AGENT, and SHORTCUT_LIVE_AD_ID")
	}
	adID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		t.Fatalf("parse SHORTCUT_LIVE_AD_ID: %v", err)
	}
	c := client.NewClient(slog.Default(), nil, nil, baseURL, os.Getenv("SHORTCUT_DOCS_BASE_URL"), adBaseURL, userAgent, os.Getenv("SHORTCUT_SITEMAP_BASE_URL"), httpratelimit.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	raw, err := c.GetAdByID(ctx, int(adID))
	if err != nil {
		t.Fatalf("fetch live ad: %v", err)
	}
	if _, err := ValidateShortcutAdPayloadV1(raw, adID); err != nil {
		t.Fatalf("validate live ad: %v", err)
	}
}

func TestLiveShortcutAdPayloadV1ValidatesSitemapSample(t *testing.T) {
	loadLiveEnv()
	if os.Getenv("SHORTCUT_LIVE_API_TESTS") != "1" {
		t.Skip("set SHORTCUT_LIVE_API_TESTS=1 to run live external API tests")
	}
	c := liveShortcutClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	entries, err := c.GetSitemapEntries(ctx)
	if err != nil {
		t.Fatalf("fetch sitemap entries: %v", err)
	}
	sampleLimit := liveSampleLimit()
	var checked int
	var listingChecked bool
	var rentalChecked bool
	for _, entry := range entries {
		if entry.Type != client.SitemapURLTypeListing && entry.Type != client.SitemapURLTypeRental {
			continue
		}
		raw, err := c.GetAdByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("fetch live ad %d: %v", entry.ID, err)
		}
		payload, err := ValidateShortcutAdPayloadV1(raw, int64(entry.ID))
		if err != nil {
			t.Fatalf("validate live ad %d (%s): %v", entry.ID, entry.Type, err)
		}
		if payload.SchemaVersion != ShortcutAdPayloadSchemaVersion {
			t.Fatalf("live ad %d schema version = %d", entry.ID, payload.SchemaVersion)
		}
		listingChecked = listingChecked || payload.AdType == AdTypeListing
		rentalChecked = rentalChecked || payload.AdType == AdTypeRental
		checked++
		if checked >= sampleLimit && listingChecked && rentalChecked {
			break
		}
	}
	if checked < sampleLimit {
		t.Fatalf("validated %d live ads, expected at least %d", checked, sampleLimit)
	}
	if !listingChecked || !rentalChecked {
		t.Fatalf("expected both listing and rental samples, listing=%t rental=%t", listingChecked, rentalChecked)
	}
}

func liveShortcutClient(t *testing.T) *client.Client {
	t.Helper()
	baseURL := os.Getenv("SHORTCUT_BASE_URL")
	adBaseURL := os.Getenv("SHORTCUT_AD_BASE_URL")
	userAgent := os.Getenv("SHORTCUT_USER_AGENT")
	sitemapBaseURL := os.Getenv("SHORTCUT_SITEMAP_BASE_URL")
	if baseURL == "" || adBaseURL == "" || userAgent == "" || sitemapBaseURL == "" {
		t.Skip("set SHORTCUT_BASE_URL, SHORTCUT_AD_BASE_URL, SHORTCUT_USER_AGENT, and SHORTCUT_SITEMAP_BASE_URL")
	}
	return client.NewClient(slog.Default(), nil, nil, baseURL, os.Getenv("SHORTCUT_DOCS_BASE_URL"), adBaseURL, userAgent, sitemapBaseURL, httpratelimit.Config{})
}

func liveSampleLimit() int {
	raw := os.Getenv("SHORTCUT_LIVE_SAMPLE_LIMIT")
	if raw == "" {
		return 25
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 25
	}
	if value > 100 {
		return 100
	}
	return value
}

func loadLiveEnv() {
	_ = godotenv.Load(".env.local", ".env", "../.env.local", "../.env", "backend/.env.local", "backend/.env")
}
