package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"koditon/internal/domain/ads"
	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
	"koditon/internal/sync/consumers"
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

type saleListingRenovationExtractInput struct {
	ID    string `path:"id"     required:"true" doc:"Canonical offering UUID"`
	Model string `query:"model" doc:"OpenRouter model ID, defaults to the configured renovation extractor model"`
}

type saleListingDescriptionExtractInput struct {
	ID    string `path:"id"     required:"true" doc:"Canonical offering UUID"`
	Model string `query:"model" doc:"OpenRouter model ID, defaults to the configured extractor model"`
}

type saleListingValuationInputsExtractInput struct {
	ID    string `path:"id"     required:"true" doc:"Canonical offering UUID"`
	Model string `query:"model" doc:"OpenRouter model ID, defaults to the configured extractor model"`
}

type saleListingCanonicalProfileProjectInput struct {
	ID string `path:"id" required:"true" doc:"Canonical offering UUID"`
}

type saleListingHouseOverviewGenerateInput struct {
	ID    string `path:"id"     required:"true" doc:"Canonical offering UUID"`
	Model string `query:"model" doc:"OpenRouter model ID, defaults to the configured extractor model"`
}

type saleListingManagerCertificateUploadInput struct {
	ID      string `path:"id" required:"true" doc:"Canonical offering UUID"`
	RawBody huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"application/pdf" required:"true"`
	}]
}

type managerCertificateUploadInput struct {
	OfferingID string `query:"offering_id" doc:"Optional canonical offering UUID"`
	RawBody    huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"application/pdf" required:"true"`
	}]
}

type propertyDocumentInput struct {
	ID string `path:"id" required:"true" doc:"Property document UUID"`
}

type propertyDocumentExtractInput struct {
	ID    string `path:"id"     required:"true" doc:"Property document UUID"`
	Model string `query:"model" doc:"OpenAI model ID, defaults to the configured manager certificate model"`
}

type propertyDocumentAttachInput struct {
	ID   string `path:"id" required:"true" doc:"Property document UUID"`
	Body struct {
		OfferingID string `json:"offering_id" required:"true" doc:"Target canonical offering UUID"`
	}
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
	MinLat         float64 `query:"min_lat" doc:"Minimum latitude"`
	MinLng         float64 `query:"min_lng" doc:"Minimum longitude"`
	MaxLat         float64 `query:"max_lat" doc:"Maximum latitude"`
	MaxLng         float64 `query:"max_lng" doc:"Maximum longitude"`
	Query          string  `query:"q" doc:"Text search"`
	City           string  `query:"city" doc:"City filter"`
	Postal         string  `query:"postal" doc:"Postal code filter"`
	Source         string  `query:"source"  doc:"Source filter: shortcut, frontdoor, or all"`
	Kind           string  `query:"kind"    doc:"Listing kind filter: ad, announcement, or all"`
	MinPrice       int64   `query:"min_price" doc:"Minimum asking price"`
	MaxPrice       int64   `query:"max_price" doc:"Maximum asking price"`
	MinArea        float64 `query:"min_area" doc:"Minimum area"`
	MaxArea        float64 `query:"max_area" doc:"Maximum area"`
	MinPricePerM2  float64 `query:"min_price_m2" doc:"Minimum price per square meter"`
	MaxPricePerM2  float64 `query:"max_price_m2" doc:"Maximum price per square meter"`
	Rooms          int32   `query:"rooms" doc:"Exact room count"`
	MinBuildYear   int32   `query:"min_build_year" doc:"Minimum build year"`
	MaxBuildYear   int32   `query:"max_build_year" doc:"Maximum build year"`
	PropertyType   string  `query:"property_type" doc:"Property type code"`
	Condition      string  `query:"condition" doc:"Condition code"`
	EnergyClass    string  `query:"energy_class" doc:"Energy class text or match code"`
	Elevator       string  `query:"elevator" doc:"true or false"`
	Sauna          string  `query:"sauna" doc:"true or false"`
	Balcony        string  `query:"balcony" doc:"true or false"`
	PlotOwned      string  `query:"plot_owned" doc:"true or false"`
	NewDevelopment string  `query:"new_development" doc:"true or false"`
	HasTransaction string  `query:"has_transaction" doc:"true or false"`
	Limit          int32   `query:"limit"   doc:"Maximum markers to return"`
}

type saleListingsMapFilterOptionsInput struct {
	Source string `query:"source" doc:"Source filter: shortcut, frontdoor, or all"`
	Kind   string `query:"kind"   doc:"Listing kind filter: ad, announcement, or all"`
}

