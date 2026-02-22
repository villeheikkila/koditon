package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"koditon-go/internal/ads"
)

func TestScreenSnapshots(t *testing.T) {
	ctx := &appContext{styles: defaultStyles(), runtime: newJobRuntime(), subsystems: buildSubsystems()}
	home := newHomeScreen(ctx)
	home.Resize(100, 34)
	actions := newActionsScreen(ctx, 1)
	actions.Resize(100, 34)
	prompt := newPromptScreen(ctx, ctx.subsystems[1].Actions[1], nil, 0, "Subsystems > Frontdoor > Actions")
	prompt.Resize(100, 34)
	city := newCityPickerScreen(ctx, ctx.subsystems[3].Actions[3], nil, "Subsystems > Prices > Actions")
	city.Resize(100, 34)
	searchForm := newTransactionsSearchFormScreen(ctx, ctx.subsystems[3].Actions[4], []string{"Helsinki"}, "Subsystems > Prices > Actions")
	searchForm.Resize(100, 34)
	adsForm := newAdsReportFormScreen(ctx, ctx.subsystems[0].Actions[0], nil, "Subsystems > Ads > Actions")
	adsForm.Resize(100, 34)
	adsBrowser := newAdsReportBrowserScreen(ctx, ctx.subsystems[0].Actions[0], ads.SearchParams{Source: "all", Kind: "all", Sort: "seen_desc", Page: 1, PageSize: 50}, normalizeAdsFormValues(nil), "Subsystems > Ads > Actions").(*adsReportBrowserScreen)
	adsBrowser.Resize(100, 34)
	adsBrowser.loading = false
	adsBrowser.rows = []ads.ReportRow{
		{Source: "shortcut", Kind: "ad", EntityID: "12345", Headline: "Mannerheimintie 1", City: "Helsinki", Postal: "00100", Price: int64Ptr(230000), Area: float64Ptr(48.5), LastSeenAt: time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)},
		{Source: "frontdoor", Kind: "announcement", EntityID: "9c58cc2a-73eb-44a9-8ede-f4c845a8ec45", Headline: "Kalevankatu 2", City: "Helsinki", Postal: "00100", Price: int64Ptr(310000), Area: float64Ptr(71.0), LastSeenAt: time.Date(2026, 2, 22, 11, 0, 0, 0, time.UTC)},
	}
	adsBrowser.total = int64(len(adsBrowser.rows))
	adsBrowser.table.SetRows(buildAdsTableRows(adsBrowser.rows))
	adsBrowser.detail = ads.Detail{Summary: []ads.DetailField{{Label: "Source", Value: "shortcut/ad"}, {Label: "Ad ID", Value: "12345"}, {Label: "URL", Value: "https://example.test/ad/12345"}}, Related: []ads.DetailField{{Label: "Building ID", Value: "3f0268f4-2c8f-4a5a-af96-07025833928f"}}}
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

func int64Ptr(v int64) *int64 {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
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
