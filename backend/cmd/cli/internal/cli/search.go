package cli

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"koditon-go/internal/domain/ads"
)

type SearchFlags struct {
	Query           string
	Source          string
	Kind            string
	ListingType     string
	City            string
	Postal          string
	MinPrice        int
	MaxPrice        int
	MinArea         float64
	MaxArea         float64
	Sort            string
	Limit           int
	Page            int
	PublishedAfter  string
	PublishedBefore string
}

func RunSearch(ctx context.Context, svc *ads.Service, f SearchFlags) error {
	params := ads.SearchParams{
		Query:       f.Query,
		Source:      f.Source,
		Kind:        f.Kind,
		ListingType: f.ListingType,
		City:        f.City,
		Postal:      f.Postal,
		Sort:        f.Sort,
		Page:        int32(f.Page),
	}
	switch f.Limit {
	case 25, 50, 100:
		params.PageSize = int32(f.Limit)
	default:
		params.PageSize = 25
	}
	if f.MinPrice > 0 {
		v := int64(f.MinPrice)
		params.MinPrice = &v
	}
	if f.MaxPrice > 0 {
		v := int64(f.MaxPrice)
		params.MaxPrice = &v
	}
	if f.MinArea > 0 {
		params.MinArea = &f.MinArea
	}
	if f.MaxArea > 0 {
		params.MaxArea = &f.MaxArea
	}
	if f.PublishedAfter != "" {
		t, err := time.Parse("2006-01-02", f.PublishedAfter)
		if err != nil {
			return fmt.Errorf("parse --after date: %w", err)
		}
		params.PublishedAfter = &t
	}
	if f.PublishedBefore != "" {
		t, err := time.Parse("2006-01-02", f.PublishedBefore)
		if err != nil {
			return fmt.Errorf("parse --before date: %w", err)
		}
		params.PublishedBefore = &t
	}

	result, err := svc.Search(ctx, params)
	if err != nil {
		return fmt.Errorf("search: %w", err)
	}

	totalPages := max(int(math.Ceil(float64(result.Total)/float64(result.PageSize))), 1)

	fmt.Println(headerStyle.Render(fmt.Sprintf("Found %d results (page %d of %d)", result.Total, result.Page, totalPages)))
	fmt.Println()

	if len(result.Rows) == 0 {
		fmt.Println(mutedStyle.Render("No results found."))
		return nil
	}

	headers := []string{"ID", "Kind", "Headline", "City", "Postal", "Price", "Area", "Source"}
	rows := make([][]string, 0, len(result.Rows))
	for _, r := range result.Rows {
		rows = append(rows, []string{
			r.CanonicalID,
			r.Kind,
			truncate(r.Headline, 40),
			r.City,
			r.Postal,
			formatPrice(r.Price),
			formatArea(r.Area),
			r.Source,
		})
	}

	fmt.Print(renderTable(headers, rows))

	if len(result.Rows) > 0 {
		fmt.Println()
		var filters []string
		if f.City != "" {
			filters = append(filters, "city="+f.City)
		}
		if f.Kind != "" && f.Kind != "all" {
			filters = append(filters, "kind="+f.Kind)
		}
		if f.Source != "" && f.Source != "all" {
			filters = append(filters, "source="+f.Source)
		}
		filterStr := ""
		if len(filters) > 0 {
			filterStr = " [" + strings.Join(filters, ", ") + "]"
		}
		fmt.Println(mutedStyle.Render(fmt.Sprintf("Showing %d of %d%s", len(result.Rows), result.Total, filterStr)))
	}

	return nil
}
