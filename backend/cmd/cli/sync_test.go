package main

import (
	"testing"
	"time"

	"koditon/cmd/cli/internal/cli"
)

func TestResolveSyncFlagsAliases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		args     []string
		provider string
		kind     string
		entityID string
	}{
		{name: "frontdoor sitemap", args: []string{"frontdoor", "sitemap"}, provider: "frontdoor", kind: "frontdoor_sitemap_sync", entityID: "frontdoor:sitemap"},
		{name: "frontdoor ad", args: []string{"frontdoor", "ad", "abc123"}, provider: "frontdoor", kind: "frontdoor_sync", entityID: "ad:abc123"},
		{name: "shortcut buildings", args: []string{"shortcut", "buildings"}, provider: "shortcut", kind: "shortcut_buildings_sitemap_sync", entityID: "shortcut:buildings_sitemap"},
		{name: "prices city", args: []string{"prices", "city", "Helsinki"}, provider: "prices", kind: "prices_sync", entityID: "city:Helsinki"},
		{name: "postal all", args: []string{"postal", "all"}, provider: "postal", kind: "postal_sync", entityID: "postal:all"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := resolveSyncFlags(tc.args, cli.SyncFlags{Watch: true, Interval: time.Second})
			if err != nil {
				t.Fatalf("resolveSyncFlags returned error: %v", err)
			}
			if got.Provider != tc.provider || got.Kind != tc.kind || got.EntityID != tc.entityID {
				t.Fatalf("got provider=%q kind=%q entity=%q", got.Provider, got.Kind, got.EntityID)
			}
			if !got.Watch || got.Interval != time.Second {
				t.Fatalf("flags were not preserved: %#v", got)
			}
		})
	}
}

func TestResolveSyncFlagsGeneric(t *testing.T) {
	t.Parallel()
	got, err := resolveSyncFlags(nil, cli.SyncFlags{Provider: "prices", Kind: "prices_sync", EntityID: "city:Espoo"})
	if err != nil {
		t.Fatalf("resolveSyncFlags returned error: %v", err)
	}
	if got.Provider != "prices" || got.Kind != "prices_sync" || got.EntityID != "city:Espoo" {
		t.Fatalf("unexpected flags: %#v", got)
	}
}

func TestResolveSyncFlagsRejectsPartialGeneric(t *testing.T) {
	t.Parallel()
	_, err := resolveSyncFlags(nil, cli.SyncFlags{Provider: "prices", Kind: "prices_sync"})
	if err == nil {
		t.Fatal("expected error")
	}
}
