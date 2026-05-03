package api

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"koditon/internal/domain/ads"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
)

type propertySearchInput struct {
	Query         string  `query:"q"                doc:"Free text search"`
	Source        string  `query:"source"           doc:"Source filter: shortcut, frontdoor, or all"`
	Kind          string  `query:"kind"             doc:"Listing kind filter: ad, announcement, or all"`
	City          string  `query:"city"             doc:"City / municipality filter"`
	Postal        string  `query:"postal"           doc:"Postal code prefix filter"`
	MinPrice      int64   `query:"min_price"        doc:"Minimum price (EUR, 0 = no minimum)"`
	MaxPrice      int64   `query:"max_price"        doc:"Maximum price (EUR, 0 = no maximum)"`
	MinArea       float64 `query:"min_area"         doc:"Minimum area (m², 0 = no minimum)"`
	MaxArea       float64 `query:"max_area"         doc:"Maximum area (m², 0 = no maximum)"`
	MinPricePerM2 float64 `query:"min_price_per_m2" doc:"Minimum price per square meter (EUR/m², 0 = no minimum)"`
	MaxPricePerM2 float64 `query:"max_price_per_m2" doc:"Maximum price per square meter (EUR/m², 0 = no maximum)"`
	Rooms         int32   `query:"rooms"            doc:"Exact room count (0 = no filter)"`
	Floor         int32   `query:"floor"            doc:"Exact floor level (0 = no filter)"`
	MinBuildYear  int32   `query:"min_build_year"   doc:"Minimum build year (0 = no minimum)"`
	MaxBuildYear  int32   `query:"max_build_year"   doc:"Maximum build year (0 = no maximum)"`
	Condition     string  `query:"condition"        doc:"Condition text filter"`
	EnergyClass   string  `query:"energy_class"     doc:"Energy class text filter"`
	Sort          string  `query:"sort"             doc:"Sort order: price_asc, price_desc, area_asc, area_desc, price_m2_asc, price_m2_desc, build_year_desc, seen_desc"`
	Page          int32   `query:"page"             doc:"Page number (1-based)" minimum:"1"`
	PageSize      int32   `query:"page_size"        doc:"Results per page: 25, 50, or 100"`
}

type propertyDetailInput struct {
	ID string `path:"id" required:"true" doc:"Canonical offering UUID"`
}

type propertySourceRawInput struct {
	ID       string `path:"id"        required:"true" doc:"Canonical offering UUID"`
	SourceID string `path:"sourceID"  required:"true" doc:"Source sale listing UUID"`
}

type transactionMatchPostalsInput struct {
	Limit int32 `query:"limit" doc:"Maximum postal codes to return"`
}

type transactionMatchCandidatesInput struct {
	Postal      string `query:"postal"      doc:"Postal code filter"`
	Status      string `query:"status"      doc:"Candidate status filter: candidate or ambiguous"`
	Transaction string `query:"transaction" doc:"Prices transaction UUID filter"`
	Limit       int32  `query:"limit"       doc:"Maximum candidates to return"`
}

type resolveCanonicalIDInput struct {
	URL string `query:"url" required:"true" doc:"Source URL"`
}

type resolveCanonicalIDOutput struct {
	Body struct {
		CanonicalID string `json:"canonical_id"`
		Source      string `json:"source"`
		Kind        string `json:"kind"`
		NativeID    string `json:"native_id"`
	}
}

type saleListingsSearchOutput struct {
	Body properties.Page[properties.SaleListingSummary]
}

type saleListingsMapInput struct {
	MinLat float64 `query:"min_lat" doc:"Minimum latitude"`
	MinLng float64 `query:"min_lng" doc:"Minimum longitude"`
	MaxLat float64 `query:"max_lat" doc:"Maximum latitude"`
	MaxLng float64 `query:"max_lng" doc:"Maximum longitude"`
	Source string  `query:"source"  doc:"Source filter: shortcut, frontdoor, or all"`
	Kind   string  `query:"kind"    doc:"Listing kind filter: ad, announcement, or all"`
	Limit  int32   `query:"limit"   doc:"Maximum markers to return"`
}

