package api

import (
	"testing"

	"koditon/internal/sync/consumers"
)

func TestSourceRefreshTaskFor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		source     string
		kind       string
		nativeID   string
		wantTask   string
		wantType   string
		wantSource string
	}{
		{name: "frontdoor ad", source: "frontdoor", kind: "ad", nativeID: "80503218", wantTask: consumers.TaskTypeFrontdoorSync, wantType: "ad", wantSource: "80503218"},
		{name: "frontdoor building", source: "frontdoor", kind: "building", nativeID: "123456", wantTask: consumers.TaskTypeFrontdoorSync, wantType: "building", wantSource: "123456"},
		{name: "shortcut ad", source: "shortcut", kind: "ad", nativeID: "8347145", wantTask: consumers.TaskTypeShortcutAPISync, wantType: "ad", wantSource: "8347145"},
		{name: "shortcut building", source: "shortcut", kind: "building", nativeID: "0163c58d-0032-4418-9fc9-5dac1e1c4580", wantTask: consumers.TaskTypeShortcutScraperSync, wantType: "building", wantSource: "0163c58d-0032-4418-9fc9-5dac1e1c4580"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := sourceRefreshTaskFor(tt.source, tt.kind, tt.nativeID)
			if err != nil {
				t.Fatalf("sourceRefreshTaskFor returned error: %v", err)
			}
			if got.TaskName != tt.wantTask || got.SourceType != tt.wantType || got.SourceID != tt.wantSource {
				t.Fatalf("sourceRefreshTaskFor = %#v, want task=%s type=%s source=%s", got, tt.wantTask, tt.wantType, tt.wantSource)
			}
		})
	}
}

func TestSourceRefreshTaskForRejectsUnsupportedKind(t *testing.T) {
	t.Parallel()
	if _, err := sourceRefreshTaskFor("frontdoor", "announcement", "123"); err == nil {
		t.Fatal("sourceRefreshTaskFor accepted unsupported frontdoor announcement")
	}
}
