package client

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"testing"
	"time"

	"koditon/internal/platform/httpratelimit"

	"github.com/joho/godotenv"
)

func TestLiveAdPayloadDecodes(t *testing.T) {
	_ = godotenv.Load(".env.local", ".env", "../.env.local", "../.env", "backend/.env.local", "backend/.env")
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
	adID, err := strconv.Atoi(rawID)
	if err != nil {
		t.Fatalf("parse SHORTCUT_LIVE_AD_ID: %v", err)
	}
	c := NewClient(slog.Default(), nil, nil, baseURL, os.Getenv("SHORTCUT_DOCS_BASE_URL"), adBaseURL, userAgent, os.Getenv("SHORTCUT_SITEMAP_BASE_URL"), httpratelimit.Config{})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	raw, err := c.GetAdByID(ctx, adID)
	if err != nil {
		t.Fatalf("fetch live ad: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected non-empty payload")
	}
}
