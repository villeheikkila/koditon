package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"koditon-go/internal/platform/buildinfo"
	"koditon-go/internal/platform/config"
)

func TestLivezIncludesBuildInfo(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	New(nil, config.AppMode{API: true}, buildinfo.Info{
		Version:   "v1",
		Commit:    "abc123",
		BuildTime: "2026-04-26T00:00:00Z",
	}).Register(mux)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/livez", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["version"] != "v1" || body["commit"] != "abc123" || body["build_time"] != "2026-04-26T00:00:00Z" {
		t.Fatalf("unexpected build info body: %#v", body)
	}
}
