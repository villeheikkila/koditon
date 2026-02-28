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
	adsSvc     *ads.Service
	queries    *db.Queries
	cfg        config.Config
	logger     *slog.Logger
	sseServer  *server.SSEServer
}

func New(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) *Server {
	s := &Server{
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
	s.sseServer = server.NewSSEServer(mcpSrv, server.WithStaticBasePath("/mcp"))
	return s
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.sseServer
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
			mcp.WithDescription("Get full details for a listing by canonical ID (e.g. shortcut:ad:12345) or URL"),
			mcp.WithString("id", mcp.Description("Canonical ID or listing URL"), mcp.Required()),
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
