package mcpserver

import (
	"log/slog"

	"koditon/internal/db"
	"koditon/internal/domain/ads"
	"koditon/internal/domain/auth"
	"koditon/internal/platform/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpRuntime struct {
	server             *mcp.Server
	impl               *toolImpl
	toolSecurityScopes map[string][]string
}

func newMCPRuntime(pool *pgxpool.Pool, cfg config.Config, logger *slog.Logger) mcpRuntime {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "koditon-mcp",
		Version: "1.0.0",
	}, &mcp.ServerOptions{Capabilities: &mcp.ServerCapabilities{}})
	impl := &toolImpl{
		adsSvc:  ads.NewService(pool),
		queries: db.New(pool),
		config: toolImplConfig{
			webBaseURL:           cfg.WebBaseURL,
			shortcutSitemapBase:  cfg.Shortcut.SitemapBase,
			frontdoorSitemapBase: cfg.Frontdoor.SitemapBase,
		},
		logger: logger.With("component", "mcpserver"),
	}
	runtime := mcpRuntime{server: server, impl: impl, toolSecurityScopes: mcpToolSecurityScopes()}
	runtime.registerFeatures(cfg)
	return runtime
}

func (r mcpRuntime) registerFeatures(cfg config.Config) {
	r.registerTools()
	registerListingsAppResource(r.server, cfg.APIPublicBaseURL, cfg.WebBaseURL)
	registerPropertyResourceTemplates(r.server, r.impl)
	registerPropertyPrompts(r.server)
}

func (r mcpRuntime) registerTools() {
	addTracedTool(r.server, r.impl.findListingsTool(), r.impl.logger, r.impl.findListings)
	addTracedTool(r.server, r.impl.queryPropertiesTool(), r.impl.logger, r.impl.queryProperties)
	addTracedTool(r.server, r.impl.getPropertyDetailTool(), r.impl.logger, r.impl.getPropertyDetail)
	addTracedTool(r.server, r.impl.comparePropertiesTool(), r.impl.logger, r.impl.compareProperties)
	addTracedTool(r.server, r.impl.getPropertyMarketContextTool(), r.impl.logger, r.impl.getPropertyMarketContext)
	addTracedTool(r.server, r.impl.searchListingsTool(), r.impl.logger, r.impl.searchListings)
	addTracedTool(r.server, r.impl.getListingDetailTool(), r.impl.logger, r.impl.getListingDetail)
	addTracedTool(r.server, r.impl.searchTransactionsTool(), r.impl.logger, r.impl.searchTransactions)
	addTracedTool(r.server, r.impl.searchTransactionsAdvancedTool(), r.impl.logger, r.impl.searchTransactionsAdvanced)
	addTracedTool(r.server, r.impl.matchAdsFromTransactionTool(), r.impl.logger, r.impl.matchAdsFromTransaction)
	addTracedTool(r.server, r.impl.listCitiesTool(), r.impl.logger, r.impl.listCities)
	addTracedTool(r.server, r.impl.listAvailableLocationsTool(), r.impl.logger, r.impl.listAvailableLocations)
	addTracedTool(r.server, r.impl.listCategoriesTool(), r.impl.logger, r.impl.listCategories)
}

func mcpToolSecurityScopes() map[string][]string {
	return map[string][]string{
		"koditon_find_listings":                {auth.ScopeMCPCoreRead},
		"koditon_query_properties":             {auth.ScopeMCPCoreRead},
		"koditon_get_property_detail":          {auth.ScopeMCPCoreRead},
		"koditon_compare_properties":           {auth.ScopeMCPCoreRead},
		"koditon_get_property_market_context":  {auth.ScopeMCPCoreRead},
		"koditon_search_listings":              {auth.ScopeMCPCoreRead},
		"koditon_get_listing_detail":           {auth.ScopeMCPCoreRead},
		"koditon_search_transactions":          {auth.ScopeMCPCoreRead},
		"koditon_search_transactions_advanced": {auth.ScopeMCPCoreRead},
		"koditon_match_ads_from_transaction":   {auth.ScopeMCPCoreRead},
		"koditon_list_cities":                  {auth.ScopeMCPCoreRead},
		"koditon_list_available_locations":     {auth.ScopeMCPCoreRead},
		"koditon_list_categories":              {auth.ScopeMCPCoreRead},
	}
}
