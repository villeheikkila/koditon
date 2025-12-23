package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var (
	ErrScraperErrorPage  = errors.New("shortcut scraper: page returned an error")
	ErrScraperForbidden  = errors.New("shortcut scraper: forbidden")
	ErrUnexpectedColumns = errors.New("shortcut scraper: unexpected column count")
)

type ScrapedBuilding struct {
	ShortcutBuildingID      int
	BuildingID              *string
	BuildingType            *string
	BuildingSubtype         *string
	ConstructionYear        *int
	FloorCount              *int
	ApartmentCount          *int
	HeatingSystem           *string
	BuildingMaterial        *string
	PlotType                *string
	WallStructure           *string
	HeatSource              *string
	HasElevator             *string
	HasSauna                *string
	Latitude                *float64
	Longitude               *float64
	AdditionalAddresses     *string
	Address                 string
	FrameConstructionMethod *string
	HousingCompany          *string
}

type BuildingListing struct {
	Index         int
	Layout        *string
	Size          *float64
	Price         *float64
	PricePerSqm   *float64
	DeletedAt     *time.Time
	MarketingTime *string
}

type RentalListing struct {
	Index         int
	Layout        *string
	Size          *float64
	Price         *float64
	DeletedAt     *time.Time
	MarketingTime *string
}

type SitemapType string

const (
	SitemapTypeAd       SitemapType = "ad"
	SitemapTypeBuilding SitemapType = "building"
)

type SitemapEntry struct {
	URL string
	ID  string
}

func (c *Client) ScrapeBuildingPage(ctx context.Context, shortcutBuildingID int, pageURL string) (*ScrapedBuilding, []BuildingListing, []RentalListing, error) {
	html, err := c.fetchHTML(ctx, pageURL)
	if err != nil {
		return nil, nil, nil, err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(html))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("parse html: %w", err)
	}
	if isErrorPage(doc) {
		return nil, nil, nil, ErrScraperErrorPage
	}
	address, err := parseAddress(doc)
	if err != nil {
		return nil, nil, nil, err
	}
	building := &ScrapedBuilding{
		ShortcutBuildingID: shortcutBuildingID,
		Address:            address,
	}
	applyBasicInfo(doc, building)
	lat, lon := parseCoordinates(doc)
	building.Latitude = lat
	building.Longitude = lon
	listings, err := parseListings(doc)
	if err != nil {
		return nil, nil, nil, err
	}
	rentals, err := parseRentalListings(doc)
	if err != nil {
		return nil, nil, nil, err
	}
	return building, listings, rentals, nil
}

func (c *Client) GetAllBuildingPages(ctx context.Context) ([]SitemapEntry, error) {
	return c.getAllSitemapPages(ctx, SitemapTypeBuilding)
}

func (c *Client) GetAllAdPages(ctx context.Context) ([]SitemapEntry, error) {
	return c.getAllSitemapPages(ctx, SitemapTypeAd)
}

func (c *Client) getAllSitemapPages(ctx context.Context, t SitemapType) ([]SitemapEntry, error) {
	const (
		maxFailedAttempts = 3
		sleepBetween      = time.Second
	)
	base := joinURL(c.sitemapBaseURL, fmt.Sprintf("/sitemaps/sm_%s_", t))
	var (
		results []SitemapEntry
		failed  int
		lastErr error
	)
	for index := 1; failed < maxFailedAttempts; index++ {
		url := fmt.Sprintf("%s%d.xml", base, index)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build sitemap request: %w", err)
		}
		req.Header.Set("Accept", "*/*")
		req.Header.Set("Accept-Encoding", "*")
		req.Header.Set("User-Agent", c.userAgent)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request sitemap %d: %w", index, err)
			failed++
			continue
		}
		// Ensure body is always closed to prevent file descriptor leaks
		body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		closeErr := resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("read sitemap %d: %w", index, err)
			failed++
			continue
		}
		if closeErr != nil {
			lastErr = fmt.Errorf("close sitemap %d response: %w", index, closeErr)
			failed++
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = &HTTPStatusError{StatusCode: resp.StatusCode, Body: ""}
			failed++
			continue
		}
		newEntries := c.parseSitemap(body, t)
		if len(newEntries) == 0 {
			lastErr = fmt.Errorf("sitemap %d contained no entries", index)
			failed++
			continue
		}
		failed = 0
		results = append(results, newEntries...)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		time.Sleep(sleepBetween)
	}
	if failed >= maxFailedAttempts && lastErr != nil {
		return results, lastErr
	}
	return results, nil
}

