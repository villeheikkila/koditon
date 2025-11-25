package hintatiedot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func (c *Client) GetTransactionsForPage(ctx context.Context, params *ApartmentSearchParams, page int) (*TransactionResponse, error) {
	endpoint, err := c.baseURL.Parse("/haku/")
	if err != nil {
		return nil, fmt.Errorf("build transactions URL: %w", err)
	}
	endpoint.RawQuery = params.ToURLValues(page).Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	headers := map[string]string{
		"User-Agent":                "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":           "en-US,en;q=0.5",
		"Accept-Encoding":           "gzip, deflate, br",
		"Connection":                "keep-alive",
		"Upgrade-Insecure-Requests": "1",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("%w: %d: %s", ErrUnexpectedStatus, resp.StatusCode, string(body))
	}
	html, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return c.parseResponse(string(html), params.City)
}

func (c *Client) parseResponse(html, city string) (*TransactionResponse, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	apartments, err := c.parseTransactions(doc, city)
	if err != nil {
		return nil, fmt.Errorf("parse transactions: %w", err)
	}
	nextPage := c.parseNextPage(doc)
	return &TransactionResponse{
		Apartments: apartments,
		NextPage:   nextPage,
	}, nil
}

func (c *Client) parseNextPage(doc *goquery.Document) *int {
	val, exists := doc.Find("table.pagination td.more[align=right] form input[name=z]").First().Attr("value")
	if !exists || val == "" {
		return nil
	}
	page, err := strconv.Atoi(val)
	if err != nil {
		return nil
	}
	return &page
}

func (c *Client) parseTransactions(doc *goquery.Document, city string) ([]*TransactionEntity, error) {
	var apartments []*TransactionEntity
	currentCategory := ""
	doc.Find("tr").Each(func(_ int, row *goquery.Selection) {
		cols := row.Find("td")
		if cols.Length() > 0 && cols.Eq(0).HasClass("section") {
			strong := cols.Eq(0).Find("strong").First()
			if strong.Length() > 0 {
				currentCategory = strings.TrimSpace(strong.Text())
			}
			return
		}
		if cols.Length() < 12 {
			return
		}
		firstCol := strings.TrimSpace(cols.Eq(0).Text())
		if firstCol == "" || cols.Eq(0).HasClass("fullWidth") {
			return
		}
		apartment := &TransactionEntity{
			City:                city,
			Neighborhood:        strings.TrimSpace(cols.Eq(0).Text()),
			Description:         strings.TrimSpace(cols.Eq(1).Text()),
			Type:                strings.TrimSpace(cols.Eq(2).Text()),
			Area:                parseFloat(cols.Eq(3).Text()),
			Price:               parseInt(cols.Eq(4).Text()),
			PricePerSquareMeter: parseInt(cols.Eq(5).Text()),
			BuildYear:           parseInt(cols.Eq(6).Text()),
			Floor:               strings.TrimSpace(cols.Eq(7).Text()),
			Elevator:            strings.TrimSpace(cols.Eq(8).Text()),
			Condition:           strings.TrimSpace(cols.Eq(9).Text()),
			Plot:                strings.TrimSpace(cols.Eq(10).Text()),
			EnergyClass:         strings.TrimSpace(cols.Eq(11).Text()),
			Category:            currentCategory,
		}
		apartments = append(apartments, apartment)
	})
	return apartments, nil
}

func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ".")
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseInt(s string) int {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	val, _ := strconv.Atoi(s)
	return val
}

func (c *Client) GetAllTransactions(ctx context.Context, city string) ([]*TransactionEntity, error) {
	var allApartments []*TransactionEntity
	nextPage := new(int)
	*nextPage = 0
	for nextPage != nil {
		page := *nextPage
		if page > 0 {
			time.Sleep(1 * time.Second)
		}
		startTime := time.Now()
		c.logger.Info("starting to parse hintatiedot page",
			"city", city,
			"page", page,
		)
		response, err := c.GetTransactionsForPage(ctx, NewApartmentSearchParams(city), page)
		if err != nil {
			c.logger.Error("page fetch failed",
				"city", city,
				"page", page,
				"error", err,
			)
			return nil, fmt.Errorf("fetch page %d: %w", page, err)
		}
		elapsed := time.Since(startTime)
		c.logger.Info("finished parsing page",
			"city", city,
			"page", page,
			"duration", fmt.Sprintf("%.2f seconds", elapsed.Seconds()),
			"apartmentsFound", len(response.Apartments),
		)
		allApartments = append(allApartments, response.Apartments...)
		nextPage = response.NextPage
	}
	return allApartments, nil
}
