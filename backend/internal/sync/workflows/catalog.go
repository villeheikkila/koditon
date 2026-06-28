package workflows

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/earendil-works/absurd/sdks/go/absurd"
)

const (
	QueueFrontdoor       = "frontdoor"
	QueueShortcutAPI     = "shortcut_api"
	QueueShortcutScraper = "shortcut_scraper"
	QueuePrices          = "prices"
	QueuePostal          = "postal"
	QueueCanonicalDB     = "canonical_db"
	QueueCanonicalLLM    = "canonical_llm"
)

// Definition is the Absurd task contract for a sync workflow.
type Definition struct {
	Provider            string
	Kind                string
	Queue               string
	DefaultMaxAttempts  int
	DefaultCancellation *absurd.CancellationPolicy
	AdminOnly           bool
	Implemented         bool
}

// Params is the compatibility payload for old provider/kind/entity job callers.
type Params struct {
	Provider string          `json:"provider"`
	Kind     string          `json:"kind"`
	EntityID string          `json:"entity_id"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

// Result is the common JSON result envelope returned by compatibility tasks.
type Result struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result,omitempty"`
}

// Handler executes a registered sync workflow task.
type Handler func(context.Context, Params) (Result, error)

// SpawnRequest is the caller-facing request for spawning an Absurd sync task.
type SpawnRequest struct {
	Provider     string
	Kind         string
	EntityID     string
	Payload      json.RawMessage
	MaxAttempts  int
	Cancellation *absurd.CancellationPolicy
}

// ErrUnknownTask reports a provider/kind pair that is not part of the Absurd sync catalog.
var ErrUnknownTask = errors.New("unknown sync workflow task")

type spawnClient interface {
	Spawn(context.Context, string, any, ...absurd.SpawnOptions) (absurd.SpawnResult, error)
}

var definitions = []Definition{
	{Provider: "frontdoor", Kind: "frontdoor_sitemap_sync", Queue: QueueFrontdoor, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "frontdoor", Kind: "frontdoor_buildings_sitemap_sync", Queue: QueueFrontdoor, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "frontdoor", Kind: "frontdoor_sync", Queue: QueueFrontdoor, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "frontdoor", Kind: "frontdoor_ad_data_hash_backfill", Queue: QueueFrontdoor, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "shortcut", Kind: "shortcut_sitemap_sync", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "shortcut", Kind: "shortcut_buildings_sitemap_sync", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "shortcut", Kind: "shortcut_scraper_sync", Queue: QueueShortcutScraper, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "shortcut", Kind: "shortcut_api_sync", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "shortcut", Kind: "shortcut_ad_data_hash_backfill", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_cities_init", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_sync", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_postal_code_sync", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_postal_code_page_sync", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_neighborhood_postal_code_sync", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_sync_all", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_match_sale_listings_backfill", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_match_sale_listings_fanout", Queue: QueuePrices, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "prices", Kind: "prices_match_sale_listing", Queue: QueuePrices, DefaultMaxAttempts: 3, DefaultCancellation: &absurd.CancellationPolicy{MaxDuration: 15552000}, Implemented: true},
	{Provider: "canonical", Kind: "canonicalize_source_ads_fanout", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonicalize_source_ad", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_match_sale_listing_sources_backfill", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_match_sale_listing_sources_fanout", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_match_sale_listing_source", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_rebuild_dimension_layer_backfill", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_rebuild_dimension_layer_listing", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_resolve_dirty_dimension_targets", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_resolve_dimension_target", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_extract_manager_certificate", Queue: QueueCanonicalLLM, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_project_manager_certificate", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "canonical", Kind: "canonical_backfill_target_sources", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, AdminOnly: true, Implemented: true},
	{Provider: "canonical", Kind: "canonical_backfill_building_coordinates", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, AdminOnly: true, Implemented: true},
	{Provider: "canonical", Kind: "canonical_backfill_detached_houses", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, Implemented: true},
	{Provider: "postal", Kind: "postal_sync", Queue: QueuePostal, DefaultMaxAttempts: 3, Implemented: true},
}

func AllDefinitions() []Definition {
	out := make([]Definition, len(definitions))
	copy(out, definitions)
	return out
}

