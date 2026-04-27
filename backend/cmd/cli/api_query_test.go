package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAPIQueryFrontdoorAdPrintsPrettyJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/announcement/details" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("friendlyId"); got != "abc123" {
			t.Fatalf("friendlyId = %s", got)
		}
		if got := r.Header.Get("User-Agent"); got != "test-agent" {
			t.Fatalf("user agent = %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":123,"friendlyId":"abc123","publishingTime":1}`))
	}))
	t.Cleanup(server.Close)
	var stdout bytes.Buffer
	err := runAPIQuery(context.Background(), []string{"frontdoor", "ad", "--friendly-id", "abc123"}, &stdout, envGetter(map[string]string{
		"FRONTDOOR_BASE_URL":   server.URL,
		"FRONTDOOR_USER_AGENT": "test-agent",
	}))
	if err != nil {
		t.Fatalf("runAPIQuery returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "\"friendlyId\": \"abc123\"") {
		t.Fatalf("output did not contain pretty JSON friendlyId: %s", out)
	}
}

func TestAPIQueryFrontdoorBuildingPageReportsParseError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body>missing state</body></html>`))
	}))
	t.Cleanup(server.Close)
	var stdout bytes.Buffer
	err := runAPIQuery(context.Background(), []string{"frontdoor", "building-page", "--url", server.URL}, &stdout, envGetter(map[string]string{
		"FRONTDOOR_USER_AGENT": "test-agent",
	}))
	if err == nil {
		t.Fatal("runAPIQuery returned nil error")
	}
	if !strings.Contains(err.Error(), "initial state not found") {
		t.Fatalf("error = %v", err)
	}
}

func TestAPIQueryShortcutAdFetchesTokenAndPrintsRawJSON(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/get":
			_, _ = w.Write([]byte(`{"user":{"cuid":"cuid-1","token":"token-1","time":12345}}`))
		case "/api/v5/5/apartments/items/42":
			if got := r.Header.Get("OTA-cuid"); got != "cuid-1" {
				t.Fatalf("OTA-cuid = %s", got)
			}
			_, _ = w.Write([]byte(`{"cardType":100,"address":"Testikatu 1"}`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	var stdout bytes.Buffer
	err := runAPIQuery(context.Background(), []string{"shortcut", "ad", "--id", "42"}, &stdout, envGetter(map[string]string{
		"SHORTCUT_BASE_URL":    server.URL,
		"SHORTCUT_AD_BASE_URL": server.URL,
		"SHORTCUT_USER_AGENT":  "test-agent",
	}))
	if err != nil {
		t.Fatalf("runAPIQuery returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "\"address\": \"Testikatu 1\"") {
		t.Fatalf("output did not contain pretty JSON address: %s", out)
	}
}

func TestAPIQueryShortcutLocationsBuildsExpectedQuery(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user/get":
			_, _ = w.Write([]byte(`{"user":{"cuid":"cuid-1","token":"token-1","time":12345}}`))
		case "/api/5.0/location":
			if got := r.URL.Query().Get("query"); got != "00100" {
				t.Fatalf("query = %s", got)
			}
			if got := r.URL.Query().Get("card_type"); got != "5" {
				t.Fatalf("card_type = %s", got)
			}
			_, _ = w.Write([]byte(`[{"card":{"name":"00100 Helsinki","cardId":1,"cardType":5},"parent":{"name":"Helsinki","cardId":2,"cardType":1}}]`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	var stdout bytes.Buffer
	err := runAPIQuery(context.Background(), []string{"shortcut", "locations", "--postal", "00100"}, &stdout, envGetter(map[string]string{
		"SHORTCUT_BASE_URL":   server.URL,
		"SHORTCUT_USER_AGENT": "test-agent",
	}))
	if err != nil {
		t.Fatalf("runAPIQuery returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "\"name\": \"00100 Helsinki\"") {
		t.Fatalf("output = %s", stdout.String())
	}
}

func TestAPIQueryShortcutSitemapPrintsParsedEntries(t *testing.T) {
	t.Parallel()
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemaps/index.xml":
			_, _ = w.Write([]byte(`<urlset><url><loc>` + serverURL + `/sitemaps/sm_ad_1.xml</loc></url></urlset>`))
		case "/sitemaps/sm_ad_1.xml":
			_, _ = w.Write([]byte(`<urlset><url><loc>` + serverURL + `/myytavat-asunnot/helsinki/42</loc></url></urlset>`))
		default:
			t.Fatalf("path = %s", r.URL.Path)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)
	var stdout bytes.Buffer
	err := runAPIQuery(context.Background(), []string{"shortcut", "sitemap"}, &stdout, envGetter(map[string]string{
		"SHORTCUT_SITEMAP_BASE_URL": server.URL,
		"SHORTCUT_USER_AGENT":       "test-agent",
	}))
	if err != nil {
		t.Fatalf("runAPIQuery returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "\"type\": \"listing\"") || !strings.Contains(out, "\"id\": \"42\"") {
		t.Fatalf("output = %s", out)
	}
}

func TestAPIQueryFrontdoorProfileSchemaUsesFixtureCache(t *testing.T) {
	t.Parallel()
	cacheDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(cacheDir, "sitemap"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "ad"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(cacheDir, "building-state"), 0o755); err != nil {
		t.Fatal(err)
	}
	sitemap := `[{"id":"abc123","type":"ad","url":"https://example.invalid/kohde/abc123"},{"id":"123","type":"building","url":"https://example.invalid/talo/123"}]`
	if err := os.WriteFile(filepath.Join(cacheDir, "sitemap", "entries.json"), []byte(sitemap), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "ad", "abc123.json"), []byte(`{"id":1,"friendlyId":"abc123","sellingPrice":250000,"property":{"postCode":{"postCode":"00100"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	buildingStatePath := filepath.Join(cacheDir, "building-state", sha256Hex("https://example.invalid/talo/123")+".json")
	if err := os.WriteFile(buildingStatePath, []byte(`{"housingCompany":{"id":123,"name":"As Oy Test"},"announcements":[{"searchPrice":100000}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	err := runAPIQuery(context.Background(), []string{"frontdoor", "profile-schema", "--cache-dir", cacheDir, "--sample-size", "2", "--compact"}, &stdout, envGetter(nil))
	if err != nil {
		t.Fatalf("runAPIQuery returned error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{`"processed":2`, `"$\.friendlyId"`, `"adSamples":1`, `"buildingSamples":1`} {
		if !strings.Contains(out, strings.ReplaceAll(want, `\`, ``)) {
			t.Fatalf("output missing %s: %s", want, out)
		}
	}
}

func TestShortcutSearchParamsValidation(t *testing.T) {
	t.Parallel()
	params, err := shortcutSearchParams(10, 5, "Helsinki", "rent", 1, 20)
	if err != nil {
		t.Fatalf("shortcutSearchParams returned error: %v", err)
	}
	if params.Location.Card.CardID != 10 || params.Location.Card.CardType != 5 || params.CardType != "101" {
		t.Fatalf("params = %#v", params)
	}
}

func envGetter(values map[string]string) func(string) string {
	return func(key string) string {
		return values[key]
	}
}
