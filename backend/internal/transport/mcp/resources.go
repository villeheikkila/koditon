package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"koditon/internal/domain/ads"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	propertyResourceTemplate       = "koditon://property/{id}"
	propertyReportResourceTemplate = "koditon://property/{id}/report"
	propertyMarketResourceTemplate = "koditon://property/{id}/market"
	propertySearchResourceTemplate = "koditon://search/properties{?query,city,postal,min_price,max_price,min_area,max_area}"
	propertyComparisonTemplate     = "koditon://comparison/{id}"
)

func registerPropertyResourceTemplates(server *mcp.Server, impl *toolImpl) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "koditon-property", Title: "Koditon Property", Description: "Canonical property detail by Koditon canonical ID. Example: koditon://property/frontdoor%3Aad%3A123", MIMEType: "application/json", URITemplate: propertyResourceTemplate}, impl.readPropertyResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "koditon-property-report", Title: "Koditon Property Report", Description: "Markdown due-diligence report for a canonical property.", MIMEType: "text/markdown", URITemplate: propertyReportResourceTemplate}, impl.readPropertyReportResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "koditon-property-market", Title: "Koditon Property Market Context", Description: "Market-context resource template for property comparables. Backed by koditon_get_property_market_context.", MIMEType: "application/json", URITemplate: propertyMarketResourceTemplate}, impl.readPropertyMarketResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "koditon-property-search", Title: "Koditon Property Search", Description: "Template URI for stable property search result resources.", MIMEType: "application/json", URITemplate: propertySearchResourceTemplate}, readPlannedResource)
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "koditon-property-comparison", Title: "Koditon Property Comparison", Description: "Template URI for saved comparison resources.", MIMEType: "application/json", URITemplate: propertyComparisonTemplate}, readPlannedResource)
}

func (t *toolImpl) readPropertyResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id, err := propertyIDFromResourceURI(req.Params.URI, "")
	if err != nil {
		return nil, err
	}
	detail, err := t.adsSvc.DetailByCanonicalID(ctx, id)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return nil, fmt.Errorf("read property resource: %w", err)
	}
	result := t.buildPropertyDetailResult(detail, false, false)
	return jsonResource(req.Params.URI, result)
}

func (t *toolImpl) readPropertyReportResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id, err := propertyIDFromResourceURI(req.Params.URI, "/report")
	if err != nil {
		return nil, err
	}
	detail, err := t.adsSvc.DetailByCanonicalID(ctx, id)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return nil, fmt.Errorf("read property report resource: %w", err)
	}
	result := t.buildPropertyDetailResult(detail, false, false)
	return textResource(req.Params.URI, "text/markdown", result.Markdown), nil
}

func (t *toolImpl) readPropertyMarketResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	id, err := propertyIDFromResourceURI(req.Params.URI, "/market")
	if err != nil {
		return nil, err
	}
	_, structured, err := t.getPropertyMarketContext(ctx, nil, PropertyMarketContextInput{ID: id})
	if err != nil {
		return nil, fmt.Errorf("read property market resource: %w", err)
	}
	return jsonResource(req.Params.URI, structured)
}

func readPlannedResource(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	return textResource(req.Params.URI, "application/json", `{"status":"planned","message":"This MCP resource template is reserved for stable saved artifacts."}`), nil
}

func propertyIDFromResourceURI(rawURI, suffix string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURI))
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "koditon" || parsed.Host != "property" {
		return "", mcp.ResourceNotFoundError(rawURI)
	}
	escapedPath := parsed.EscapedPath()
	if suffix != "" {
		if !strings.HasSuffix(parsed.Path, suffix) {
			return "", mcp.ResourceNotFoundError(rawURI)
		}
		escapedPath = strings.TrimSuffix(escapedPath, suffix)
	}
	escapedID := strings.Trim(strings.TrimPrefix(escapedPath, "/"), "/")
	id, err := url.PathUnescape(escapedID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(id) == "" {
		return "", mcp.ResourceNotFoundError(rawURI)
	}
	return id, nil
}

func jsonResource(uri string, value any) (*mcp.ReadResourceResult, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal resource: %w", err)
	}
	return textResource(uri, "application/json", string(data)), nil
}

func textResource(uri, mimeType, text string) *mcp.ReadResourceResult {
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: mimeType, Text: text}}}
}
