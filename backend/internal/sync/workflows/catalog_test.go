package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/earendil-works/absurd/sdks/go/absurd"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestRegisterAllRegistersCatalogWithAbsurd(t *testing.T) {
	t.Parallel()
	app, err := absurd.New(absurd.Options{DriverName: "pgx", DatabaseURL: "postgres://unused/unused", QueueName: QueueCanonicalDB})
	if err != nil {
		t.Fatalf("absurd.New returned error: %v", err)
	}
	t.Cleanup(func() { _ = app.Close() })
	err = RegisterAll(app, func(context.Context, Params) (Result, error) {
		return Result{Status: "ok"}, nil
	})
	if err != nil {
		t.Fatalf("RegisterAll returned error: %v", err)
	}
}

func TestAllDefinitionsCoverTargetWorkflowShape(t *testing.T) {
	t.Parallel()
	defs := AllDefinitions()
	if len(defs) != 33 {
		t.Fatalf("definition count = %d, want 33", len(defs))
	}
	wantQueues := map[string]bool{
		QueueFrontdoor:       false,
		QueueShortcutAPI:     false,
		QueueShortcutScraper: false,
		QueuePrices:          false,
		QueuePostal:          false,
		QueueCanonicalDB:     false,
		QueueCanonicalLLM:    false,
	}
	seen := make(map[string]bool, len(defs))
	for _, def := range defs {
		if def.Provider == "" || def.Kind == "" || def.Queue == "" {
			t.Fatalf("definition has empty contract fields: %#v", def)
		}
		if seen[def.Provider+"/"+def.Kind] {
			t.Fatalf("duplicate definition: %s/%s", def.Provider, def.Kind)
		}
		seen[def.Provider+"/"+def.Kind] = true
		if _, ok := wantQueues[def.Queue]; !ok {
			t.Fatalf("%s uses unexpected queue %q", def.Kind, def.Queue)
		}
		wantQueues[def.Queue] = true
	}
	for queue, used := range wantQueues {
		if !used {
			t.Fatalf("queue %q is unused", queue)
		}
	}
}

func TestHandlerOnlyCanonicalJobsAreClassified(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind      string
		adminOnly bool
	}{
		{kind: "canonical_backfill_target_sources", adminOnly: true},
		{kind: "canonical_backfill_building_coordinates", adminOnly: true},
		{kind: "canonical_backfill_detached_houses", adminOnly: false},
	} {
		def, ok := FindDefinition("canonical", tc.kind)
		if !ok {
			t.Fatalf("%s missing", tc.kind)
		}
		if def.Queue != QueueCanonicalDB {
			t.Fatalf("%s queue = %q, want %q", tc.kind, def.Queue, QueueCanonicalDB)
		}
		if def.AdminOnly != tc.adminOnly {
			t.Fatalf("%s adminOnly = %v, want %v", tc.kind, def.AdminOnly, tc.adminOnly)
		}
	}
	if _, ok := FindDefinition("canonical", "canonical_rebuild_spatial_read_model"); ok {
		t.Fatal("spatial read model no-op should not be an Absurd task")
	}
}

func TestIdempotencyKeyMatchesOldDedupShape(t *testing.T) {
	t.Parallel()
	got := IdempotencyKey(" frontdoor ", " frontdoor_sync ", " ad:123 ")
	if got != "frontdoor:frontdoor_sync:ad:123" {
		t.Fatalf("idempotency key = %q", got)
	}
}

func TestValidateParams(t *testing.T) {
	t.Parallel()
	err := ValidateParams(Params{Provider: "frontdoor", Kind: "frontdoor_sync", EntityID: "ad:123", Payload: json.RawMessage(`{"source":"test"}`)})
	if err != nil {
		t.Fatalf("ValidateParams valid = %v", err)
	}
	err = ValidateParams(Params{Provider: "frontdoor", Kind: "missing", EntityID: "ad:123"})
	if !errors.Is(err, ErrUnknownTask) {
		t.Fatalf("ValidateParams unknown = %v, want ErrUnknownTask", err)
	}
	err = ValidateParams(Params{Provider: "frontdoor", Kind: "frontdoor_sync", EntityID: "ad:123", Payload: json.RawMessage(`{`)})
	if err == nil {
		t.Fatal("ValidateParams invalid payload succeeded")
	}
}