type saleListingsMapOutput struct {
	Body properties.SaleListingMap
}

type rentalsSearchOutput struct {
	Body properties.Page[properties.RentalSummary]
}

type saleListingDetailOutput struct {
	Body properties.SaleListing
}

type saleListingSourceRawOutput struct {
	Body json.RawMessage
}

type rentalDetailOutput struct {
	Body properties.Rental
}

type housingCompanyDetailOutput struct {
	Body properties.Building
}

type transactionMatchPostalsOutput struct {
	Body struct {
		Postals []properties.TransactionMatchPostalSummary `json:"postals"`
	}
}

type transactionMatchCandidatesOutput struct {
	Body struct {
		Candidates []properties.TransactionMatchCandidate `json:"candidates"`
	}
}

func (a *API) saleListingsSearchHandler(ctx context.Context, input *propertySearchInput) (*saleListingsSearchOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.sale_listings_search"))
	page, err := a.propertiesService.SearchSaleListings(ctx, propertySearchParams(input))
	if err != nil {
		logger.ErrorContext(ctx, "sale listing search failed", "error", err, "query", input.Query, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("sale listing search failed")
	}
	return &saleListingsSearchOutput{Body: page}, nil
}

func (a *API) saleListingsMapHandler(ctx context.Context, input *saleListingsMapInput) (*saleListingsMapOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.sale_listings_map"))
	markers, err := a.propertiesService.SaleListingMap(ctx, propertyMapBounds(input))
	if err != nil {
		logger.ErrorContext(ctx, "sale listing map failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("sale listing map failed")
	}
	return &saleListingsMapOutput{Body: markers}, nil
}

func (a *API) rentalsSearchHandler(ctx context.Context, input *propertySearchInput) (*rentalsSearchOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.rentals_search"))
	page, err := a.propertiesService.SearchRentals(ctx, propertySearchParams(input))
	if err != nil {
		logger.ErrorContext(ctx, "rental search failed", "error", err, "query", input.Query, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("rental search failed")
	}
	return &rentalsSearchOutput{Body: page}, nil
}

func (a *API) saleListingDetailHandler(ctx context.Context, input *propertyDetailInput) (*saleListingDetailOutput, error) {
	listing, err := a.propertiesService.SaleListingByID(ctx, input.ID, "", "")
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		a.logger.ErrorContext(ctx, "sale listing detail failed", "id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("invalid sale listing canonical ID")
	}
	return &saleListingDetailOutput{Body: listing}, nil
}

func (a *API) saleListingSourceRawHandler(ctx context.Context, input *propertySourceRawInput) (*saleListingSourceRawOutput, error) {
	payload, err := a.propertiesService.SaleOfferingSourceRawPayload(ctx, input.ID, input.SourceID)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("source payload not found")
		}
		a.logger.ErrorContext(ctx, "sale listing source payload failed", "id", input.ID, "source_id", input.SourceID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("invalid source payload request")
	}
	return &saleListingSourceRawOutput{Body: payload.Payload}, nil
}

func (a *API) transactionMatchPostalsHandler(ctx context.Context, input *transactionMatchPostalsInput) (*transactionMatchPostalsOutput, error) {
	postals, err := a.propertiesService.TransactionMatchPostals(ctx, input.Limit)
	if err != nil {
		a.logger.ErrorContext(ctx, "transaction match postal list failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("transaction match postal list failed")
	}
	out := &transactionMatchPostalsOutput{}
	out.Body.Postals = postals
	return out, nil
}

func (a *API) transactionMatchCandidatesHandler(ctx context.Context, input *transactionMatchCandidatesInput) (*transactionMatchCandidatesOutput, error) {
	candidates, err := a.propertiesService.TransactionMatchCandidates(ctx, input.Postal, input.Status, input.Transaction, input.Limit)
	if err != nil {
		a.logger.ErrorContext(ctx, "transaction match candidate list failed", "postal", input.Postal, "status", input.Status, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("transaction match candidate list failed")
	}
	out := &transactionMatchCandidatesOutput{}
	out.Body.Candidates = candidates
	return out, nil
}

func (a *API) rentalDetailHandler(ctx context.Context, input *propertyDetailInput) (*rentalDetailOutput, error) {
	rental, err := a.propertiesService.RentalByID(ctx, input.ID, "", "")
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("rental not found")
		}
		return nil, huma.Error400BadRequest("invalid rental canonical ID")
	}
	return &rentalDetailOutput{Body: rental}, nil
}

func (a *API) housingCompanyDetailHandler(ctx context.Context, input *propertyDetailInput) (*housingCompanyDetailOutput, error) {
	building, err := a.propertiesService.BuildingByID(ctx, input.ID, "", "")
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("housing company not found")
		}
		return nil, huma.Error400BadRequest("invalid housing company canonical ID")
	}
	return &housingCompanyDetailOutput{Body: building}, nil
}

