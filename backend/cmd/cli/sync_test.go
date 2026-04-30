package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"koditon/cmd/cli/internal/cli"
	"koditon/internal/db"
	syncjobs "koditon/internal/sync/jobs"
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
	for _, want := range []string{"enqueue", "status", "list", "maintenance", "run"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in help:\n%s", want, out)
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
	for _, want := range []string{"--workers", "--maintenance", "--maintenance-interval", "--stale-after", "--maintenance-limit"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in help:\n%s", want, out)
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
	err := validateSyncJobTarget("prices", "frontdoor_sync")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not valid") {
		t.Fatalf("error = %v", err)
	}
}

func TestFormatSyncJobSnapshot(t *testing.T) {
	t.Parallel()
	snapshot := syncjobs.JobSnapshot{Job: db.SyncJob{
		SyncJobProvider:     "prices",
		SyncJobKind:         "prices_sync",
		SyncJobEntityID:     "city:Helsinki",
		SyncJobStatus:       syncjobs.StatusPending,
		SyncJobAttemptCount: 0,
		SyncJobMaxAttempts:  3,
	}}
	got := cli.FormatSyncJobSnapshot(snapshot)
	for _, want := range []string{"status=pending", "provider=prices", "kind=prices_sync", "entity=city:Helsinki", "attempts=0/3"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected %q in %q", want, got)
		}
	}
}
