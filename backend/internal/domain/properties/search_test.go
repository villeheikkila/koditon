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
		for _, want := range []string{
			"FROM public.listing_search_documents doc",
			"lower(COALESCE(doc.city, ''))",
			"lower(COALESCE(doc.postal, ''))",
		} {
			if !strings.Contains(query, want) {
				t.Fatalf("expected %s to include %q", queryName, want)
			}
		}
	}
	searchQuery := namedSQLSection(t, sql, "SearchSaleListings")
	if !strings.Contains(searchQuery, "doc.city") {
		t.Fatal("expected search rows to expose projected city")
	}
	if !strings.Contains(searchQuery, "doc.postal") {
		t.Fatal("expected search rows to expose projected postal")
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

func TestRentalSearchDoesNotReadOriginTables(t *testing.T) {
	sql := readPropertySearchSQL(t)
	for _, queryName := range []string{"SearchRentalListings", "CountRentalListings", "ListRentalCanonicalIDs", "ListBuildingCanonicalIDs"} {
		query := namedSQLSection(t, sql, queryName)
		if strings.Contains(query, "origin.") {
			t.Fatalf("expected %s to avoid origin tables", queryName)
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
