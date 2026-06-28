package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRootHelpListsCanonicalCommands(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(context.Background(), &stdout, &stderr, envGetter(nil))
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"search", "detail", "transactions", "prices", "sync", "api-query", "--json", "--no-color"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in help:\n%s", want, out)
		}
	}
}

func TestPricesHelpListsMatchSubcommand(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(context.Background(), &stdout, &stderr, envGetter(nil))
	cmd.SetArgs([]string{"prices", "match", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "sale-listings") {
		t.Fatalf("expected sale-listings in help:\n%s", out)
	}
}

func TestPricesMatchSaleListingsHelpListsSafetyFlags(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(context.Background(), &stdout, &stderr, envGetter(nil))
	cmd.SetArgs([]string{"prices", "match", "sale-listings", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"--auto-link-safe", "--threshold", "--margin"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in help:\n%s", want, out)
		}
	}
}

func TestSyncHelpListsCanonicalSubcommands(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(context.Background(), &stdout, &stderr, envGetter(nil))
	cmd.SetArgs([]string{"sync", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"enqueue", "status", "run"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in help:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"list", "maintenance"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("did not expect %q in help:\n%s", unwanted, out)
		}
	}
}

func TestSyncRunHelpListsWorkerFlags(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(context.Background(), &stdout, &stderr, envGetter(nil))
	cmd.SetArgs([]string{"sync", "run", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	out := stdout.String()
	for _, want := range []string{"--workers", "--maintenance", "--maintenance-interval"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in help:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{"--stale-after", "--maintenance-limit"} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("did not expect %q in help:\n%s", unwanted, out)
		}
	}
}

func TestLegacySyncShorthandIsRejected(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	cmd := newRootCommand(context.Background(), &stdout, &stderr, envGetter(nil))
	cmd.SetArgs([]string{"sync", "prices", "city", "Helsinki"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidateSyncJobTarget(t *testing.T) {
	t.Parallel()
	if err := validateSyncJobTarget("prices", "prices_sync"); err != nil {
		t.Fatalf("validateSyncJobTarget returned error: %v", err)
	}
	if err := validateSyncJobTarget("frontdoor", "frontdoor_ad_data_hash_backfill"); err != nil {
		t.Fatalf("validateSyncJobTarget returned error: %v", err)
	}
	if err := validateSyncJobTarget("shortcut", "shortcut_ad_data_hash_backfill"); err != nil {
		t.Fatalf("validateSyncJobTarget returned error: %v", err)
	}
	if err := validateSyncJobTarget("canonical", "canonicalize_source_ads_fanout"); err != nil {
		t.Fatalf("validateSyncJobTarget returned error: %v", err)
	}
	if err := validateSyncJobTarget("canonical", "canonical_rebuild_dimension_layer_backfill"); err != nil {
		t.Fatalf("validateSyncJobTarget returned error: %v", err)
	}
	err := validateSyncJobTarget("prices", "frontdoor_sync")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not implemented") {
		t.Fatalf("error = %v", err)
	}
}
