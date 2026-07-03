package properties

import (
	"os"
	"strings"
	"testing"
)

func transactionMatchCandidatesSQLForTest(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../../db/queries/ads/property_matches.sql")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestTransactionMatchCandidatesUseListingDocuments(t *testing.T) {
	query := transactionMatchCandidatesSQLForTest(t)
	for _, want := range []string{
		"JOIN public.listing_search_documents doc ON doc.primary_source_listing_id = latest.sale_listing_id",
		"doc.city AS listing_city",
		"doc.postal AS listing_postal",
		"doc.listing_status = 'active'",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected transaction match candidates SQL to include %q", want)
		}
	}
	for _, sourceOnly := range []string{
		"JOIN public.property_source_offerings sl",
		"JOIN public.target_sources source_link",
	} {
		if strings.Contains(query, sourceOnly) {
			t.Fatalf("expected transaction match candidates SQL to avoid %q", sourceOnly)
		}
	}
}

func TestTransactionMatchCandidatesIncludeLinkedRowsForTransactionReview(t *testing.T) {
	query := transactionMatchCandidatesSQLForTest(t)
	for _, want := range []string{
		"review_rows AS",
		"'candidate'::text AS link_type",
		"'match_candidate'::text AS link_method",
		"'listing'::text",
		"'source_listing'::text",
		"FROM public.price_links pl",
		"pl.price_link_id::text || ':' || doc.listing_id::text",
		"JOIN public.listing_search_documents doc ON doc.property_offering_id = pl.target_id",
		"pl.prices_transaction_id = sqlc.narg('transaction_id')::uuid",
		"pl.target_type = 'source_listing'",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected transaction review SQL to include %q", want)
		}
	}
	if !strings.Contains(query, "WHERE (sqlc.narg('transaction_id')::uuid IS NOT NULL OR latest.status = ANY") {
		t.Fatal("expected postal review to remain limited to candidate statuses")
	}
	if strings.Contains(query, "po.primary_sale_listing_id = sl.sale_listing_id") {
		t.Fatal("expected transaction review to include all offering source records, not only primary listings")
	}
}

func TestTransactionMatchCandidatesUseProjectedAvailability(t *testing.T) {
	query := transactionMatchCandidatesSQLForTest(t)
	for _, want := range []string{
		"COALESCE(doc.url, '') <> '' AS external_url_available",
		"doc.listing_status = 'active'",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("expected transaction match SQL to include %q", want)
		}
	}
}
