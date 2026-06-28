package workflows

import (
	"encoding/json"
	"errors"
	"testing"
)

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
		if def.Name == "" || def.Queue == "" {
			t.Fatalf("definition has empty contract fields: %#v", def)
		}
		if seen[def.Name] {
			t.Fatalf("duplicate definition: %s", def.Name)
		}
		seen[def.Name] = true
		if _, ok := wantQueues[def.Queue]; !ok {
			t.Fatalf("%s uses unexpected queue %q", def.Name, def.Queue)
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
		name      string
		adminOnly bool
	}{
		{name: "canonical_backfill_target_sources", adminOnly: true},
		{name: "canonical_backfill_building_coordinates", adminOnly: true},
		{name: "canonical_backfill_detached_houses", adminOnly: false},
	} {
		def, ok := FindDefinition(tc.name)
		if !ok {
			t.Fatalf("%s missing", tc.name)
		}
		if def.Queue != QueueCanonicalDB {
			t.Fatalf("%s queue = %q, want %q", tc.name, def.Queue, QueueCanonicalDB)
		}
		if def.AdminOnly != tc.adminOnly {
			t.Fatalf("%s adminOnly = %v, want %v", tc.name, def.AdminOnly, tc.adminOnly)
		}
	}
	if _, ok := FindDefinition("canonical_rebuild_spatial_read_model"); ok {
		t.Fatal("spatial read model no-op should not be an Absurd task")
	}
}

func TestIdempotencyKeyUsesTaskNameAndCanonicalParams(t *testing.T) {
	t.Parallel()
	a := IdempotencyKey("frontdoor_sync", json.RawMessage(`{"source_type":"ad","source_id":"123"}`))
	b := IdempotencyKey("frontdoor_sync", json.RawMessage(`{ "source_type" : "ad", "source_id" : "123" }`))
	if a != b {
		t.Fatalf("idempotency keys differ: %q != %q", a, b)
	}
	if a == "frontdoor_sync" {
		t.Fatalf("non-empty params should affect idempotency key")
	}
}

func TestValidateTaskParams(t *testing.T) {
	t.Parallel()
	if err := ValidateTaskParams("frontdoor_sync", json.RawMessage(`{"source_type":"ad","source_id":"123"}`)); err != nil {
		t.Fatalf("ValidateTaskParams valid = %v", err)
	}
	err := ValidateTaskParams("missing", json.RawMessage(`{}`))
	if !errors.Is(err, ErrUnknownTask) {
		t.Fatalf("ValidateTaskParams unknown = %v, want ErrUnknownTask", err)
	}
	if err := ValidateTaskParams("frontdoor_sync", json.RawMessage(`{`)); err == nil {
		t.Fatal("ValidateTaskParams invalid JSON succeeded")
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
	for name, want := range cases {
		def, ok := FindDefinition(name)
		if !ok {
			t.Fatalf("%s missing", name)
		}
		if def.Queue != want {
			t.Fatalf("%s queue = %q, want %q", name, def.Queue, want)
		}
	}
}

func TestTaskRegistryMarksExecutableAbsurdTasks(t *testing.T) {
	t.Parallel()
	for _, def := range AllDefinitions() {
		if _, ok := FindDefinition(def.Name); !ok {
			t.Fatalf("%s should be registered", def.Name)
		}
	}
}
