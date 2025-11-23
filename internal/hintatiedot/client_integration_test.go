package hintatiedot

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestFetchCitiesIntegration(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := NewClient(&http.Client{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	cities, err := client.FetchCities(ctx)
	if err != nil {
		t.Fatalf("FetchCities: %v", err)
	}

	if len(cities) == 0 {
		t.Fatalf("expected at least one city, got %d", len(cities))
	}

	t.Logf("fetched %d cities, sample: %v", len(cities), cities[:min(3, len(cities))])
}

func TestFetchPostalCodesIntegration(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := NewClient(&http.Client{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	postalCodes, err := client.FetchPostalCodes(ctx, "Helsinki")
	if err != nil {
		t.Fatalf("FetchPostalCodes: %v", err)
	}

	if len(postalCodes) == 0 {
		t.Fatalf("expected at least one postal code, got %d", len(postalCodes))
	}

	t.Logf("fetched %d postal codes, sample: %v", len(postalCodes), postalCodes[:min(3, len(postalCodes))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
