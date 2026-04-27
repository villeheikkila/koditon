package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	frontdoorclient "koditon/internal/clients/frontdoor"
	shortcutclient "koditon/internal/clients/shortcut"
)

type providerConfig struct {
	ShortcutBaseURL      string
	ShortcutDocsBaseURL  string
	ShortcutAdBaseURL    string
	ShortcutUserAgent    string
	ShortcutSitemapBase  string
	FrontdoorBaseURL     string
	FrontdoorUserAgent   string
	FrontdoorCookie      string
	FrontdoorSitemapBase string
}

type apiQueryOptions struct {
	Compact bool
	Timeout time.Duration
}

func runAPIQuery(ctx context.Context, args []string, stdout io.Writer, getenv func(string) string) error {
	_ = godotenv.Load(".env.local", ".env", "backend/.env.local", "backend/.env")
	if len(args) < 1 {
		printAPIQueryUsage()
		return fmt.Errorf("usage: cli api-query <frontdoor|shortcut> <query> [flags]")
	}
	cfg := providerConfigFromEnv(getenv)
	source := args[0]
	switch source {
	case "frontdoor":
		return runFrontdoorAPIQuery(ctx, args[1:], stdout, cfg)
	case "shortcut":
		return runShortcutAPIQuery(ctx, args[1:], stdout, cfg)
	case "help", "-h", "--help":
		printAPIQueryUsage()
		return nil
	default:
		printAPIQueryUsage()
		return fmt.Errorf("unknown api-query source: %s", source)
	}
}

func providerConfigFromEnv(getenv func(string) string) providerConfig {
	return providerConfig{
		ShortcutBaseURL:      strings.TrimSpace(getenv("SHORTCUT_BASE_URL")),
		ShortcutDocsBaseURL:  strings.TrimSpace(getenv("SHORTCUT_DOCS_BASE_URL")),
		ShortcutAdBaseURL:    strings.TrimSpace(getenv("SHORTCUT_AD_BASE_URL")),
		ShortcutUserAgent:    strings.TrimSpace(getenv("SHORTCUT_USER_AGENT")),
		ShortcutSitemapBase:  strings.TrimSpace(getenv("SHORTCUT_SITEMAP_BASE_URL")),
		FrontdoorBaseURL:     strings.TrimSpace(getenv("FRONTDOOR_BASE_URL")),
		FrontdoorUserAgent:   strings.TrimSpace(getenv("FRONTDOOR_USER_AGENT")),
		FrontdoorCookie:      strings.TrimSpace(getenv("FRONTDOOR_COOKIE")),
		FrontdoorSitemapBase: strings.TrimSpace(getenv("FRONTDOOR_SITEMAP_BASE_URL")),
	}
}

func printAPIQueryUsage() {
	output := flag.CommandLine.Output()
	_, _ = fmt.Fprintln(output, "Usage: cli api-query <frontdoor|shortcut> <query> [flags]")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Frontdoor queries:")
	_, _ = fmt.Fprintln(output, "  ad --friendly-id <id>")
	_, _ = fmt.Fprintln(output, "  building-page --url <url>")
	_, _ = fmt.Fprintln(output, "  sitemap")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Shortcut queries:")
	_, _ = fmt.Fprintln(output, "  ad --id <numeric-id>")
	_, _ = fmt.Fprintln(output, "  locations --postal <postal-code-or-query>")
	_, _ = fmt.Fprintln(output, "  buildings --query <text>")
	_, _ = fmt.Fprintln(output, "  building-data --location-id <id>")
	_, _ = fmt.Fprintln(output, "  search-apartments --location-card-id <id> --location-card-type <type> --location-name <name> [--card-type sale|rent] [--page N] [--page-size N]")
	_, _ = fmt.Fprintln(output, "  sitemap")
	_, _ = fmt.Fprintln(output)
	_, _ = fmt.Fprintln(output, "Common flags: --compact --timeout 30s")
}

func runFrontdoorAPIQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cli api-query frontdoor <ad|building-page|sitemap> [flags]")
	}
	query := args[0]
	switch query {
	case "ad":
		return runFrontdoorAdQuery(ctx, args[1:], stdout, cfg)
	case "building-page":
		return runFrontdoorBuildingPageQuery(ctx, args[1:], stdout, cfg)
	case "sitemap":
		return runFrontdoorSitemapQuery(ctx, args[1:], stdout, cfg)
	default:
		return fmt.Errorf("unknown frontdoor api query: %s", query)
	}
}

func runFrontdoorAdQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query frontdoor ad")
	friendlyID := fs.String("friendly-id", "", "Frontdoor announcement friendly ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*friendlyID) == "" {
		return fmt.Errorf("--friendly-id is required")
	}
	if err := requireProviderValues(map[string]string{"FRONTDOOR_BASE_URL": cfg.FrontdoorBaseURL, "FRONTDOOR_USER_AGENT": cfg.FrontdoorUserAgent}); err != nil {
		return err
	}
	client := frontdoorclient.New(cfg.FrontdoorBaseURL, cfg.FrontdoorUserAgent, cfg.FrontdoorCookie, cfg.FrontdoorSitemapBase)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	result, err := client.GetAdByFriendlyID(reqCtx, *friendlyID)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result, opts.Compact)
}

func runFrontdoorBuildingPageQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query frontdoor building-page")
	pageURL := fs.String("url", "", "Frontdoor building page URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*pageURL) == "" {
		return fmt.Errorf("--url is required")
	}
	if err := requireProviderValues(map[string]string{"FRONTDOOR_USER_AGENT": cfg.FrontdoorUserAgent}); err != nil {
		return err
	}
	client := frontdoorclient.New(cfg.FrontdoorBaseURL, cfg.FrontdoorUserAgent, cfg.FrontdoorCookie, cfg.FrontdoorSitemapBase)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	result, err := client.GetBuildingPageData(reqCtx, *pageURL)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result, opts.Compact)
}

func runFrontdoorSitemapQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query frontdoor sitemap")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireProviderValues(map[string]string{"FRONTDOOR_SITEMAP_BASE_URL": cfg.FrontdoorSitemapBase, "FRONTDOOR_USER_AGENT": cfg.FrontdoorUserAgent}); err != nil {
		return err
	}
	client := frontdoorclient.New(cfg.FrontdoorBaseURL, cfg.FrontdoorUserAgent, cfg.FrontdoorCookie, cfg.FrontdoorSitemapBase)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	entries, err := client.GetSitemapEntries(reqCtx)
	if err != nil {
		return err
	}
	return writeJSON(stdout, frontdoorSitemapOutput(entries), opts.Compact)
}

func runShortcutAPIQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: cli api-query shortcut <ad|locations|buildings|building-data|search-apartments|sitemap> [flags]")
	}
	query := args[0]
	switch query {
	case "ad":
		return runShortcutAdQuery(ctx, args[1:], stdout, cfg)
	case "locations":
		return runShortcutLocationsQuery(ctx, args[1:], stdout, cfg)
	case "buildings":
		return runShortcutBuildingsQuery(ctx, args[1:], stdout, cfg)
	case "building-data":
		return runShortcutBuildingDataQuery(ctx, args[1:], stdout, cfg)
	case "search-apartments":
		return runShortcutSearchApartmentsQuery(ctx, args[1:], stdout, cfg)
	case "sitemap":
		return runShortcutSitemapQuery(ctx, args[1:], stdout, cfg)
	default:
		return fmt.Errorf("unknown shortcut api query: %s", query)
	}
}

func runShortcutAdQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query shortcut ad")
	id := fs.Int("id", 0, "Shortcut ad numeric ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id <= 0 {
		return fmt.Errorf("--id must be positive")
	}
	if err := requireShortcutValues(cfg, "SHORTCUT_BASE_URL", "SHORTCUT_AD_BASE_URL", "SHORTCUT_USER_AGENT"); err != nil {
		return err
	}
	client := newShortcutAPIClient(cfg)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	raw, err := client.GetAdByID(reqCtx, *id)
	if err != nil {
		return err
	}
	return writeJSON(stdout, raw, opts.Compact)
}

func runShortcutLocationsQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query shortcut locations")
	postal := fs.String("postal", "", "Postal code or location query")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*postal) == "" {
		return fmt.Errorf("--postal is required")
	}
	if err := requireShortcutValues(cfg, "SHORTCUT_BASE_URL", "SHORTCUT_USER_AGENT"); err != nil {
		return err
	}
	client := newShortcutAPIClient(cfg)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	result, err := client.FetchLocationIDs(reqCtx, *postal)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result, opts.Compact)
}

func runShortcutBuildingsQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query shortcut buildings")
	query := fs.String("query", "", "Shortcut building search query")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*query) == "" {
		return fmt.Errorf("--query is required")
	}
	if err := requireShortcutValues(cfg, "SHORTCUT_BASE_URL", "SHORTCUT_USER_AGENT"); err != nil {
		return err
	}
	client := newShortcutAPIClient(cfg)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	result, err := client.FetchBuildings(reqCtx, *query)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result, opts.Compact)
}

func runShortcutBuildingDataQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query shortcut building-data")
	locationID := fs.String("location-id", "", "Shortcut location ID")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(*locationID) == "" {
		return fmt.Errorf("--location-id is required")
	}
	if err := requireShortcutValues(cfg, "SHORTCUT_BASE_URL", "SHORTCUT_USER_AGENT"); err != nil {
		return err
	}
	client := newShortcutAPIClient(cfg)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	result, err := client.GetBuildingData(reqCtx, *locationID)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result, opts.Compact)
}

func runShortcutSearchApartmentsQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query shortcut search-apartments")
	locationCardID := fs.Int("location-card-id", 0, "Shortcut location card ID")
	locationCardType := fs.Int("location-card-type", 0, "Shortcut location card type")
	locationName := fs.String("location-name", "", "Shortcut location name")
	cardType := fs.String("card-type", "sale", "Apartment card type: sale or rent")
	page := fs.Int("page", 0, "Zero-based page number")
	pageSize := fs.Int("page-size", 30, "Page size")
	if err := fs.Parse(args); err != nil {
		return err
	}
	params, err := shortcutSearchParams(*locationCardID, *locationCardType, *locationName, *cardType, *page, *pageSize)
	if err != nil {
		return err
	}
	if err := requireShortcutValues(cfg, "SHORTCUT_BASE_URL", "SHORTCUT_USER_AGENT"); err != nil {
		return err
	}
	client := newShortcutAPIClient(cfg)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	result, err := client.SearchApartments(reqCtx, params)
	if err != nil {
		return err
	}
	return writeJSON(stdout, result, opts.Compact)
}

func runShortcutSitemapQuery(ctx context.Context, args []string, stdout io.Writer, cfg providerConfig) error {
	fs, opts := newAPIFlagSet("api-query shortcut sitemap")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := requireShortcutValues(cfg, "SHORTCUT_SITEMAP_BASE_URL", "SHORTCUT_USER_AGENT"); err != nil {
		return err
	}
	client := newShortcutAPIClient(cfg)
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()
	entries, err := client.GetSitemapEntries(reqCtx)
	if err != nil {
		return err
	}
	return writeJSON(stdout, shortcutSitemapOutput(entries), opts.Compact)
}

func newAPIFlagSet(name string) (*flag.FlagSet, *apiQueryOptions) {
	opts := &apiQueryOptions{Timeout: 30 * time.Second}
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.BoolVar(&opts.Compact, "compact", false, "Print compact JSON")
	fs.DurationVar(&opts.Timeout, "timeout", opts.Timeout, "Request timeout")
	return fs, opts
}