func (a *API) resolveCanonicalIDHandler(ctx context.Context, input *resolveCanonicalIDInput) (*resolveCanonicalIDOutput, error) {
	if !strings.HasPrefix(strings.TrimSpace(input.URL), "http://") && !strings.HasPrefix(strings.TrimSpace(input.URL), "https://") {
		return nil, huma.Error400BadRequest("url must be an http or https source URL")
	}
	canonicalID, err := ads.ResolveInput(input.URL, a.cfg.Shortcut.SitemapBase, a.cfg.Frontdoor.SitemapBase)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid source URL")
	}
	source, kind, nativeID, err := ads.ParseCanonicalID(canonicalID)
	if err != nil {
		return nil, huma.Error400BadRequest("resolved invalid canonical ID")
	}
	out := &resolveCanonicalIDOutput{}
	out.Body.CanonicalID = canonicalID
	out.Body.Source = source
	out.Body.Kind = kind
	out.Body.NativeID = nativeID
	return out, nil
}

func propertySearchParams(input *propertySearchInput) properties.SearchParams {
	page := max(input.Page, 1)
	pageSize := input.PageSize
	if pageSize <= 0 {
		pageSize = 25
	}
	return properties.SearchParams{Query: input.Query, Source: input.Source, Kind: input.Kind, City: input.City, Postal: input.Postal, MinPrice: positiveInt64Ptr(input.MinPrice), MaxPrice: positiveInt64Ptr(input.MaxPrice), MinArea: positiveFloat64Ptr(input.MinArea), MaxArea: positiveFloat64Ptr(input.MaxArea), MinPricePerM2: positiveFloat64Ptr(input.MinPricePerM2), MaxPricePerM2: positiveFloat64Ptr(input.MaxPricePerM2), Rooms: positiveInt32Ptr(input.Rooms), Floor: positiveInt32Ptr(input.Floor), MinBuildYear: positiveInt32Ptr(input.MinBuildYear), MaxBuildYear: positiveInt32Ptr(input.MaxBuildYear), Condition: input.Condition, EnergyClass: input.EnergyClass, Sort: input.Sort, Page: page, PageSize: pageSize}
}

func propertyMapBounds(input *saleListingsMapInput) properties.MapBounds {
	var minLat, minLng, maxLat, maxLng *float64
	if input.MinLat != 0 || input.MinLng != 0 || input.MaxLat != 0 || input.MaxLng != 0 {
		minLat = &input.MinLat
		minLng = &input.MinLng
		maxLat = &input.MaxLat
		maxLng = &input.MaxLng
	}
	return properties.MapBounds{MinLat: minLat, MinLng: minLng, MaxLat: maxLat, MaxLng: maxLng, Source: input.Source, Kind: input.Kind, Limit: input.Limit}
}

func positiveInt64Ptr(value int64) *int64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func positiveFloat64Ptr(value float64) *float64 {
	if value <= 0 {
		return nil
	}
	return &value
}

func positiveInt32Ptr(value int32) *int32 {
	if value <= 0 {
		return nil
	}
	return &value
}
