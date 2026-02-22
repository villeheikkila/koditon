package ads

import "testing"

func TestNormalizeSearchParamsDefaults(t *testing.T) {
	normalized := normalizeSearchParams(SearchParams{})
	if normalized.Source != "all" {
		t.Fatalf("expected source all, got %s", normalized.Source)
	}
	if normalized.Kind != "all" {
		t.Fatalf("expected kind all, got %s", normalized.Kind)
	}
	if normalized.Sort != "seen_desc" {
		t.Fatalf("expected sort seen_desc, got %s", normalized.Sort)
	}
	if normalized.Page != 1 {
		t.Fatalf("expected page 1, got %d", normalized.Page)
	}
	if normalized.PageSize != 50 {
		t.Fatalf("expected page size 50, got %d", normalized.PageSize)
	}
}

func TestNormalizePageSizeAllowedValues(t *testing.T) {
	cases := []int32{25, 50, 100}
	for _, c := range cases {
		got := normalizePageSize(c)
		if got != c {
			t.Fatalf("expected %d, got %d", c, got)
		}
	}
}

func TestNormalizeSortFallback(t *testing.T) {
	got := normalizeSort("unknown")
	if got != "seen_desc" {
		t.Fatalf("expected seen_desc, got %s", got)
	}
}
