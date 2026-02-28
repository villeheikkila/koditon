package mcpserver

import (
	"log/slog"
	"net/http"
	"strings"

	"koditon-go/internal/ads"
	"koditon-go/internal/config"
	"koditon-go/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Server struct {
	mcpServer  *server.MCPServer
	pool       *pgxpool.Pool
	adsSvc     *ads.Service
	queries    *db.Queries
	cfg        config.Config
	logger     *slog.Logger
	httpServer *server.StreamableHTTPServer
}

func New(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) *Server {
	s := &Server{
		pool:    pool,
		adsSvc:  ads.NewService(pool),
		queries: db.New(pool),
		cfg:     cfg,
		logger:  logger.With("component", "mcp"),
	}

	mcpSrv := server.NewMCPServer(
		"koditon",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	s.mcpServer = mcpSrv
	s.registerTools()
	s.httpServer = server.NewStreamableHTTPServer(mcpSrv)
	return s
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.httpServer
	if s.cfg.MCPAuthToken != "" {
		handler = authMiddleware(s.cfg.MCPAuthToken, handler)
	}
	return handler
}

func authMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if bearer != token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) registerTools() {
	s.mcpServer.AddTool(
		mcp.NewTool("search_listings",
			mcp.WithDescription("Search property listings (ads, buildings, announcements) with filters"),
			mcp.WithString("query", mcp.Description("Free-text search query")),
			mcp.WithString("address", mcp.Description("Address search alias for query (partial or exact)")),
			mcp.WithString("source", mcp.Description("Source filter: shortcut, frontdoor, or all")),
			mcp.WithString("kind", mcp.Description("Entity kind: ad, building, announcement, or all")),
			mcp.WithString("listing_type", mcp.Description("Listing type: listing, rental, or all")),
			mcp.WithString("city", mcp.Description("City name filter")),
			mcp.WithString("postal", mcp.Description("Postal code filter")),
			mcp.WithNumber("min_price", mcp.Description("Minimum price in euros")),
			mcp.WithNumber("max_price", mcp.Description("Maximum price in euros")),
			mcp.WithNumber("min_area", mcp.Description("Minimum area in square meters")),
			mcp.WithNumber("max_area", mcp.Description("Maximum area in square meters")),
			mcp.WithString("sort", mcp.Description("Sort mode: price_asc, price_desc, area_asc, area_desc, seen_desc")),
			mcp.WithNumber("page", mcp.Description("Page number (default 1)")),
			mcp.WithNumber("page_size", mcp.Description("Results per page: 25, 50, or 100 (default 50)")),
		),
		s.handleSearchListings,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("get_listing_detail",
			mcp.WithDescription("Get full listing detail by canonical ID, URL, or address text. Returns normalized fields for valuation and optional raw_json payload"),
			mcp.WithString("id", mcp.Description("Canonical ID or listing URL (backward-compatible alias of input)")),
			mcp.WithString("input", mcp.Description("Canonical ID, listing URL, or address text (partial or exact)")),
			mcp.WithString("source", mcp.Description("Optional source filter when resolving address text: shortcut, frontdoor, or all")),
			mcp.WithString("kind", mcp.Description("Optional kind filter when resolving address text: ad, building, announcement, or all (default ad for text input)")),
			mcp.WithString("city", mcp.Description("Optional city filter when resolving address text")),
			mcp.WithString("postal", mcp.Description("Optional postal code filter when resolving address text")),
			mcp.WithNumber("max_candidates", mcp.Description("Maximum candidates to scan for text resolution (default 25, allowed: 25, 50, 100)")),
			mcp.WithBoolean("include_raw_json", mcp.Description("Include raw_json payload (default false)")),
		),
		s.handleGetListingDetail,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("search_transactions",
			mcp.WithDescription("Search property price transactions by city and optional address"),
			mcp.WithString("city", mcp.Description("City name (required)"), mcp.Required()),
			mcp.WithString("address", mcp.Description("Optional address search term")),
			mcp.WithNumber("limit", mcp.Description("Maximum results (default 200)")),
		),
		s.handleSearchTransactions,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("search_transactions_advanced",
			mcp.WithDescription("Advanced prices transaction search with exact filters and flexible free-text"),
			mcp.WithString("city", mcp.Description("Optional city filter (partial match)")),
			mcp.WithString("query", mcp.Description("Optional free-text filter over address/neighborhood/postal/category/type")),
			mcp.WithArray("municipality_ids", mcp.Description("Optional municipality UUID filters"), mcp.WithStringItems()),
			mcp.WithArray("postal_code_ids", mcp.Description("Optional postal code UUID filters"), mcp.WithStringItems()),
			mcp.WithArray("postal_codes", mcp.Description("Optional postal code text filters"), mcp.WithStringItems()),
			mcp.WithArray("categories", mcp.Description("Optional transaction category filters"), mcp.WithStringItems()),
			mcp.WithArray("types", mcp.Description("Optional transaction type filters"), mcp.WithStringItems()),
			mcp.WithNumber("min_price", mcp.Description("Minimum toteutunut hinta in euros")),
			mcp.WithNumber("max_price", mcp.Description("Maximum toteutunut hinta in euros")),
			mcp.WithNumber("min_area", mcp.Description("Minimum area in square meters")),
			mcp.WithNumber("max_area", mcp.Description("Maximum area in square meters")),
			mcp.WithString("sort", mcp.Description("Sort mode: date_desc, date_asc, price_desc, price_asc, area_desc, area_asc")),
			mcp.WithNumber("limit", mcp.Description("Maximum rows to return (default 200, max 5000)")),
		),
		s.handleSearchTransactionsAdvanced,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("match_ads_from_transaction",
			mcp.WithDescription("Find matching shortcut/frontdoor ads/buildings for a prices transaction using postal code, area and flexible room-layout hints"),
			mcp.WithString("transaction_id", mcp.Description("Optional prices_transaction_id UUID; when set, transaction context is loaded automatically")),
			mcp.WithString("city", mcp.Description("Optional city override/filter")),
			mcp.WithString("postal_code", mcp.Description("Optional postal code override/filter")),
			mcp.WithNumber("area", mcp.Description("Optional target area in square meters")),
			mcp.WithNumber("price", mcp.Description("Optional target toteutunut hinta in euros")),
			mcp.WithString("room_hint", mcp.Description("Optional room layout hint, supports partial/truncated values")),
			mcp.WithString("query", mcp.Description("Optional extra text query for listing search")),
			mcp.WithString("source", mcp.Description("Source filter: shortcut, frontdoor, all")),
			mcp.WithString("kind", mcp.Description("Kind filter: ad, building, announcement, all")),
			mcp.WithString("listing_type", mcp.Description("Listing type filter: listing, rental, all")),
			mcp.WithNumber("area_tolerance", mcp.Description("Area tolerance in square meters around target area (default 8.0)")),
			mcp.WithNumber("price_tolerance_pct", mcp.Description("Price tolerance percentage around target price (default 0.35 = 35%)")),
			mcp.WithNumber("max_candidates", mcp.Description("Ads search candidate window: 25, 50, 100 (default 100)")),
			mcp.WithNumber("max_results", mcp.Description("Final ranked matches to return (default 20, max 100)")),
		),
		s.handleMatchAdsFromTransaction,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("list_cities",
			mcp.WithDescription("List all municipalities with their postal codes"),
		),
		s.handleListCities,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("list_available_locations",
			mcp.WithDescription("List municipalities that have price transaction data"),
		),
		s.handleListAvailableLocations,
	)

	s.mcpServer.AddTool(
		mcp.NewTool("list_categories",
			mcp.WithDescription("List distinct building categories (e.g. Kerrostalo, Rivitalo, etc.)"),
		),
		s.handleListCategories,
	)
}