func requireShortcutValues(cfg providerConfig, names ...string) error {
	values := map[string]string{
		"SHORTCUT_BASE_URL":         cfg.ShortcutBaseURL,
		"SHORTCUT_DOCS_BASE_URL":    cfg.ShortcutDocsBaseURL,
		"SHORTCUT_AD_BASE_URL":      cfg.ShortcutAdBaseURL,
		"SHORTCUT_USER_AGENT":       cfg.ShortcutUserAgent,
		"SHORTCUT_SITEMAP_BASE_URL": cfg.ShortcutSitemapBase,
	}
	required := make(map[string]string, len(names))
	for _, name := range names {
		required[name] = values[name]
	}
	return requireProviderValues(required)
}

func requireProviderValues(values map[string]string) error {
	var missing []string
	for name, value := range values {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required provider env vars: %s", strings.Join(missing, ", "))
	}
	for name, value := range values {
		if strings.HasSuffix(name, "_URL") || strings.HasSuffix(name, "_BASE_URL") {
			if err := validateHTTPURL(name, value); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", name)
	}
	return nil
}

func newShortcutAPIClient(cfg providerConfig) *shortcutclient.Client {
	return shortcutclient.NewClient(
		slog.New(slog.DiscardHandler),
		nil,
		nil,
		cfg.ShortcutBaseURL,
		cfg.ShortcutDocsBaseURL,
		cfg.ShortcutAdBaseURL,
		cfg.ShortcutUserAgent,
		cfg.ShortcutSitemapBase,
	)
}

func shortcutSearchParams(locationCardID, locationCardType int, locationName, cardType string, page, pageSize int) (shortcutclient.SearchParams, error) {
	if locationCardID <= 0 {
		return shortcutclient.SearchParams{}, fmt.Errorf("--location-card-id must be positive")
	}
	if locationCardType <= 0 {
		return shortcutclient.SearchParams{}, fmt.Errorf("--location-card-type must be positive")
	}
	if strings.TrimSpace(locationName) == "" {
		return shortcutclient.SearchParams{}, fmt.Errorf("--location-name is required")
	}
	if page < 0 {
		return shortcutclient.SearchParams{}, fmt.Errorf("--page must be non-negative")
	}
	if pageSize <= 0 {
		return shortcutclient.SearchParams{}, fmt.Errorf("--page-size must be positive")
	}
	var parsedCardType shortcutclient.CardType
	switch strings.ToLower(strings.TrimSpace(cardType)) {
	case "sale":
		parsedCardType = shortcutclient.CardTypeSale
	case "rent":
		parsedCardType = shortcutclient.CardTypeRent
	default:
		return shortcutclient.SearchParams{}, fmt.Errorf("--card-type must be sale or rent")
	}
	location := shortcutclient.LocationResponse{}
	location.Card.CardID = locationCardID
	location.Card.CardType = locationCardType
	location.Card.Name = locationName
	return shortcutclient.SearchParams{
		Location: location,
		CardType: parsedCardType,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func writeJSON(w io.Writer, value any, compact bool) error {
	var data []byte
	var err error
	if raw, ok := value.(json.RawMessage); ok {
		if compact {
			data = raw
		} else {
			data, err = json.MarshalIndent(raw, "", "  ")
		}
	} else if compact {
		data, err = json.Marshal(value)
	} else {
		data, err = json.MarshalIndent(value, "", "  ")
	}
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

type sitemapEntryOutput struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	URL  string `json:"url"`
}

func frontdoorSitemapOutput(entries []frontdoorclient.SitemapEntry) []sitemapEntryOutput {
	out := make([]sitemapEntryOutput, 0, len(entries))
	for _, entry := range entries {
		out = append(out, sitemapEntryOutput{
			ID:   entry.ID,
			Type: string(entry.Type),
			URL:  entry.URL.String(),
		})
	}
	return out
}

func shortcutSitemapOutput(entries []shortcutclient.ShortcutSitemapEntry) []sitemapEntryOutput {
	out := make([]sitemapEntryOutput, 0, len(entries))
	for _, entry := range entries {
		out = append(out, sitemapEntryOutput{
			ID:   strconv.Itoa(entry.ID),
			Type: string(entry.Type),
			URL:  entry.URL.String(),
		})
	}
	return out
}