func (c *Client) fetchHTML(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build scraper request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "*")
	req.Header.Set("Accept-Encoding", "*")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform scraper request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden {
		return nil, ErrScraperForbidden
	}
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Body: ""}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read scraper response: %w", err)
	}
	return body, nil
}

func isErrorPage(doc *goquery.Document) bool {
	body := doc.Find("body")
	return body.HasClass("error-page")
}

func parseAddress(doc *goquery.Document) (string, error) {
	selectors := []string{"h1.hero__title", ".hero h1", "h1"}
	for _, selector := range selectors {
		if text := strings.TrimSpace(doc.Find(selector).First().Text()); text != "" {
			if strings.Contains(strings.ToLower(text), "403 forbidden") {
				return "", ErrScraperForbidden
			}
			return text, nil
		}
	}
	return "", ErrScraperErrorPage
}

func applyBasicInfo(doc *goquery.Document, building *ScrapedBuilding) {
	doc.Find(".info-table__row").Each(func(_ int, row *goquery.Selection) {
		titleSel := row.Find(".info-table__title").First()
		valueSel := row.Find(".info-table__value").First()
		title := strings.TrimSpace(titleSel.Text())
		value := strings.TrimSpace(valueSel.Text())
		if title == "" || value == "" {
			return
		}
		switch title {
		case "Rakennustunnus":
			building.BuildingID = stringPtr(value)
		case "Talotyyppi":
			building.BuildingType = stringPtr(value)
		case "Talotyyppi tarkemmin":
			building.BuildingSubtype = stringPtr(value)
		case "Rakennusvuosi":
			if v, ok := parseInt(value); ok {
				building.ConstructionYear = &v
			}
		case "Kerroksia":
			if v, ok := parseInt(value); ok {
				building.FloorCount = &v
			}
		case "Huoneistoja":
			if v, ok := parseInt(value); ok {
				building.ApartmentCount = &v
			}
		case "Lämmitys":
			building.HeatingSystem = stringPtr(value)
		case "Rakennusmateriaali":
			building.BuildingMaterial = stringPtr(value)
		case "Tontti":
			building.PlotType = stringPtr(value)
		case "Seinärakenne":
			building.WallStructure = stringPtr(value)
		case "Lämmönlähde":
			building.HeatSource = stringPtr(value)
		case "Hissi":
			building.HasElevator = stringPtr(value)
		case "Sauna":
			building.HasSauna = stringPtr(value)
		case "Muut osoitteet":
			building.AdditionalAddresses = stringPtr(value)
		case "Rungon rakennustapa":
			building.FrameConstructionMethod = stringPtr(value)
		case "Taloyhtiö":
			building.HousingCompany = stringPtr(value)
		}
	})
}

func parseCoordinates(doc *goquery.Document) (lat *float64, lon *float64) {
	if el := doc.Find("building-map").First(); el != nil {
		if v, err := strconv.ParseFloat(strings.TrimSpace(el.AttrOr("latitude", "")), 64); err == nil {
			lat = &v
		}
		if v, err := strconv.ParseFloat(strings.TrimSpace(el.AttrOr("longitude", "")), 64); err == nil {
			lon = &v
		}
	}
	return lat, lon
}

func parseListings(doc *goquery.Document) ([]BuildingListing, error) {
	table := doc.Find("div[ng-if=\"cardType === '100'\"] table.building-price-table")
	if table.Length() == 0 {
		return nil, nil
	}
	rows := table.Find("tbody tr")
	total := rows.Length()
	listings := make([]BuildingListing, 0, total)
	var parseErr error
	rows.EachWithBreak(func(idx int, row *goquery.Selection) bool {
		cells := row.Find("td")
		if cells.Length() < 6 {
			parseErr = ErrUnexpectedColumns
			return false
		}
		reverseIndex := total - 1 - idx
		entry := BuildingListing{
			Index:         reverseIndex,
			Layout:        optionalString(cells.Eq(1).Text()),
			Size:          parseSize(cells.Eq(2).Text()),
			Price:         parseCurrency(cells.Eq(3).Text()),
			PricePerSqm:   parseCurrency(cells.Eq(4).Text()),
			DeletedAt:     parseDate(cells.Eq(0).Text()),
			MarketingTime: optionalString(cells.Eq(5).Text()),
		}
		listings = append(listings, entry)
		return true
	})
	if parseErr != nil {
		return nil, fmt.Errorf("parse listings: %w", parseErr)
	}
	return listings, nil
}