// FindDefinition resolves a provider/kind pair from the Absurd sync catalog.
func FindDefinition(provider, kind string) (Definition, bool) {
	provider = strings.TrimSpace(provider)
	kind = strings.TrimSpace(kind)
	for _, def := range definitions {
		if def.Provider == provider && def.Kind == kind {
			return def, true
		}
	}
	return Definition{}, false
}

// IsImplemented reports whether an Absurd worker currently executes the task.
func IsImplemented(provider, kind string) bool {
	def, ok := FindDefinition(provider, kind)
	return ok && def.Implemented
}

// QueueNames returns every target Absurd queue needed by sync workflows.
func QueueNames() []string {
	return []string{QueueFrontdoor, QueueShortcutAPI, QueueShortcutScraper, QueuePrices, QueuePostal, QueueCanonicalDB, QueueCanonicalLLM}
}

// IdempotencyKey preserves the legacy provider:kind:entity_id dedup shape.
func IdempotencyKey(provider, kind, entityID string) string {
	return strings.Join([]string{strings.TrimSpace(provider), strings.TrimSpace(kind), strings.TrimSpace(entityID)}, ":")
}

// ValidateParams validates the common task payload before spawn or execution.
func ValidateParams(params Params) error {
	if strings.TrimSpace(params.Provider) == "" || strings.TrimSpace(params.Kind) == "" || strings.TrimSpace(params.EntityID) == "" {
		return errors.New("provider, kind, and entity id are required")
	}
	if _, ok := FindDefinition(params.Provider, params.Kind); !ok {
		return fmt.Errorf("%w: %s/%s", ErrUnknownTask, params.Provider, params.Kind)
	}
	if len(params.Payload) > 0 && !json.Valid(params.Payload) {
		return errors.New("payload must be valid JSON")
	}
	return nil
}

// RegisterAll registers every sync workflow task on an Absurd client.
func RegisterAll(app *absurd.Client, handler Handler) error {
	if app == nil {
		return errors.New("absurd client is required")
	}
	if handler == nil {
		return errors.New("workflow handler is required")
	}
	for _, def := range definitions {
		task := absurd.Task[Params, Result](def.Kind, absurd.TaskHandler[Params, Result](handler), absurd.TaskOptions{QueueName: def.Queue, DefaultMaxAttempts: def.DefaultMaxAttempts, DefaultCancellation: def.DefaultCancellation})
		if err := app.Register(task); err != nil {
			return fmt.Errorf("register %s: %w", def.Kind, err)
		}
	}
	return nil
}

// Spawn starts a sync workflow task with old enqueue dedup semantics.
func Spawn(ctx context.Context, app spawnClient, req SpawnRequest) (absurd.SpawnResult, error) {
	if app == nil {
		return absurd.SpawnResult{}, errors.New("absurd client is required")
	}
	def, ok := FindDefinition(req.Provider, req.Kind)
	if !ok {
		return absurd.SpawnResult{}, fmt.Errorf("%w: %s/%s", ErrUnknownTask, req.Provider, req.Kind)
	}
	params := Params{Provider: strings.TrimSpace(req.Provider), Kind: strings.TrimSpace(req.Kind), EntityID: strings.TrimSpace(req.EntityID), Payload: req.Payload}
	if err := ValidateParams(params); err != nil {
		return absurd.SpawnResult{}, err
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = def.DefaultMaxAttempts
	}
	cancellation := req.Cancellation
	if cancellation == nil {
		cancellation = def.DefaultCancellation
	}
	if cancellation == nil {
		cancellation = &absurd.CancellationPolicy{MaxDelay: 35 * 60}
	}
	return app.Spawn(ctx, def.Kind, params, absurd.SpawnOptions{
		QueueName:      def.Queue,
		MaxAttempts:    maxAttempts,
		IdempotencyKey: IdempotencyKey(params.Provider, params.Kind, params.EntityID),
		RetryStrategy:  &absurd.RetryStrategy{Kind: "exponential", BaseSeconds: 5, Factor: 2, MaxSeconds: 300},
		Cancellation:   cancellation,
	})
}
