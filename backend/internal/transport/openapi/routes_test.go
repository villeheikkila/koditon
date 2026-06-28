package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func TestAddRoutesBuildsSchemas(t *testing.T) {
	api := humago.New(http.NewServeMux(), NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
}

func TestAddRoutesIncludesAddressLookup(t *testing.T) {
	api := humago.New(http.NewServeMux(), NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
	data, err := json.Marshal(api.OpenAPI())
	if err != nil {
		t.Fatalf("marshal openapi: %v", err)
	}
	doc := string(data)
	if !strings.Contains(doc, "/api/v1/address-lookup") {
		t.Fatal("expected address lookup path in OpenAPI document")
	}
	if !strings.Contains(doc, "address-lookup") {
		t.Fatal("expected address lookup operation id in OpenAPI document")
	}
	for _, want := range []string{"connected prices links", "raw prices history", "source-specific offering matches", "raw_transactions", "linked_to_lookup", "candidate_to_lookup", "scope", "is_matched", "matched_listing_count", "source_records", "source_candidates", "candidate_offering_id", "transactions", "matches", "link_type", "external_url_available", "reasons_summary", "canonical_id", "offering_id"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected address lookup OpenAPI schema to include %q", want)
		}
	}
}

func TestAddRoutesIncludesEntityRawPayload(t *testing.T) {
	api := humago.New(http.NewServeMux(), NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
	data, err := json.Marshal(api.OpenAPI())
	if err != nil {
		t.Fatalf("marshal openapi: %v", err)
	}
	doc := string(data)
	for _, want := range []string{"/api/v1/entity", "raw", "pretty", "original_bytes", "external_url_available"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected entity OpenAPI schema to include %q", want)
		}
	}
}

func TestAddRoutesIncludesTransactionMatchReview(t *testing.T) {
	api := humago.New(http.NewServeMux(), NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
	data, err := json.Marshal(api.OpenAPI())
	if err != nil {
		t.Fatalf("marshal openapi: %v", err)
	}
	doc := string(data)
	for _, want := range []string{"/api/v1/property-model/transaction-match-candidates", "transaction-match-candidates", "Review prices matches", "candidate plus linked rows", "price_delta_percent", "listing", "transaction", "canonical_id", "offering_id", "native_id", "external_url_available", "link_type", "link_method"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("expected transaction match OpenAPI schema to include %q", want)
		}
	}
}

func TestAddressLookupRequiresAddress(t *testing.T) {
	mux := http.NewServeMux()
	api := humago.New(mux, NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/address-lookup", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "address is required") {
		t.Fatalf("expected address validation error, got %s", rec.Body.String())
	}
}

func TestTransactionMatchCandidatesRejectsInvalidTransactionID(t *testing.T) {
	mux := http.NewServeMux()
	api := humago.New(mux, NewConfig("Koditon API", "test"))
	a := API{logger: slog.Default()}
	a.AddRoutes(api)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/property-model/transaction-match-candidates?transaction=not-a-uuid", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "transaction must be a valid UUID") {
		t.Fatalf("expected transaction validation error, got %s", rec.Body.String())
	}
}
