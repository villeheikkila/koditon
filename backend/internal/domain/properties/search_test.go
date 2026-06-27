package properties

import (
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
