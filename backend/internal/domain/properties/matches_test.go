package properties

import (
	"strings"
	"testing"
)

func TestTransactionMatchCandidatesUseListingLocationFallbacks(t *testing.T) {
	for _, want := range []string{
		"COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')",
		"COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '')",
	} {
		if !strings.Contains(transactionMatchCandidatesSQL, want) {
			t.Fatalf("expected transaction match candidates SQL to include %q", want)
		}
	}
	for _, rawOnly := range []string{
		"COALESCE(sl.sale_listing_city, '')",
		"COALESCE(sl.sale_listing_postal_norm, '')",
	} {
		if strings.Contains(transactionMatchCandidatesSQL, rawOnly) {
			t.Fatalf("expected transaction match candidates SQL to avoid raw-only location expression %q", rawOnly)
		}
	}
}

func TestTransactionMatchCandidatesIncludeLinkedRowsForTransactionReview(t *testing.T) {
	for _, want := range []string{
		"review_rows AS",
		"'candidate'::text AS link_type",
		"'match_candidate'::text AS link_method",
		"'listing'::text",
		"'source_listing'::text",
		"FROM public.price_links pl",
		"pl.price_link_id::text || ':' || sl.sale_listing_id::text",
		"JOIN public.target_sources source_link ON source_link.target_type = 'listing'",
		"JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id",
		"pl.prices_transaction_id = $3::uuid",
		"pl.target_type = 'source_listing'",
	} {
		if !strings.Contains(transactionMatchCandidatesSQL, want) {
			t.Fatalf("expected transaction review SQL to include %q", want)
		}
	}
	if !strings.Contains(transactionMatchCandidatesSQL, "WHERE ($3::uuid IS NOT NULL OR latest.status = ANY") {
		t.Fatal("expected postal review to remain limited to candidate statuses")
	}
	if strings.Contains(transactionMatchCandidatesSQL, "po.primary_sale_listing_id = sl.sale_listing_id") {
		t.Fatal("expected transaction review to include all offering source records, not only primary listings")
	}
}

func TestTransactionMatchCandidatesUseLiveShortcutAdAvailability(t *testing.T) {
	for _, want := range []string{
		"sl.sale_listing_source_provider = 'shortcut' AND sl.sale_listing_source_kind = 'ad'",
		"COALESCE(sl.sale_listing_url, '') <> '' AND sl.sale_listing_last_seen_at >= now() - interval '7 days'",
	} {
		if !strings.Contains(transactionMatchCandidatesSQL, want) {
			t.Fatalf("expected transaction match SQL to include %q", want)
		}
	}
}
