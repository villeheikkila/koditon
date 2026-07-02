package properties

import (
	"os"
	"strings"
	"testing"
)

func TestSaleListingSearchUsesNormalizedLocationFallbacks(t *testing.T) {
	sql := readPropertySearchSQL(t)
	for _, queryName := range []string{"SearchSaleListings", "CountSaleListings"} {
		query := namedSQLSection(t, sql, queryName)
		if strings.Contains(query, "lower(COALESCE(sl.sale_listing_city, ''))") {
			t.Fatal("expected city filters to fall back to normalized city")
		}
		if strings.Contains(query, "lower(COALESCE(sl.sale_listing_postal, ''))") {
			t.Fatal("expected postal filters to fall back to normalized postal")
		}
	}
	searchQuery := namedSQLSection(t, sql, "SearchSaleListings")
	if !strings.Contains(searchQuery, "COALESCE(pso.sale_listing_city, pso.sale_listing_city_norm, '')") {
		t.Fatal("expected search rows to expose normalized city fallback")
	}
	if !strings.Contains(searchQuery, "COALESCE(pso.sale_listing_postal, pso.sale_listing_postal_norm, '')") {
		t.Fatal("expected search rows to expose normalized postal fallback")
	}
}

func TestPropertySearchSQLAvoidsRawOnlyListingLocation(t *testing.T) {
	text := readPropertySearchSQL(t)
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

func readPropertySearchSQL(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile("../../../db/queries/ads/property_search.sql")
	if err != nil {
		t.Fatalf("read property search sql: %v", err)
	}
	return string(source)
}

func namedSQLSection(t *testing.T, sql string, name string) string {
	t.Helper()
	startMarker := "-- name: " + name + " "
	start := strings.Index(sql, startMarker)
	if start == -1 {
		t.Fatalf("missing sqlc query %s", name)
	}
	rest := sql[start+len(startMarker):]
	before, _, ok := strings.Cut(rest, "\n-- name: ")
	if !ok {
		return rest
	}
	return before
}
