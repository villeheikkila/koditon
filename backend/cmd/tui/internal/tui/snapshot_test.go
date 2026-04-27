package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"koditon-go/internal/domain/ads"
)

func TestScreenSnapshots(t *testing.T) {
	ctx := &appContext{styles: defaultStyles(), runtime: newJobRuntime(), subsystems: buildSubsystems()}
	home := newHomeScreen(ctx)
	home.Resize(100, 34)
	actions := newActionsScreen(ctx, 1)
	actions.Resize(100, 34)
	prompt := newPromptScreen(ctx, ctx.subsystems[1].Actions[3], nil, 0, "Subsystems > Frontdoor > Actions")
	prompt.Resize(100, 34)
	city := newCityPickerScreen(ctx, ctx.subsystems[3].Actions[4], nil, "Subsystems > Prices > Actions")
	city.Resize(100, 34)
	searchForm := newTransactionsSearchFormScreen(ctx, ctx.subsystems[3].Actions[5], []string{"Helsinki"}, "Subsystems > Prices > Actions")
	searchForm.Resize(100, 34)
	adsForm := newAdsReportFormScreen(ctx, ctx.subsystems[0].Actions[0], nil, "Subsystems > Ads > Actions")
	adsForm.Resize(100, 34)
	adsBrowser := newAdsReportBrowserScreen(ctx, ctx.subsystems[0].Actions[0], ads.SearchParams{Source: "all", Kind: "all", Sort: "seen_desc", Page: 1, PageSize: 50}, normalizeAdsFormValues(nil), "Subsystems > Ads > Actions").(*adsReportBrowserScreen)
	adsBrowser.Resize(100, 34)
	adsBrowser.loading = false
	adsBrowser.rows = []ads.UnifiedEntityRow{{CanonicalID: "shortcut:ad:12345", Source: "shortcut", Kind: "ad", NativeID: "12345", Headline: "Mannerheimintie 1", City: "Helsinki", Postal: "00100", Price: new(int64(230000)), Area: new(48.5), LastSeenAt: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)}, {CanonicalID: "frontdoor:announcement:9c58cc2a-73eb-44a9-8ede-f4c845a8ec45", Source: "frontdoor", Kind: "announcement", NativeID: "9c58cc2a-73eb-44a9-8ede-f4c845a8ec45", Headline: "Kalevankatu 2", City: "Helsinki", Postal: "00100", Price: new(int64(310000)), Area: new(71.0), LastSeenAt: time.Date(2026, 2, 22, 11, 0, 0, 0, time.UTC)}, {CanonicalID: "frontdoor:building:3f0268f4-2c8f-4a5a-af96-07025833928f", Source: "frontdoor", Kind: "building", NativeID: "3f0268f4-2c8f-4a5a-af96-07025833928f", Headline: "As Oy Example", City: "Espoo", Postal: "02100", LastSeenAt: time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)}}
	adsBrowser.total = int64(len(adsBrowser.rows))
	adsBrowser.table.SetRows(buildAdsTableRows(adsBrowser.rows))
	adsDetail := newAdsEntityDetailScreen(ctx, adsBrowser.rows[0], "Subsystems > Ads > Actions > Detail").(*adsEntityDetailScreen)
	adsDetail.Resize(100, 34)
	adsDetail.loading = false
	adsDetail.detail = ads.UnifiedEntityDetail{
		Canonical:      ads.UnifiedCanonicalFields{CanonicalID: "shortcut:ad:12345", Source: "shortcut", Kind: "ad", NativeID: "12345", Headline: "Mannerheimintie 1", Address: "Mannerheimintie 1", City: "Helsinki", Postal: "00100", Price: new(int64(230000)), Area: new(48.5), RoomLayout: "2h+k", URL: "https://example.test/ad/12345", LastSeenAt: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)},
		SourceSpecific: []ads.DetailField{{Label: "Ad Type", Value: "listing"}, {Label: "Building ID", Value: "3f0268f4-2c8f-4a5a-af96-07025833928f"}},
		Related:        []ads.DetailField{{Label: "Building Listings", Value: "17"}, {Label: "Building Rentals", Value: "6"}},
		Raw:            ads.RawPayload{Pretty: "{\n  \"foo\": \"bar\"\n}", OriginalBytes: 18},
	}
	cases := []struct {
		name   string
		screen Screen
	}{
		{name: "home", screen: home},
		{name: "actions", screen: actions},
		{name: "prompt", screen: prompt},
		{name: "city_loading", screen: city},
		{name: "transactions_search_form", screen: searchForm},
		{name: "ads_report_form", screen: adsForm},
		{name: "ads_report_browser", screen: adsBrowser},
		{name: "ads_entity_detail", screen: adsDetail},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := tc.screen.ShellState()
			state.Body = tc.screen.View()
			got := renderShell(ctx.styles, state)
			assertGolden(t, tc.name+".golden", got)
		})
	}
}

func assertGolden(t *testing.T, file string, got string) {
	t.Helper()
	path := filepath.Join("testdata", file)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
	}
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(wantBytes) {
		t.Fatalf("snapshot mismatch for %s", file)
	}
}
