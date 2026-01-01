package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/encoding/charmap"
)

const (
	indexURL              = "https://www.posti.fi/webpcode/unzip/"
	defaultRequestTimeout = 2 * time.Minute
)

var (
	filePattern   = regexp.MustCompile(`https://www.posti.fi/webpcode/unzip/PCF_[0-9]*?\.dat`)
	recordPattern = regexp.MustCompile(`^PONOT(.{8})(.{5})(.{30})(.{30})(.{12})(.{12})(.{8})(.{1})(.{5})(.{30})(.{30})(.{3})(.{20})(.{20})(.{1})`)
)

// titleCase converts a string to title case, capitalizing after spaces and hyphens.
// This handles Finnish place names like "SAARI-KÄMÄ" -> "Saari-Kämä".
func titleCase(s string) string {
	var result strings.Builder
	result.Grow(len(s))
	capitalizeNext := true
	for _, r := range s {
		if r == ' ' || r == '-' {
			result.WriteRune(r)
			capitalizeNext = true
		} else if capitalizeNext {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(unicode.ToLower(r))
		}
	}
	return result.String()
}

type PostalCodeRecord struct {
	Date                       string
	Postcode                   string
	PostcodeNameFi             string
	PostcodeNameSv             string
	PostcodeAbbrFi             string
	PostcodeAbbrSv             string
	ValidFrom                  string
	TypeCode                   string
	AdAreaCode                 string
	AdAreaFi                   string
	AdAreaSv                   string
	MunicipalCode              string
	MunicipalNameFi            string
	MunicipalNameSv            string
	MunicipalLanguageRatioCode string
}

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultRequestTimeout,
		},
	}
}

func (c *Client) FetchPostalCodes(ctx context.Context) ([]*PostalCodeRecord, error) {
	dataURL, err := c.findDataFileURL(ctx)
	if err != nil {
		return nil, fmt.Errorf("find data file URL: %w", err)
	}
	records, err := c.fetchAndParseRecords(ctx, dataURL)
	if err != nil {
		return nil, fmt.Errorf("fetch and parse records: %w", err)
	}
	return records, nil
}

func (c *Client) findDataFileURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, indexURL, nil)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	matches := filePattern.FindStringSubmatch(string(body))
	if len(matches) == 0 {
		return "", fmt.Errorf("data file URL not found in index")
	}
	return matches[0], nil
}

func (c *Client) fetchAndParseRecords(ctx context.Context, dataURL string) ([]*PostalCodeRecord, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dataURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("perform request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	decoder := charmap.ISO8859_1.NewDecoder()
	body, err := io.ReadAll(decoder.Reader(resp.Body))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	lines := strings.Split(string(body), "\n")
	records := make([]*PostalCodeRecord, 0, len(lines))
	for _, line := range lines {
		record := parseRecord(line)
		if record != nil {
			records = append(records, record)
		}
	}
	return records, nil
}

func parseRecord(line string) *PostalCodeRecord {
	matches := recordPattern.FindStringSubmatch(line)
	if len(matches) < 16 {
		return nil
	}
	return &PostalCodeRecord{
		Date:                       strings.TrimSpace(matches[1]),
		Postcode:                   strings.TrimSpace(matches[2]),
		PostcodeNameFi:             titleCase(strings.TrimSpace(matches[3])),
		PostcodeNameSv:             titleCase(strings.TrimSpace(matches[4])),
		PostcodeAbbrFi:             titleCase(strings.TrimSpace(matches[5])),
		PostcodeAbbrSv:             titleCase(strings.TrimSpace(matches[6])),
		ValidFrom:                  strings.TrimSpace(matches[7]),
		TypeCode:                   strings.TrimSpace(matches[8]),
		AdAreaCode:                 strings.TrimSpace(matches[9]),
		AdAreaFi:                   titleCase(strings.TrimSpace(matches[10])),
		AdAreaSv:                   titleCase(strings.TrimSpace(matches[11])),
		MunicipalCode:              strings.TrimSpace(matches[12]),
		MunicipalNameFi:            titleCase(strings.TrimSpace(matches[13])),
		MunicipalNameSv:            titleCase(strings.TrimSpace(matches[14])),
		MunicipalLanguageRatioCode: strings.TrimSpace(matches[15]),
	}
}
