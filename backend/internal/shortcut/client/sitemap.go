package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"koditon-go/internal/logging"
)

type SitemapURLType string

const (
	SitemapURLTypeListing           SitemapURLType = "listing"
	SitemapURLTypeRental            SitemapURLType = "rental"
	SitemapURLTypeBuilding          SitemapURLType = "building"
	SitemapURLTypeBuildingLander    SitemapURLType = "building_lander"
	SitemapURLTypeCommercial        SitemapURLType = "commercial"
	SitemapURLTypePlot              SitemapURLType = "plot"
	SitemapURLTypeFarm              SitemapURLType = "farm"
	SitemapURLTypeGarage            SitemapURLType = "garage"
	SitemapURLTypeVacationHome      SitemapURLType = "vacation_home"
	SitemapURLTypeRentalVacation    SitemapURLType = "rental_vacation"
	SitemapURLTypeRentalGarage      SitemapURLType = "rental_garage"
	SitemapURLTypeCommercialForSale SitemapURLType = "commercial_for_sale"
)

type ShortcutSitemapEntry struct {
	ID   int
	URL  *url.URL
	Type SitemapURLType
}

func (c *Client) GetSitemapEntries(ctx context.Context) ([]ShortcutSitemapEntry, error) {
	indexURL := joinURL(c.sitemapBaseURL, "/sitemaps/index.xml")
	indexXML, err := c.fetchSitemapXML(ctx, indexURL)
	if err != nil {
		return nil, fmt.Errorf("fetch sitemap index: %w", err)
	}
	var indexURLs []string
	for _, loc := range extractLocs(indexXML) {
		if strings.Contains(loc, "/sm_building_") || strings.Contains(loc, "/sm_ad_") {
			indexURLs = append(indexURLs, loc)
		}
	}
	var entries []ShortcutSitemapEntry
	var fetchErrors []error
	for _, sitemapURL := range indexURLs {
		sitemapXML, err := c.fetchSitemapXML(ctx, sitemapURL)
		if err != nil {
			fetchErrors = append(fetchErrors, fmt.Errorf("fetch %s: %w", sitemapURL, err))
			continue
		}
		for _, loc := range extractLocs(sitemapXML) {
			entry, ok := parseShortcutEntry(loc)
			if !ok {
				logging.With(c.logger, logging.Op("shortcut.sitemap.parse")).Warn("unknown url pattern in sitemap, skipping", "url", loc)
				continue
			}
			entries = append(entries, *entry)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if len(entries) == 0 && len(fetchErrors) > 0 {
		return nil, fmt.Errorf("all sitemap fetches failed: %w", errors.Join(fetchErrors...))
	}
	return entries, nil
}

func parseShortcutEntry(raw string) (*ShortcutSitemapEntry, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 1 {
		return nil, false
	}
	var id int
	var entryType SitemapURLType
	if len(parts) == 1 && parts[0] == "talo" {
		id = 0
		entryType = SitemapURLTypeBuildingLander
	} else {
		parsedID, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			return nil, false
		}
		id = parsedID
	}
	if entryType == "" {
		switch {
		case strings.Contains(raw, "/myytavat-asunnot/"):
			entryType = SitemapURLTypeListing
		case strings.Contains(raw, "/vuokra-asunnot/"):
			entryType = SitemapURLTypeRental
		case strings.Contains(raw, "/talo/"):
			entryType = SitemapURLTypeBuilding
		case strings.Contains(raw, "/vuokrattavat-toimitilat/"):
			entryType = SitemapURLTypeCommercial
		case strings.Contains(raw, "/myytavat-toimitilat/"):
			entryType = SitemapURLTypeCommercialForSale
		case strings.Contains(raw, "/myytavat-tontit/"):
			entryType = SitemapURLTypePlot
		case strings.Contains(raw, "/myytavat-metsatilat-ja-maatilat/"):
			entryType = SitemapURLTypeFarm
		case strings.Contains(raw, "/myytavat-autotallit/"):
			entryType = SitemapURLTypeGarage
		case strings.Contains(raw, "/myytavat-loma-asunnot/"):
			entryType = SitemapURLTypeVacationHome
		case strings.Contains(raw, "/vuokrattavat-loma-asunnot/"):
			entryType = SitemapURLTypeRentalVacation
		case strings.Contains(raw, "/vuokrattavat-autotallit/"):
			entryType = SitemapURLTypeRentalGarage
		default:
			return nil, false
		}
	}
	return &ShortcutSitemapEntry{ID: id, URL: u, Type: entryType}, true
}

var locPattern = regexp.MustCompile(`<loc>([^<]+)</loc>`)

func extractLocs(xml string) []string {
	matches := locPattern.FindAllStringSubmatch(xml, -1)
	results := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, strings.TrimSpace(match[1]))
		}
	}
	return results
}

func (c *Client) fetchSitemapXML(ctx context.Context, url string) (string, error) {
	reqCtx := ctx
	if c.requestTimeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, c.requestTimeout)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept-Encoding", "*")
	req.Header.Set("User-Agent", c.userAgent)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return "", &HTTPStatusError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	return string(body), nil
}
