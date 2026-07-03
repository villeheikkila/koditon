package workflows

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/earendil-works/absurd/sdks/go/absurd"

	"koditon/internal/platform/telemetry"
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

type Definition struct {
	Name                string
	Queue               string
	DefaultMaxAttempts  int
	DefaultCancellation *absurd.CancellationPolicy
	AdminOnly           bool
}

type SpawnTaskRequest struct {
	TaskName       string
	Params         json.RawMessage
	MaxAttempts    int
	Cancellation   *absurd.CancellationPolicy
	IdempotencyKey string
}

var ErrUnknownTask = errors.New("unknown sync workflow task")

type spawnClient interface {
	Spawn(context.Context, string, any, ...absurd.SpawnOptions) (absurd.SpawnResult, error)
}

var definitions = []Definition{
	{Name: "frontdoor_sitemap_sync", Queue: QueueFrontdoor, DefaultMaxAttempts: 3},
	{Name: "frontdoor_buildings_sitemap_sync", Queue: QueueFrontdoor, DefaultMaxAttempts: 3},
	{Name: "frontdoor_sync", Queue: QueueFrontdoor, DefaultMaxAttempts: 3},
	{Name: "frontdoor_ad_data_hash_backfill", Queue: QueueFrontdoor, DefaultMaxAttempts: 3},
	{Name: "shortcut_sitemap_sync", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3},
	{Name: "shortcut_buildings_sitemap_sync", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3},
	{Name: "shortcut_scraper_sync", Queue: QueueShortcutScraper, DefaultMaxAttempts: 3},
	{Name: "shortcut_api_sync", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3},
	{Name: "shortcut_ad_data_hash_backfill", Queue: QueueShortcutAPI, DefaultMaxAttempts: 3},
	{Name: "prices_match_sale_listings_backfill", Queue: QueuePrices, DefaultMaxAttempts: 3},
	{Name: "prices_match_sale_listings_fanout", Queue: QueuePrices, DefaultMaxAttempts: 3},
	{Name: "prices_match_sale_listing", Queue: QueuePrices, DefaultMaxAttempts: 3, DefaultCancellation: &absurd.CancellationPolicy{MaxDuration: 15552000}},
	{Name: "canonicalize_source_ads_fanout", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonicalize_source_ad", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonical_rebuild_dimension_layer_backfill", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonical_rebuild_dimension_layer_listing", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonical_resolve_dirty_dimension_targets", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonical_resolve_dimension_target", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonical_extract_manager_certificate", Queue: QueueCanonicalLLM, DefaultMaxAttempts: 3},
	{Name: "canonical_project_manager_certificate", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "canonical_backfill_building_coordinates", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3, AdminOnly: true},
	{Name: "canonical_backfill_detached_houses", Queue: QueueCanonicalDB, DefaultMaxAttempts: 3},
	{Name: "postal_sync", Queue: QueuePostal, DefaultMaxAttempts: 3},
}

func AllDefinitions() []Definition {
	out := make([]Definition, len(definitions))
	copy(out, definitions)
	return out
}

func FindDefinition(taskName string) (Definition, bool) {
	taskName = strings.TrimSpace(taskName)
	for _, def := range definitions {
		if def.Name == taskName {
			return def, true
		}
	}
	return Definition{}, false
}

func QueueNames() []string {
	return []string{QueueFrontdoor, QueueShortcutAPI, QueueShortcutScraper, QueuePrices, QueuePostal, QueueCanonicalDB, QueueCanonicalLLM}
}

func ValidateTaskParams(taskName string, params json.RawMessage) error {
	if strings.TrimSpace(taskName) == "" {
		return errors.New("task name is required")
	}
	if _, ok := FindDefinition(taskName); !ok {
		return fmt.Errorf("%w: %s", ErrUnknownTask, taskName)
	}
	if len(params) == 0 {
		return nil
	}
	if !json.Valid(params) {
		return errors.New("params must be valid JSON")
	}
	return nil
}

func Spawn(ctx context.Context, app spawnClient, req SpawnTaskRequest) (absurd.SpawnResult, error) {
	if app == nil {
		return absurd.SpawnResult{}, errors.New("absurd client is required")
	}
	def, ok := FindDefinition(req.TaskName)
	if !ok {
		return absurd.SpawnResult{}, fmt.Errorf("%w: %s", ErrUnknownTask, req.TaskName)
	}
	params := normalizeParams(req.Params)
	if err := ValidateTaskParams(def.Name, params); err != nil {
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
	idempotencyKey := strings.TrimSpace(req.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = IdempotencyKey(def.Name, params)
	}
	return app.Spawn(ctx, def.Name, params, absurd.SpawnOptions{
		QueueName:      def.Queue,
		MaxAttempts:    maxAttempts,
		IdempotencyKey: idempotencyKey,
		RetryStrategy:  &absurd.RetryStrategy{Kind: "exponential", BaseSeconds: 5, Factor: 2, MaxSeconds: 300},
		Cancellation:   cancellation,
		Headers:        telemetry.InjectWorkflowTraceHeaders(ctx, nil),
	})
}

func IdempotencyKey(taskName string, params json.RawMessage) string {
	params = normalizeParams(params)
	if string(params) == "{}" {
		return strings.TrimSpace(taskName)
	}
	sum := sha256.Sum256(params)
	return strings.TrimSpace(taskName) + ":" + hex.EncodeToString(sum[:])[:24]
}

func CronSlotIdempotencyKey(taskName, scheduleName string, slot time.Time) string {
	slotText := slot.UTC().Truncate(time.Minute).Format("2006-01-02T15:04")
	raw := strings.TrimSpace(taskName) + "|" + strings.TrimSpace(scheduleName) + "|" + slotText
	sum := sha256.Sum256([]byte(raw))
	return "cron:" + hex.EncodeToString(sum[:])[:24]
}

func normalizeParams(params json.RawMessage) json.RawMessage {
	if len(params) == 0 || string(params) == "null" {
		return json.RawMessage(`{}`)
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, params); err != nil {
		return params
	}
	return json.RawMessage(buf.Bytes())
}
