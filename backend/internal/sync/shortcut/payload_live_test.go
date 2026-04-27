package shortcut

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	client "koditon/internal/clients/shortcut"
)

func TestLiveShortcutAdPayloadV1Validates(t *testing.T) {
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
	c := client.NewClient(slog.Default(), nil, nil, baseURL, os.Getenv("SHORTCUT_DOCS_BASE_URL"), adBaseURL, userAgent, os.Getenv("SHORTCUT_SITEMAP_BASE_URL"))
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