func TestQueueAssignmentsUseAbsurdIsolationQueues(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"shortcut_api_sync":                     QueueShortcutAPI,
		"shortcut_scraper_sync":                 QueueShortcutScraper,
		"canonical_extract_manager_certificate": QueueCanonicalLLM,
		"canonical_project_manager_certificate": QueueCanonicalDB,
		"prices_match_sale_listing":             QueuePrices,
		"frontdoor_ad_data_hash_backfill":       QueueFrontdoor,
		"postal_sync":                           QueuePostal,
	}
	for kind, want := range cases {
		provider := providerForKind(kind)
		def, ok := FindDefinition(provider, kind)
		if !ok {
			t.Fatalf("%s missing", kind)
		}
		if def.Queue != want {
			t.Fatalf("%s queue = %q, want %q", kind, def.Queue, want)
		}
	}
}

func TestImplementedCatalogMarksExecutableAbsurdTasks(t *testing.T) {
	t.Parallel()
	if !IsImplemented("postal", "postal_sync") {
		t.Fatal("postal_sync should be marked implemented")
	}
	for _, kind := range []string{
		"prices_cities_init",
		"prices_sync_all",
		"prices_sync",
		"prices_postal_code_sync",
		"prices_postal_code_page_sync",
		"prices_neighborhood_postal_code_sync",
		"prices_match_sale_listings_backfill",
		"prices_match_sale_listings_fanout",
		"prices_match_sale_listing",
	} {
		if !IsImplemented("prices", kind) {
			t.Fatalf("%s should be marked implemented", kind)
		}
	}
	for _, kind := range []string{
		"frontdoor_sitemap_sync",
		"frontdoor_buildings_sitemap_sync",
		"frontdoor_sync",
		"frontdoor_ad_data_hash_backfill",
	} {
		if !IsImplemented("frontdoor", kind) {
			t.Fatalf("%s should be marked implemented", kind)
		}
	}
	for _, kind := range []string{
		"shortcut_sitemap_sync",
		"shortcut_buildings_sitemap_sync",
		"shortcut_scraper_sync",
		"shortcut_api_sync",
		"shortcut_ad_data_hash_backfill",
	} {
		if !IsImplemented("shortcut", kind) {
			t.Fatalf("%s should be marked implemented", kind)
		}
	}
	for _, kind := range []string{
		"canonicalize_source_ads_fanout",
		"canonicalize_source_ad",
		"canonical_match_sale_listing_sources_backfill",
		"canonical_match_sale_listing_sources_fanout",
		"canonical_match_sale_listing_source",
		"canonical_rebuild_dimension_layer_backfill",
		"canonical_rebuild_dimension_layer_listing",
		"canonical_resolve_dirty_dimension_targets",
		"canonical_resolve_dimension_target",
		"canonical_extract_manager_certificate",
		"canonical_project_manager_certificate",
		"canonical_backfill_target_sources",
		"canonical_backfill_building_coordinates",
		"canonical_backfill_detached_houses",
	} {
		if !IsImplemented("canonical", kind) {
			t.Fatalf("%s should be marked implemented", kind)
		}
	}
}

func providerForKind(kind string) string {
	switch {
	case strings.HasPrefix(kind, "frontdoor"):
		return "frontdoor"
	case strings.HasPrefix(kind, "shortcut"):
		return "shortcut"
	case strings.HasPrefix(kind, "prices"):
		return "prices"
	case strings.HasPrefix(kind, "postal"):
		return "postal"
	default:
		return "canonical"
	}
}
