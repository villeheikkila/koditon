package hintatiedot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

var (
	ErrUnexpectedStatus = errors.New("unexpected HTTP status")
	ErrInvalidResponse  = errors.New("invalid response format")
	ErrParsingError     = errors.New("parsing error")
)

type BuildingType string

const (
	BuildingTypeApartment BuildingType = "1"
	BuildingTypeRowHouse  BuildingType = "2"
	BuildingTypeHouse     BuildingType = "3"
)

type RoomCount string

const (
	RoomCountOne      RoomCount = "1"
	RoomCountTwo      RoomCount = "2"
	RoomCountThree    RoomCount = "3"
	RoomCountFourPlus RoomCount = "4"
)

type ApartmentSearchParams struct {
	City          string
	PostalCodes   []string
	BuildingTypes []BuildingType
	RoomCounts    []RoomCount
	MinArea       *float64
	MaxArea       *float64
	RenderType    string
	Print         bool
}

func NewApartmentSearchParams(city string) *ApartmentSearchParams {
	return &ApartmentSearchParams{
		City:       city,
		RenderType: "renderTypeTable",
		Print:      false,
	}
}

func (p *ApartmentSearchParams) ToURLValues(page int) url.Values {
	values := url.Values{}
	values.Add("c", p.City)
	values.Add("cr", "1")
	for _, code := range p.PostalCodes {
		values.Add("ps", code)
	}
	values.Add("nc", "1")
	for _, buildingType := range p.BuildingTypes {
		values.Add("h", string(buildingType))
	}
	for _, roomCount := range p.RoomCounts {
		values.Add("r", string(roomCount))
	}
	if p.MinArea != nil {
		values.Add("amin", fmt.Sprintf("%f", *p.MinArea))
	} else {
		values.Add("amin", "")
	}
	if p.MaxArea != nil {
		values.Add("amax", fmt.Sprintf("%f", *p.MaxArea))
	} else {
		values.Add("amax", "")
	}
	values.Add("renderType", "renderTypeTable")
	if p.Print {
		values.Add("print", "1")
	} else {
		values.Add("print", "0")
	}
	if page > 0 {
		values.Add("z", strconv.Itoa(page))
	}
	values.Add("search", "1")
	if page > 0 {
		values.Add("submit", "seuraava+sivu+»")
	}
	return values
}

type TransactionEntity struct {
	City                string
	Neighborhood        string
	Description         string
	Type                string
	Area                float64
	Price               int
	PricePerSquareMeter int
	BuildYear           int
	Floor               string
	Elevator            string
	Condition           string
	Plot                string
	EnergyClass         string
	Category            string
}

type TransactionResponse struct {
	Apartments []*TransactionEntity
	NextPage   *int
}

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	logger     *slog.Logger
}

func NewClient(httpClient *http.Client, baseURL string, logger *slog.Logger) (*Client, error) {
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    parsedBaseURL,
		logger:     logger,
	}, nil
}

func (c *Client) setCommonHeaders(req *http.Request) {
	headers := map[string]string{
		"Accept":           "*/*",
		"Accept-Language":  "en-US,en;q=0.9",
		"Cache-Control":    "no-cache",
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"Pragma":           "no-cache",
		"Priority":         "u=5, i",
		"User-Agent":       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/26.1 Safari/605.1.15",
		"X-Requested-With": "XMLHttpRequest",
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
}

func (c *Client) doRequest(ctx context.Context, req *http.Request, target any) error {
	c.setCommonHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("%w: %d: %s", ErrUnexpectedStatus, resp.StatusCode, bytes.TrimSpace(body))
	}
	reader := c.getBodyReader(resp)
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) getBodyReader(resp *http.Response) io.Reader {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "charset=iso-8859-1") {
		return charmap.ISO8859_1.NewDecoder().Reader(resp.Body)
	}
	return resp.Body
}

func parseList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	if trimmed == "" {
		return []string{}
	}

	parts := strings.Split(trimmed, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}