type saleListingsMapOutput struct {
	Body properties.SaleListingMap
}

type saleListingsMapFilterOptionsOutput struct {
	Body properties.SaleListingMapFilterOptions
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

type saleListingRenovationExtractOutput struct {
	Body properties.RenovationExtractionResult
}

type saleListingDescriptionExtractOutput struct {
	Body properties.DescriptionExtractionResult
}

type saleListingValuationInputsExtractOutput struct {
	Body properties.ValuationInputExtractionResult
}

type saleListingCanonicalProfileProjectOutput struct {
	Body properties.CanonicalProfileProjectionResult
}

type saleListingHouseOverviewGenerateOutput struct {
	Body properties.HouseOverviewGenerationResult
}

type propertyDocumentJobResult struct {
	Document properties.PropertyDocumentSummary `json:"document"`
	JobID    string                             `json:"job_id"`
	Queued   bool                               `json:"queued"`
}

type propertyDocumentJobOutput struct {
	Body propertyDocumentJobResult
}

type propertyDocumentDownloadOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	ContentLength      int64  `header:"Content-Length"`
	Body               []byte
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

func (a *API) saleListingsMapFilterOptionsHandler(ctx context.Context, input *saleListingsMapFilterOptionsInput) (*saleListingsMapFilterOptionsOutput, error) {
	logger := logging.With(a.logger, logging.Op("api.sale_listings_map_filter_options"))
	options, err := a.propertiesService.SaleListingMapFilterOptions(ctx, input.Source, input.Kind)
	if err != nil {
		logger.ErrorContext(ctx, "sale listing map filter options failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("sale listing map filter options failed")
	}
	return &saleListingsMapFilterOptionsOutput{Body: options}, nil
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
		return nil, huma.Error500InternalServerError("sale listing detail failed")
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

func (a *API) saleListingRenovationExtractHandler(ctx context.Context, input *saleListingRenovationExtractInput) (*saleListingRenovationExtractOutput, error) {
	result, err := a.propertiesService.ExtractSaleListingRenovations(ctx, input.ID, input.Model)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		if errors.Is(err, properties.ErrRenovationExtractorNotConfigured) {
			return nil, huma.Error503ServiceUnavailable("renovation extractor not configured")
		}
		a.logger.ErrorContext(ctx, "sale listing renovation extraction failed", "id", input.ID, "model", input.Model, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("renovation extraction failed")
	}
	return &saleListingRenovationExtractOutput{Body: result}, nil
}

func (a *API) saleListingDescriptionExtractHandler(ctx context.Context, input *saleListingDescriptionExtractInput) (*saleListingDescriptionExtractOutput, error) {
	result, err := a.propertiesService.ExtractSaleListingDescriptionInsights(ctx, input.ID, input.Model)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		if errors.Is(err, properties.ErrRenovationExtractorNotConfigured) {
			return nil, huma.Error503ServiceUnavailable("description extractor not configured")
		}
		a.logger.ErrorContext(ctx, "sale listing description extraction failed", "id", input.ID, "model", input.Model, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("description extraction failed")
	}
	return &saleListingDescriptionExtractOutput{Body: result}, nil
}

func (a *API) saleListingValuationInputsExtractHandler(ctx context.Context, input *saleListingValuationInputsExtractInput) (*saleListingValuationInputsExtractOutput, error) {
	result, err := a.propertiesService.ExtractSaleListingValuationInputs(ctx, input.ID, input.Model)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		if errors.Is(err, properties.ErrRenovationExtractorNotConfigured) {
			return nil, huma.Error503ServiceUnavailable("valuation input extractor not configured")
		}
		a.logger.ErrorContext(ctx, "sale listing valuation input extraction failed", "id", input.ID, "model", input.Model, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("valuation input extraction failed")
	}
	return &saleListingValuationInputsExtractOutput{Body: result}, nil
}

func (a *API) saleListingCanonicalProfileProjectHandler(ctx context.Context, input *saleListingCanonicalProfileProjectInput) (*saleListingCanonicalProfileProjectOutput, error) {
	result, err := a.propertiesService.ProjectSaleListingCanonicalProfile(ctx, input.ID)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		a.logger.ErrorContext(ctx, "sale listing canonical profile projection failed", "id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("canonical profile projection failed")
	}
	return &saleListingCanonicalProfileProjectOutput{Body: result}, nil
}

func (a *API) saleListingHouseOverviewGenerateHandler(ctx context.Context, input *saleListingHouseOverviewGenerateInput) (*saleListingHouseOverviewGenerateOutput, error) {
	result, err := a.propertiesService.GenerateSaleListingHouseOverview(ctx, input.ID, input.Model)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		if errors.Is(err, properties.ErrRenovationExtractorNotConfigured) {
			return nil, huma.Error503ServiceUnavailable("house overview generator not configured")
		}
		a.logger.ErrorContext(ctx, "sale listing house overview generation failed", "id", input.ID, "model", input.Model, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error400BadRequest("house overview generation failed")
	}
	return &saleListingHouseOverviewGenerateOutput{Body: result}, nil
}

func (a *API) saleListingManagerCertificateUploadHandler(ctx context.Context, input *saleListingManagerCertificateUploadInput) (*propertyDocumentJobOutput, error) {
	file := input.RawBody.Data().File
	if !file.IsSet {
		return nil, huma.Error400BadRequest("manager certificate PDF is required")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, huma.Error400BadRequest("read manager certificate PDF failed")
	}
	document, err := a.propertiesService.UploadManagerCertificate(ctx, input.ID, properties.PropertyDocumentUpload{Filename: file.Filename, MimeType: file.ContentType, Bytes: data})
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		if errors.Is(err, properties.ErrPropertyDocumentTooLarge) {
			return nil, huma.Error413RequestEntityTooLarge("manager certificate PDF too large")
		}
		if errors.Is(err, properties.ErrPropertyDocumentInvalid) {
			return nil, huma.Error400BadRequest("invalid manager certificate PDF")
		}
		a.logger.ErrorContext(ctx, "manager certificate upload failed", "id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate upload failed")
	}
	result, err := a.enqueueManagerCertificateExtraction(ctx, document, "")
	if err != nil {
		a.logger.ErrorContext(ctx, "manager certificate extraction enqueue failed", "id", document.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate extraction enqueue failed")
	}
	return &propertyDocumentJobOutput{Body: result}, nil
}

func (a *API) managerCertificateUploadHandler(ctx context.Context, input *managerCertificateUploadInput) (*propertyDocumentJobOutput, error) {
	file := input.RawBody.Data().File
	if !file.IsSet {
		return nil, huma.Error400BadRequest("manager certificate PDF is required")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, huma.Error400BadRequest("read manager certificate PDF failed")
	}
	upload := properties.PropertyDocumentUpload{Filename: file.Filename, MimeType: file.ContentType, Bytes: data}
	var document properties.PropertyDocumentSummary
	if strings.TrimSpace(input.OfferingID) == "" {
		document, err = a.propertiesService.UploadDetachedManagerCertificate(ctx, upload)
	} else {
		document, err = a.propertiesService.UploadManagerCertificate(ctx, input.OfferingID, upload)
	}
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("sale listing not found")
		}
		if errors.Is(err, properties.ErrPropertyDocumentTooLarge) {
			return nil, huma.Error413RequestEntityTooLarge("manager certificate PDF too large")
		}
		if errors.Is(err, properties.ErrPropertyDocumentInvalid) {
			return nil, huma.Error400BadRequest("invalid manager certificate PDF")
		}
		a.logger.ErrorContext(ctx, "manager certificate upload failed", "offering_id", input.OfferingID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate upload failed")
	}
	result, err := a.enqueueManagerCertificateExtraction(ctx, document, "")
	if err != nil {
		a.logger.ErrorContext(ctx, "manager certificate extraction enqueue failed", "id", document.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate extraction enqueue failed")
	}
	return &propertyDocumentJobOutput{Body: result}, nil
}

func (a *API) enqueueManagerCertificateExtraction(ctx context.Context, document properties.PropertyDocumentSummary, model string) (propertyDocumentJobResult, error) {
	payload, err := json.Marshal(map[string]string{"document_id": document.ID, "model": strings.TrimSpace(model)})
	if err != nil {
		return propertyDocumentJobResult{}, fmt.Errorf("marshal manager certificate extraction payload: %w", err)
	}
	jobID, queued, err := a.spawnSyncWorkflow(ctx, consumers.TaskTypeCanonicalExtractManagerCertificate, payload)
	if err != nil {
		return propertyDocumentJobResult{}, err
	}
	return propertyDocumentJobResult{Document: document, JobID: jobID, Queued: queued}, nil
}

func (a *API) enqueueManagerCertificateProjection(ctx context.Context, document properties.PropertyDocumentSummary) (propertyDocumentJobResult, error) {
	payload, err := json.Marshal(map[string]string{"document_id": document.ID})
	if err != nil {
		return propertyDocumentJobResult{}, fmt.Errorf("marshal manager certificate projection payload: %w", err)
	}
	jobID, queued, err := a.spawnSyncWorkflow(ctx, consumers.TaskTypeCanonicalProjectManagerCertificate, payload)
	if err != nil {
		return propertyDocumentJobResult{}, err
	}
	return propertyDocumentJobResult{Document: document, JobID: jobID, Queued: queued}, nil
}

func (a *API) propertyDocumentDownloadHandler(ctx context.Context, input *propertyDocumentInput) (*propertyDocumentDownloadOutput, error) {
	document, err := a.propertiesService.DownloadPropertyDocument(ctx, input.ID)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("property document not found")
		}
		a.logger.ErrorContext(ctx, "property document download failed", "id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("property document download failed")
	}
	return &propertyDocumentDownloadOutput{ContentType: document.MimeType, ContentDisposition: fmt.Sprintf("attachment; filename=%q", document.Filename), ContentLength: int64(len(document.Bytes)), Body: document.Bytes}, nil
}

func (a *API) propertyDocumentExtractHandler(ctx context.Context, input *propertyDocumentExtractInput) (*propertyDocumentJobOutput, error) {
	summary, err := a.propertiesService.PropertyDocumentSummary(ctx, input.ID)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("property document not found")
		}
		a.logger.ErrorContext(ctx, "property document lookup failed", "id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("property document lookup failed")
	}
	result, err := a.enqueueManagerCertificateExtraction(ctx, summary, input.Model)
	if err != nil {
		a.logger.ErrorContext(ctx, "manager certificate extraction enqueue failed", "id", input.ID, "model", input.Model, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate extraction enqueue failed")
	}
	return &propertyDocumentJobOutput{Body: result}, nil
}

func (a *API) propertyDocumentAttachHandler(ctx context.Context, input *propertyDocumentAttachInput) (*propertyDocumentJobOutput, error) {
	document, err := a.propertiesService.AttachPropertyDocumentToOffering(ctx, input.ID, input.Body.OfferingID)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("property document or target offering not found")
		}
		a.logger.ErrorContext(ctx, "property document attach failed", "id", input.ID, "offering_id", input.Body.OfferingID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("property document attach failed")
	}
	result, err := a.enqueueManagerCertificateProjection(ctx, document)
	if err != nil {
		a.logger.ErrorContext(ctx, "manager certificate projection enqueue failed", "id", input.ID, "offering_id", input.Body.OfferingID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate projection enqueue failed")
	}
	return &propertyDocumentJobOutput{Body: result}, nil
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
	if strings.TrimSpace(input.Transaction) != "" {
		if _, err := uuid.Parse(strings.TrimSpace(input.Transaction)); err != nil {
			return nil, huma.Error400BadRequest("transaction must be a valid UUID")
		}
	}
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
	return properties.MapBounds{MinLat: minLat, MinLng: minLng, MaxLat: maxLat, MaxLng: maxLng, Query: input.Query, City: input.City, Postal: input.Postal, Source: input.Source, Kind: input.Kind, MinPrice: positiveInt64Ptr(input.MinPrice), MaxPrice: positiveInt64Ptr(input.MaxPrice), MinArea: positiveFloat64Ptr(input.MinArea), MaxArea: positiveFloat64Ptr(input.MaxArea), MinPricePerM2: positiveFloat64Ptr(input.MinPricePerM2), MaxPricePerM2: positiveFloat64Ptr(input.MaxPricePerM2), Rooms: positiveInt32Ptr(input.Rooms), MinBuildYear: positiveInt32Ptr(input.MinBuildYear), MaxBuildYear: positiveInt32Ptr(input.MaxBuildYear), PropertyType: input.PropertyType, Condition: input.Condition, EnergyClass: input.EnergyClass, Elevator: boolQueryPtr(input.Elevator), Sauna: boolQueryPtr(input.Sauna), Balcony: boolQueryPtr(input.Balcony), PlotOwned: boolQueryPtr(input.PlotOwned), NewDevelopment: boolQueryPtr(input.NewDevelopment), HasTransaction: boolQueryPtr(input.HasTransaction), Limit: input.Limit}
}

func boolQueryPtr(value string) *bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		out := true
		return &out
	case "false", "0", "no":
		out := false
		return &out
	default:
		return nil
	}
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