func parseRentalListings(doc *goquery.Document) ([]RentalListing, error) {
	table := doc.Find("div[ng-if=\"cardType === '101'\"] table.building-price-table")
	if table.Length() == 0 {
		return nil, nil
	}
	rows := table.Find("tbody tr")
	total := rows.Length()
	listings := make([]RentalListing, 0, total)
	var parseErr error
	rows.EachWithBreak(func(idx int, row *goquery.Selection) bool {
		cells := row.Find("td")
		if cells.Length() < 5 {
			parseErr = ErrUnexpectedColumns
			return false
		}
		reverseIndex := total - 1 - idx
		entry := RentalListing{
			Index:         reverseIndex,
			Layout:        optionalString(cells.Eq(1).Text()),
			Size:          parseSize(cells.Eq(2).Text()),
			Price:         parseCurrency(cells.Eq(3).Text()),
			DeletedAt:     parseDate(cells.Eq(0).Text()),
			MarketingTime: optionalString(cells.Eq(4).Text()),
		}
		listings = append(listings, entry)
		return true
	})
	if parseErr != nil {
		return nil, fmt.Errorf("parse rental listings: %w", parseErr)
	}
	return listings, nil
}

func (c *Client) parseSitemap(data []byte, t SitemapType) []SitemapEntry {
	locMatches := sitemapLocRegexp.FindAllStringSubmatch(string(data), -1)
	results := make([]SitemapEntry, 0, len(locMatches))
	for _, match := range locMatches {
		url := strings.TrimSpace(match[1])
		if entry, ok := c.parseSitemapURL(url, t); ok {
			results = append(results, entry)
		}
	}
	return results
}

func (c *Client) parseSitemapURL(url string, t SitemapType) (SitemapEntry, bool) {
	patterns := c.buildSitemapURLPatterns()
	re, ok := patterns[t]
	if !ok {
		return SitemapEntry{}, false
	}
	match := re.FindStringSubmatch(url)
	if len(match) < 2 {
		return SitemapEntry{}, false
	}
	return SitemapEntry{
		URL: url,
		ID:  match[1],
	}, true
}

func parseSize(raw string) *float64 {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil
	}
	return parseNumber(parts[0])
}

func parseCurrency(raw string) *float64 {
	parts := strings.Fields(raw)
	if len(parts) == 0 {
		return nil
	}
	return parseNumber(parts[0])
}

func parseNumber(raw string) *float64 {
	cleaned := numberCleanupRegexp.ReplaceAllString(raw, "")
	if cleaned == "" {
		return nil
	}
	value, err := strconv.ParseFloat(strings.ReplaceAll(cleaned, ",", "."), 64)
	if err != nil {
		return nil
	}
	return &value
}

func parseInt(raw string) (int, bool) {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return 0, false
	}
	return value, true
}

func parseDate(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	t, err := time.Parse("02.01.2006", raw)
	if err != nil {
		return nil
	}
	return &t
}

func stringPtr(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

func optionalString(v string) *string {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return &v
}

var (
	numberCleanupRegexp = regexp.MustCompile(`[^0-9,\.]+`)
	sitemapLocRegexp    = regexp.MustCompile(`(?i)<loc>\s*(.*?)\s*</loc>`)
)

func (c *Client) buildSitemapURLPatterns() map[SitemapType]*regexp.Regexp {
	baseURL := regexp.QuoteMeta(c.baseURL)
	return map[SitemapType]*regexp.Regexp{
		SitemapTypeBuilding: regexp.MustCompile(`^` + baseURL + `/talo/.*?/([0-9]+)`),
		SitemapTypeAd:       regexp.MustCompile(`^` + baseURL + `/myytavat-asunnot/.*?/([0-9]+)`),
	}
}
