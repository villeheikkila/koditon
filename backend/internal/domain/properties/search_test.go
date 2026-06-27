package properties

import (
	"os"
	"strings"
	"testing"
)

func TestSaleListingSearchUsesNormalizedLocationFallbacks(t *testing.T) {
	for _, sql := range []string{searchSaleListingsSQL, countSaleListingsSQL} {
		if strings.Contains(sql, "lower(COALESCE(sl.sale_listing_city, ''))") {
			t.Fatal("expected city filters to fall back to normalized city")
		}
		if strings.Contains(sql, "lower(COALESCE(sl.sale_listing_postal, ''))") {
			t.Fatal("expected postal filters to fall back to normalized postal")
		}
	}
	if !strings.Contains(searchSaleListingsSQL, "COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')") {
		t.Fatal("expected search rows to expose normalized city fallback")
	}
	if !strings.Contains(searchSaleListingsSQL, "COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '')") {
		t.Fatal("expected search rows to expose normalized postal fallback")
	}
}

func TestPropertySearchSQLAvoidsRawOnlyListingLocation(t *testing.T) {
	source, err := os.ReadFile("search.go")
	if err != nil {
		t.Fatalf("read search source: %v", err)
	}
	text := string(source)
	for _, rawOnly := range []string{
		"COALESCE(sl.sale_listing_city, '')",
		"COALESCE(sl.sale_listing_postal, '')",
		"COALESCE(sl.sale_listing_postal, pb.housing_company_postal_norm",
	} {
		if strings.Contains(text, rawOnly) {
			t.Fatalf("expected property search SQL to avoid raw-only location expression %q", rawOnly)
		}
	}
}
