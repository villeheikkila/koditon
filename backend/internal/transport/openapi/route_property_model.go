package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"koditon/internal/domain/properties"
	"koditon/internal/platform/logging"
	"koditon/internal/platform/taskqueue"
	"koditon/internal/sync/consumers"
	syncjobs "koditon/internal/sync/jobs"
)

type canonicalTargetInput struct {
	TargetType string `path:"targetType" required:"true" doc:"Canonical target type: offering, unit, building, housing_company"`
	TargetID   string `path:"targetID"   required:"true" doc:"Canonical target UUID"`
}

type propertyTargetsMapInput struct {
	MinLat float64 `query:"min_lat" doc:"Minimum latitude for visible map bounds, 0 = no bounds"`
	MinLng float64 `query:"min_lng" doc:"Minimum longitude for visible map bounds, 0 = no bounds"`
	MaxLat float64 `query:"max_lat" doc:"Maximum latitude for visible map bounds, 0 = no bounds"`
	MaxLng float64 `query:"max_lng" doc:"Maximum longitude for visible map bounds, 0 = no bounds"`
	Query  string  `query:"q"       doc:"Search by housing company name, address, city, or postal code"`
	Limit  int     `query:"limit"   doc:"Maximum number of housing company markers"`
}

type canonicalTargetOutput struct {
	Body CanonicalTargetResource
}

type propertyTargetsMapOutput struct {
	Body PropertyTargetMap
}

type resolvedValuesOutput struct {
	Body struct {
		Target CanonicalTargetRef `json:"target"`
		Values []ResolvedValue    `json:"values"`
	}
}

type sourceClaimsOutput struct {
	Body struct {
		Target CanonicalTargetRef `json:"target"`
		Claims []SourceClaim      `json:"claims"`
	}
}

type renovationEventsOutput struct {
	Body struct {
		Target CanonicalTargetRef `json:"target"`
		Events []RenovationEvent  `json:"events"`
	}
}

type targetDocumentsOutput struct {
	Body struct {
		Target    CanonicalTargetRef                   `json:"target"`
		Documents []properties.PropertyDocumentSummary `json:"documents"`
	}
}

type resolveTargetOutput struct {
	Body QueuedCanonicalJob `json:"body"`
}

type modelManagerCertificateUploadInput struct {
	TargetType string `query:"target_type" doc:"Optional target type: offering, unit, building, housing_company"`
	TargetID   string `query:"target_id"   doc:"Optional target UUID"`
	RawBody    huma.MultipartFormFiles[struct {
		File huma.FormFile `form:"file" contentType:"application/pdf" required:"true"`
	}]
}

type modelPropertyDocumentInput struct {
	ID string `path:"id" required:"true" doc:"Property document UUID"`
}

type propertyDocumentSummaryOutput struct {
	Body properties.PropertyDocumentSummary
}

type propertyDocumentAttachOutput struct {
	Body struct {
		Document properties.PropertyDocumentSummary `json:"document"`
		Job      QueuedCanonicalJob                 `json:"job"`
	}
}

type propertyDocumentAttachModelInput struct {
	ID   string `path:"id" required:"true" doc:"Property document UUID"`
	Body struct {
		Target *CanonicalTargetRef `json:"target,omitempty" doc:"New target. Omit or null to detach."`
	}
}

type CanonicalTargetRef struct {
	Type string `json:"type" enum:"offering,unit,building,housing_company,listing,document,transaction"`
	ID   string `json:"id"`
}

type CanonicalTargetResource struct {
	Target           CanonicalTargetRef                   `json:"target"`
	Overview         *TargetOverview                      `json:"overview,omitempty"`
	ResolvedValues   []ResolvedValue                      `json:"resolved_values"`
	RenovationEvents []RenovationEvent                    `json:"renovation_events,omitempty"`
	Documents        []properties.PropertyDocumentSummary `json:"documents,omitempty"`
	Buildings        []TargetBuildingSummary              `json:"buildings,omitempty"`
	Units            []TargetUnitSummary                  `json:"units,omitempty"`
	Offerings        []TargetOfferingSummary              `json:"offerings,omitempty"`
	Sources          []TargetSourceLink                   `json:"sources,omitempty"`
}

type TargetOverview struct {
	Title    string                  `json:"title"`
	Subtitle string                  `json:"subtitle,omitempty"`
	Fields   []TargetOverviewField   `json:"fields,omitempty"`
	Related  []TargetOverviewRelated `json:"related,omitempty"`
	Sources  []TargetSourceLink      `json:"sources,omitempty"`
}

type TargetOverviewField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type TargetOverviewRelated struct {
	Label  string             `json:"label"`
	Title  string             `json:"title"`
	Target CanonicalTargetRef `json:"target"`
}

type TargetSourceLink struct {
	Label         string     `json:"label"`
	Provider      string     `json:"provider"`
	Kind          string     `json:"kind"`
	SourceTable   string     `json:"source_table,omitempty"`
	SourceID      string     `json:"source_id,omitempty"`
	SourceIDValue string     `json:"source_id_value,omitempty"`
	ExternalID    string     `json:"external_id,omitempty"`
	Title         string     `json:"title"`
	URL           string     `json:"url,omitempty"`
	LinkStatus    string     `json:"link_status,omitempty"`
	LastSeenAt    *time.Time `json:"last_seen_at,omitempty"`
}

type TargetBuildingSummary struct {
	Target        CanonicalTargetRef  `json:"target"`
	HousingTarget *CanonicalTargetRef `json:"housing_target,omitempty"`
	Title         string              `json:"title"`
	Address       string              `json:"address,omitempty"`
	City          string              `json:"city,omitempty"`
	Postal        string              `json:"postal,omitempty"`
	BuildYear     *int32              `json:"build_year,omitempty"`
	Lat           *float64            `json:"lat,omitempty"`
	Lng           *float64            `json:"lng,omitempty"`
	UnitCount     int64               `json:"unit_count"`
	OfferingCount int64               `json:"offering_count"`
}

type TargetUnitSummary struct {
	Target         CanonicalTargetRef  `json:"target"`
	BuildingTarget *CanonicalTargetRef `json:"building_target,omitempty"`
	HousingTarget  *CanonicalTargetRef `json:"housing_target,omitempty"`
	Title          string              `json:"title"`
	Address        string              `json:"address,omitempty"`
	Layout         string              `json:"layout,omitempty"`
	AreaM2         *float64            `json:"area_m2,omitempty"`
	Floor          string              `json:"floor,omitempty"`
	OfferingCount  int64               `json:"offering_count"`
}

type TargetOfferingSummary struct {
	Target         CanonicalTargetRef  `json:"target"`
	UnitTarget     CanonicalTargetRef  `json:"unit_target"`
	BuildingTarget *CanonicalTargetRef `json:"building_target,omitempty"`
	HousingTarget  *CanonicalTargetRef `json:"housing_target,omitempty"`
	Title          string              `json:"title"`
	Layout         string              `json:"layout,omitempty"`
	AreaM2         *float64            `json:"area_m2,omitempty"`
	AskingPriceEUR *int64              `json:"asking_price_eur,omitempty"`
	LastSeenAt     *time.Time          `json:"last_seen_at,omitempty"`
}

type EvidenceRef struct {
	ID                string          `json:"id,omitempty"`
	Kind              string          `json:"kind" enum:"claim,renovation_event,document,listing,manual"`
	Scope             string          `json:"scope,omitempty"`
	SourceTable       string          `json:"source_table,omitempty"`
	SourceID          string          `json:"source_id,omitempty"`
	SourceField       string          `json:"source_field,omitempty"`
	ProjectionVersion string          `json:"projection_version,omitempty"`
	ObservedAt        *time.Time      `json:"observed_at,omitempty"`
	CreatedAt         *time.Time      `json:"created_at,omitempty"`
	Confidence        *float64        `json:"confidence,omitempty"`
	SourceReliability *float64        `json:"source_reliability,omitempty"`
	Evidence          json.RawMessage `json:"evidence,omitempty"`
}

type ResolvedValue struct {
	Target             CanonicalTargetRef `json:"target"`
	DimensionKey       string             `json:"dimension_key"`
	Value              json.RawMessage    `json:"value"`
	ValueKind          string             `json:"value_kind"`
	Unit               string             `json:"unit,omitempty"`
	Confidence         float64            `json:"confidence"`
	SelectedReason     string             `json:"selected_reason"`
	ConflictStatus     string             `json:"conflict_status"`
	SelectedEvidence   *EvidenceRef       `json:"selected_evidence,omitempty"`
	SupportingClaimIDs []string           `json:"supporting_claim_ids,omitempty"`
	RejectedClaimIDs   []string           `json:"rejected_claim_ids,omitempty"`
	ResolvedAt         time.Time          `json:"resolved_at"`
}

type SourceClaim struct {
	ID                string             `json:"id"`
	Target            CanonicalTargetRef `json:"target"`
	DimensionKey      string             `json:"dimension_key"`
	Value             json.RawMessage    `json:"value"`
	ValueKind         string             `json:"value_kind"`
	Unit              string             `json:"unit,omitempty"`
	ClaimScope        string             `json:"claim_scope"`
	SourceTable       string             `json:"source_table"`
	SourceID          string             `json:"source_id"`
	SourceField       string             `json:"source_field,omitempty"`
	ProjectionVersion string             `json:"projection_version"`
	ObservedAt        *time.Time         `json:"observed_at,omitempty"`
	ValidFrom         string             `json:"valid_from,omitempty"`
	ValidUntil        string             `json:"valid_until,omitempty"`
	Confidence        float64            `json:"confidence"`
	SourceReliability float64            `json:"source_reliability"`
	Evidence          json.RawMessage    `json:"evidence,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type RenovationEvent struct {
	ID                string             `json:"id"`
	Target            CanonicalTargetRef `json:"target"`
	Scope             string             `json:"scope"`
	SourceTable       string             `json:"source_table"`
	SourceID          string             `json:"source_id"`
	SourceField       string             `json:"source_field,omitempty"`
	ProjectionVersion string             `json:"projection_version"`
	Category          string             `json:"category"`
	Component         string             `json:"component,omitempty"`
	Status            string             `json:"status"`
	Stage             string             `json:"stage,omitempty"`
	EventScope        string             `json:"event_scope,omitempty"`
	Responsibility    string             `json:"responsibility,omitempty"`
	Year              *int32             `json:"year,omitempty"`
	StartYear         *int32             `json:"start_year,omitempty"`
	EndYear           *int32             `json:"end_year,omitempty"`
	CostEstimateEUR   *int64             `json:"cost_estimate_eur,omitempty"`
	Summary           string             `json:"summary,omitempty"`
	Evidence          json.RawMessage    `json:"evidence,omitempty"`
	Confidence        float64            `json:"confidence"`
	SourceReliability float64            `json:"source_reliability"`
	ObservedAt        *time.Time         `json:"observed_at,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
}

type QueuedCanonicalJob struct {
	JobID  string `json:"job_id"`
	Queued bool   `json:"queued"`
}

type PropertyTargetMap struct {
	Markers []PropertyTargetMapMarker `json:"markers"`
}

type PropertyTargetMapMarker struct {
	Target         CanonicalTargetRef          `json:"target"`
	FallbackTarget *CanonicalTargetRef         `json:"fallback_target,omitempty"`
	Title          string                      `json:"title,omitempty"`
	Name           string                      `json:"name,omitempty"`
	Address        string                      `json:"address,omitempty"`
	City           string                      `json:"city,omitempty"`
	Postal         string                      `json:"postal,omitempty"`
	BuildYear      *int32                      `json:"build_year,omitempty"`
	Lat            float64                     `json:"lat"`
	Lng            float64                     `json:"lng"`
	BuildingCount  int64                       `json:"building_count"`
	UnitCount      int64                       `json:"unit_count"`
	OfferingCount  int64                       `json:"offering_count"`
	SourceCount    int64                       `json:"source_count"`
	DocumentCount  int64                       `json:"document_count"`
	Offerings      []PropertyTargetMapOffering `json:"offerings"`
}

type PropertyTargetMapOffering struct {
	Target     CanonicalTargetRef `json:"target"`
	UnitTarget CanonicalTargetRef `json:"unit_target"`
	Headline   string             `json:"headline,omitempty"`
	RoomLayout string             `json:"room_layout,omitempty"`
	AreaM2     *float64           `json:"area_m2,omitempty"`
	PriceEUR   *int64             `json:"price_eur,omitempty"`
	LastSeenAt *time.Time         `json:"last_seen_at,omitempty"`
}

func (a *API) propertyTargetsMapHandler(ctx context.Context, input *propertyTargetsMapInput) (*propertyTargetsMapOutput, error) {
	markers, err := a.listPropertyTargetMapMarkers(ctx, input)
	if err != nil {
		a.logger.ErrorContext(ctx, "list property target map markers failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("property target map failed")
	}
	return &propertyTargetsMapOutput{Body: PropertyTargetMap{Markers: markers}}, nil
}

func (a *API) canonicalTargetHandler(ctx context.Context, input *canonicalTargetInput) (*canonicalTargetOutput, error) {
	target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	overview, err := a.getTargetOverview(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "get target overview failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("target overview failed")
	}
	values, err := a.listResolvedValues(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list resolved values failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("resolved values failed")
	}
	events, err := a.listRenovationEvents(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list renovation events failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("renovation events failed")
	}
	documents, err := a.listTargetDocuments(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list target documents failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("documents failed")
	}
	buildings, units, offerings, err := a.listTargetChildren(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list target children failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("target children failed")
	}
	sources, err := a.listTargetSources(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list target sources failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("target sources failed")
	}
	return &canonicalTargetOutput{Body: CanonicalTargetResource{Target: target, Overview: overview, ResolvedValues: values, RenovationEvents: events, Documents: documents, Buildings: buildings, Units: units, Offerings: offerings, Sources: sources}}, nil
}

func (a *API) resolvedValuesHandler(ctx context.Context, input *canonicalTargetInput) (*resolvedValuesOutput, error) {
	target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	values, err := a.listResolvedValues(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list resolved values failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("resolved values failed")
	}
	out := &resolvedValuesOutput{}
	out.Body.Target = target
	out.Body.Values = values
	return out, nil
}

func (a *API) sourceClaimsHandler(ctx context.Context, input *canonicalTargetInput) (*sourceClaimsOutput, error) {
	target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	claims, err := a.listSourceClaims(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list source claims failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("claims failed")
	}
	out := &sourceClaimsOutput{}
	out.Body.Target = target
	out.Body.Claims = claims
	return out, nil
}

func (a *API) renovationEventsHandler(ctx context.Context, input *canonicalTargetInput) (*renovationEventsOutput, error) {
	target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	events, err := a.listRenovationEvents(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list renovation events failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("renovation events failed")
	}
	out := &renovationEventsOutput{}
	out.Body.Target = target
	out.Body.Events = events
	return out, nil
}

func (a *API) targetDocumentsHandler(ctx context.Context, input *canonicalTargetInput) (*targetDocumentsOutput, error) {
	target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	documents, err := a.listTargetDocuments(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "list target documents failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("documents failed")
	}
	out := &targetDocumentsOutput{}
	out.Body.Target = target
	out.Body.Documents = documents
	return out, nil
}

func (a *API) resolveTargetHandler(ctx context.Context, input *canonicalTargetInput) (*resolveTargetOutput, error) {
	target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
	if err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	job, err := a.enqueueTargetResolution(ctx, target)
	if err != nil {
		a.logger.ErrorContext(ctx, "enqueue target resolution failed", "target_type", target.Type, "target_id", target.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("target resolution enqueue failed")
	}
	return &resolveTargetOutput{Body: job}, nil
}

func (a *API) modelManagerCertificateUploadHandler(ctx context.Context, input *modelManagerCertificateUploadInput) (*propertyDocumentAttachOutput, error) {
	file := input.RawBody.Data().File
	if !file.IsSet {
		return nil, huma.Error400BadRequest("manager certificate PDF is required")
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, huma.Error400BadRequest("read manager certificate PDF failed")
	}
	document, err := a.createPropertyDocument(ctx, file.Filename, file.ContentType, data)
	if err != nil {
		if errors.Is(err, properties.ErrPropertyDocumentTooLarge) {
			return nil, huma.Error413RequestEntityTooLarge("manager certificate PDF too large")
		}
		if errors.Is(err, properties.ErrPropertyDocumentInvalid) {
			return nil, huma.Error400BadRequest("invalid manager certificate PDF")
		}
		a.logger.ErrorContext(ctx, "manager certificate upload failed", "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate upload failed")
	}
	if strings.TrimSpace(input.TargetType) != "" || strings.TrimSpace(input.TargetID) != "" {
		target, err := parseCanonicalTarget(input.TargetType, input.TargetID)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		document, err = a.attachPropertyDocument(ctx, document.ID, &target)
		if err != nil {
			a.logger.ErrorContext(ctx, "manager certificate attach failed", "document_id", document.ID, "error", err, "outcome", logging.OutcomeError)
			return nil, huma.Error500InternalServerError("manager certificate attach failed")
		}
	}
	job, err := a.enqueueManagerCertificateExtraction(ctx, document, "")
	if err != nil {
		a.logger.ErrorContext(ctx, "manager certificate extraction enqueue failed", "document_id", document.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate extraction enqueue failed")
	}
	out := &propertyDocumentAttachOutput{}
	out.Body.Document = document
	out.Body.Job = QueuedCanonicalJob{JobID: job.JobID, Queued: job.Queued}
	return out, nil
}

func (a *API) propertyDocumentSummaryHandler(ctx context.Context, input *modelPropertyDocumentInput) (*propertyDocumentSummaryOutput, error) {
	document, err := a.propertiesService.PropertyDocumentSummary(ctx, input.ID)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("property document not found")
		}
		a.logger.ErrorContext(ctx, "property document summary failed", "document_id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("property document summary failed")
	}
	return &propertyDocumentSummaryOutput{Body: document}, nil
}

func (a *API) propertyDocumentAttachModelHandler(ctx context.Context, input *propertyDocumentAttachModelInput) (*propertyDocumentAttachOutput, error) {
	document, err := a.attachPropertyDocument(ctx, input.ID, input.Body.Target)
	if err != nil {
		if errors.Is(err, properties.ErrNotFound) {
			return nil, huma.Error404NotFound("property document not found")
		}
		a.logger.ErrorContext(ctx, "property document attach failed", "document_id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("property document attach failed")
	}
	job, err := a.enqueueManagerCertificateProjection(ctx, document)
	if err != nil {
		a.logger.ErrorContext(ctx, "manager certificate projection enqueue failed", "document_id", input.ID, "error", err, "outcome", logging.OutcomeError)
		return nil, huma.Error500InternalServerError("manager certificate projection enqueue failed")
	}
	out := &propertyDocumentAttachOutput{}
	out.Body.Document = document
	out.Body.Job = QueuedCanonicalJob{JobID: job.JobID, Queued: job.Queued}
	return out, nil
}

func (a *API) listPropertyTargetMapMarkers(ctx context.Context, input *propertyTargetsMapInput) ([]PropertyTargetMapMarker, error) {
	limit := input.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	minLat, minLng, maxLat, maxLng := nullableBounds(input.MinLat, input.MinLng, input.MaxLat, input.MaxLng)
	query := strings.TrimSpace(input.Query)
	rows, err := a.pool.Query(ctx, `
WITH building_markers AS (
    SELECT
        'building'::text AS target_type,
        pb.physical_building_id AS target_id,
        hc.housing_company_id,
        COALESCE(pb.physical_building_address_norm, hc.housing_company_address_norm, '') AS address,
        COALESCE(pb.physical_building_city_norm, hc.housing_company_city_norm, '') AS city,
        COALESCE(pb.physical_building_postal_norm, hc.housing_company_postal_norm, '') AS postal,
        COALESCE(pb.physical_building_build_year, hc.housing_company_build_year) AS build_year,
        pb.physical_building_latitude AS lat,
        pb.physical_building_longitude AS lng,
        1::bigint AS building_count,
        COALESCE(counts.unit_count, 0) AS unit_count,
        COALESCE(counts.offering_count, 0) AS offering_count,
        COALESCE(source_counts.source_count, 0) AS source_count,
        COALESCE(document_counts.document_count, 0) AS document_count,
        COALESCE(hc.housing_company_name, '') AS housing_company_name,
        max(po.property_offering_last_seen_at) AS last_seen_at
    FROM public.physical_buildings pb
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pb.housing_company_id
    LEFT JOIN LATERAL (
        SELECT count(DISTINCT pu.property_unit_id)::bigint AS unit_count, count(DISTINCT po.property_offering_id)::bigint AS offering_count
        FROM public.property_units pu
        LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
        WHERE pu.physical_building_id = pb.physical_building_id
    ) counts ON true
    LEFT JOIN LATERAL (
        SELECT count(*)::bigint AS source_count
        FROM public.property_target_sources pts
        WHERE pts.target_type = 'building'
            AND pts.target_id = pb.physical_building_id
            AND pts.link_status <> 'rejected'
    ) source_counts ON true
    LEFT JOIN LATERAL (
        SELECT count(*)::bigint AS document_count
        FROM public.property_documents pd
        WHERE pd.physical_building_id = pb.physical_building_id
    ) document_counts ON true
    LEFT JOIN public.property_units pu ON pu.physical_building_id = pb.physical_building_id
    LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    WHERE pb.physical_building_latitude IS NOT NULL
        AND pb.physical_building_longitude IS NOT NULL
        AND ($1::double precision IS NULL OR (
            pb.physical_building_latitude BETWEEN $1::double precision AND $3::double precision
            AND pb.physical_building_longitude BETWEEN $2::double precision AND $4::double precision
        ))
        AND (
            $6::text = ''
            OR lower(concat_ws(' ', hc.housing_company_name, hc.housing_company_business_id, pb.physical_building_address_norm, pb.physical_building_city_norm, pb.physical_building_postal_norm, hc.housing_company_address_norm, hc.housing_company_city_norm, hc.housing_company_postal_norm)) LIKE ('%' || lower($6::text) || '%')
            OR EXISTS (
                SELECT 1
                FROM public.property_target_sources pts
                WHERE pts.target_type = 'building'
                    AND pts.target_id = pb.physical_building_id
                    AND pts.link_status <> 'rejected'
                    AND lower(concat_ws(' ', pts.source_id_value, pts.source_external_id, pts.source_url)) LIKE ('%' || lower($6::text) || '%')
            )
        )
    GROUP BY pb.physical_building_id, hc.housing_company_id, counts.unit_count, counts.offering_count, source_counts.source_count, document_counts.document_count
),
company_markers AS (
    SELECT
        'housing_company'::text AS target_type,
        hc.housing_company_id AS target_id,
        hc.housing_company_id,
        COALESCE(hc.housing_company_address_norm, '') AS address,
        COALESCE(hc.housing_company_city_norm, '') AS city,
        COALESCE(hc.housing_company_postal_norm, '') AS postal,
        hc.housing_company_build_year AS build_year,
        postgis.ST_Y(hc.housing_company_geom)::double precision AS lat,
        postgis.ST_X(hc.housing_company_geom)::double precision AS lng,
        COALESCE(counts.building_count, 0) AS building_count,
        COALESCE(counts.unit_count, 0) AS unit_count,
        COALESCE(counts.offering_count, 0) AS offering_count,
        COALESCE(source_counts.source_count, 0) AS source_count,
        COALESCE(document_counts.document_count, 0) AS document_count,
        COALESCE(hc.housing_company_name, '') AS housing_company_name,
        counts.last_seen_at
    FROM public.housing_companies hc
    LEFT JOIN LATERAL (
        SELECT count(DISTINCT pb.physical_building_id)::bigint AS building_count, count(DISTINCT pu.property_unit_id)::bigint AS unit_count, count(DISTINCT po.property_offering_id)::bigint AS offering_count, max(po.property_offering_last_seen_at) AS last_seen_at
        FROM public.property_units pu
        LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
        LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
        WHERE pu.housing_company_id = hc.housing_company_id
    ) counts ON true
    LEFT JOIN LATERAL (
        SELECT count(*)::bigint AS source_count
        FROM public.property_target_sources pts
        WHERE pts.target_type = 'housing_company'
            AND pts.target_id = hc.housing_company_id
            AND pts.link_status <> 'rejected'
    ) source_counts ON true
    LEFT JOIN LATERAL (
        SELECT count(*)::bigint AS document_count
        FROM public.property_documents pd
        WHERE pd.housing_company_id = hc.housing_company_id
    ) document_counts ON true
    WHERE hc.housing_company_geom IS NOT NULL
        AND (
            $1::double precision IS NULL
            OR postgis.ST_Intersects(
                hc.housing_company_geom,
                postgis.ST_MakeEnvelope($2::double precision, $1::double precision, $4::double precision, $3::double precision, 4326)
            )
        )
        AND (
            $6::text = ''
            OR lower(concat_ws(' ', hc.housing_company_name, hc.housing_company_business_id, hc.housing_company_address_norm, hc.housing_company_city_norm, hc.housing_company_postal_norm)) LIKE ('%' || lower($6::text) || '%')
            OR EXISTS (
                SELECT 1
                FROM public.property_target_sources pts
                WHERE pts.target_type = 'housing_company'
                    AND pts.target_id = hc.housing_company_id
                    AND pts.link_status <> 'rejected'
                    AND lower(concat_ws(' ', pts.source_id_value, pts.source_external_id, pts.source_url)) LIKE ('%' || lower($6::text) || '%')
            )
        )
        AND NOT EXISTS (
            SELECT 1
            FROM public.physical_buildings pb
            WHERE pb.housing_company_id = hc.housing_company_id
                AND pb.physical_building_latitude IS NOT NULL
                AND pb.physical_building_longitude IS NOT NULL
        )
),
visible AS (
    SELECT * FROM building_markers
    UNION ALL
    SELECT * FROM company_markers
)
SELECT
    visible.target_type,
    visible.target_id,
    visible.housing_company_id,
    visible.housing_company_name,
    visible.address,
    visible.city,
    visible.postal,
    visible.build_year,
    visible.lat,
    visible.lng,
    visible.building_count,
    visible.unit_count,
    visible.offering_count,
    visible.source_count,
    visible.document_count
FROM visible
ORDER BY visible.last_seen_at DESC NULLS LAST, visible.offering_count DESC, visible.housing_company_name, visible.address
LIMIT $5::int`, minLat, minLng, maxLat, maxLng, limit, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	markers := []PropertyTargetMapMarker{}
	for rows.Next() {
		var marker PropertyTargetMapMarker
		var targetID, housingCompanyID uuid.UUID
		var targetType string
		if err := rows.Scan(&targetType, &targetID, &housingCompanyID, &marker.Name, &marker.Address, &marker.City, &marker.Postal, &marker.BuildYear, &marker.Lat, &marker.Lng, &marker.BuildingCount, &marker.UnitCount, &marker.OfferingCount, &marker.SourceCount, &marker.DocumentCount); err != nil {
			return nil, err
		}
		marker.Target = CanonicalTargetRef{Type: targetType, ID: targetID.String()}
		marker.Title = firstNonEmpty(marker.Name, marker.Address, targetID.String())
		if targetType == "building" {
			marker.FallbackTarget = &CanonicalTargetRef{Type: "housing_company", ID: housingCompanyID.String()}
		}
		marker.Offerings = []PropertyTargetMapOffering{}
		markers = append(markers, marker)
	}
	return markers, rows.Err()
}

func nullableBounds(minLat float64, minLng float64, maxLat float64, maxLng float64) (any, any, any, any) {
	if minLat == 0 || minLng == 0 || maxLat == 0 || maxLng == 0 {
		return nil, nil, nil, nil
	}
	return minLat, minLng, maxLat, maxLng
}

func decodeMapOfferings(data []byte, out *[]PropertyTargetMapOffering) error {
	var rows []struct {
		OfferingID string     `json:"offering_id"`
		UnitID     string     `json:"unit_id"`
		Headline   string     `json:"headline"`
		RoomLayout string     `json:"room_layout"`
		AreaM2     *float64   `json:"area_m2"`
		PriceEUR   *int64     `json:"price_eur"`
		LastSeenAt *time.Time `json:"last_seen_at"`
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		return err
	}
	offerings := make([]PropertyTargetMapOffering, 0, len(rows))
	for _, row := range rows {
		offerings = append(offerings, PropertyTargetMapOffering{
			Target: CanonicalTargetRef{
				Type: "offering",
				ID:   row.OfferingID,
			},
			UnitTarget: CanonicalTargetRef{
				Type: "unit",
				ID:   row.UnitID,
			},
			Headline:   row.Headline,
			RoomLayout: row.RoomLayout,
			AreaM2:     row.AreaM2,
			PriceEUR:   row.PriceEUR,
			LastSeenAt: row.LastSeenAt,
		})
	}
	*out = offerings
	return nil
}

func (a *API) getTargetOverview(ctx context.Context, target CanonicalTargetRef) (*TargetOverview, error) {
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, err
	}
	switch target.Type {
	case "offering":
		return a.getOfferingOverview(ctx, targetID)
	case "unit":
		return a.getUnitOverview(ctx, targetID)
	case "building":
		return a.getBuildingOverview(ctx, targetID)
	case "housing_company":
		return a.getHousingCompanyOverview(ctx, targetID)
	default:
		return nil, nil
	}
}

func (a *API) getOfferingOverview(ctx context.Context, id uuid.UUID) (*TargetOverview, error) {
	var offeringID, unitID uuid.UUID
	var housingCompanyID *uuid.UUID
	var headline, roomLayout, companyName, address, city, postal string
	var askingPrice, debtFreePrice *int64
	var areaM2 *float64
	var lastSeenAt *time.Time
	err := a.pool.QueryRow(ctx, `
SELECT
    po.property_offering_id,
    po.property_offering_headline,
    po.property_offering_asking_price,
    po.property_offering_debt_free_price,
    po.property_offering_last_seen_at,
    pu.property_unit_id,
    COALESCE(pu.property_unit_room_layout, ''),
    pu.property_unit_area_value,
    pu.housing_company_id,
    COALESCE(hc.housing_company_name, ''),
    COALESCE(hc.housing_company_address_norm, ''),
    COALESCE(hc.housing_company_city_norm, ''),
    COALESCE(hc.housing_company_postal_norm, '')
FROM public.property_offerings po
JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
WHERE po.property_offering_id = $1`, id).Scan(&offeringID, &headline, &askingPrice, &debtFreePrice, &lastSeenAt, &unitID, &roomLayout, &areaM2, &housingCompanyID, &companyName, &address, &city, &postal)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	overview := &TargetOverview{
		Title:    firstNonEmpty(headline, roomLayout, "Offering"),
		Subtitle: strings.TrimSpace(strings.Join(nonEmpty(address, postal, city), " ")),
		Fields: []TargetOverviewField{
			{Label: "Asking price", Value: formatOptionalInt(askingPrice, " EUR")},
			{Label: "Debt-free price", Value: formatOptionalInt(debtFreePrice, " EUR")},
			{Label: "Area", Value: formatOptionalFloat(areaM2, " m2")},
			{Label: "Layout", Value: roomLayout},
			{Label: "Last seen", Value: formatOptionalTime(lastSeenAt)},
		},
		Related: []TargetOverviewRelated{
			{Label: "Unit", Title: firstNonEmpty(roomLayout, unitID.String()), Target: CanonicalTargetRef{Type: "unit", ID: unitID.String()}},
		},
	}
	if housingCompanyID != nil {
		overview.Related = append(overview.Related, TargetOverviewRelated{Label: "Housing company", Title: firstNonEmpty(companyName, address, housingCompanyID.String()), Target: CanonicalTargetRef{Type: "housing_company", ID: housingCompanyID.String()}})
	}
	if err := a.appendTargetSourceLinks(ctx, overview, CanonicalTargetRef{Type: "offering", ID: id.String()}); err != nil {
		return nil, err
	}
	return overview, nil
}

func (a *API) getUnitOverview(ctx context.Context, id uuid.UUID) (*TargetOverview, error) {
	var unitID uuid.UUID
	var housingCompanyID *uuid.UUID
	var roomLayout, address, floorLevel, companyName, city, postal string
	var areaM2 *float64
	err := a.pool.QueryRow(ctx, `
SELECT
    pu.property_unit_id,
    COALESCE(pu.property_unit_room_layout, ''),
    COALESCE(pu.property_unit_address_norm, ''),
    COALESCE(pu.property_unit_floor_level, ''),
    pu.property_unit_area_value,
    pu.housing_company_id,
    COALESCE(hc.housing_company_name, ''),
    COALESCE(hc.housing_company_city_norm, ''),
    COALESCE(hc.housing_company_postal_norm, '')
FROM public.property_units pu
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
WHERE pu.property_unit_id = $1`, id).Scan(&unitID, &roomLayout, &address, &floorLevel, &areaM2, &housingCompanyID, &companyName, &city, &postal)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	overview := &TargetOverview{
		Title:    firstNonEmpty(roomLayout, address, "Unit"),
		Subtitle: strings.TrimSpace(strings.Join(nonEmpty(address, postal, city), " ")),
		Fields: []TargetOverviewField{
			{Label: "Area", Value: formatOptionalFloat(areaM2, " m2")},
			{Label: "Layout", Value: roomLayout},
			{Label: "Floor", Value: floorLevel},
		},
	}
	if housingCompanyID != nil {
		overview.Related = []TargetOverviewRelated{{Label: "Housing company", Title: firstNonEmpty(companyName, address, housingCompanyID.String()), Target: CanonicalTargetRef{Type: "housing_company", ID: housingCompanyID.String()}}}
	}
	return overview, nil
}

func (a *API) getBuildingOverview(ctx context.Context, id uuid.UUID) (*TargetOverview, error) {
	var buildingID uuid.UUID
	var housingCompanyID *uuid.UUID
	var address, city, postal, companyName string
	var buildYear, floorCount, apartmentCount *int32
	var lat, lng *float64
	err := a.pool.QueryRow(ctx, `
SELECT
    pb.physical_building_id,
    pb.housing_company_id,
    COALESCE(pb.physical_building_address_norm, hc.housing_company_address_norm, ''),
    COALESCE(pb.physical_building_city_norm, hc.housing_company_city_norm, ''),
    COALESCE(pb.physical_building_postal_norm, hc.housing_company_postal_norm, ''),
    COALESCE(hc.housing_company_name, ''),
    pb.physical_building_build_year,
    pb.physical_building_floor_count,
    pb.physical_building_apartment_count,
    pb.physical_building_latitude,
    pb.physical_building_longitude
FROM public.physical_buildings pb
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pb.housing_company_id
WHERE pb.physical_building_id = $1`, id).Scan(&buildingID, &housingCompanyID, &address, &city, &postal, &companyName, &buildYear, &floorCount, &apartmentCount, &lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	overview := &TargetOverview{
		Title:    firstNonEmpty(companyName, address, "Building"),
		Subtitle: strings.TrimSpace(strings.Join(nonEmpty(address, postal, city), " ")),
		Fields: []TargetOverviewField{
			{Label: "Build year", Value: formatOptionalInt32(buildYear, "")},
			{Label: "Floors", Value: formatOptionalInt32(floorCount, "")},
			{Label: "Apartments", Value: formatOptionalInt32(apartmentCount, "")},
			{Label: "Latitude", Value: formatOptionalFloat(lat, "")},
			{Label: "Longitude", Value: formatOptionalFloat(lng, "")},
		},
	}
	if housingCompanyID != nil {
		overview.Related = []TargetOverviewRelated{{Label: "Housing company", Title: firstNonEmpty(companyName, address, housingCompanyID.String()), Target: CanonicalTargetRef{Type: "housing_company", ID: housingCompanyID.String()}}}
	}
	if err := a.appendTargetSourceLinks(ctx, overview, CanonicalTargetRef{Type: "building", ID: id.String()}); err != nil {
		return nil, err
	}
	if err := a.appendBuildingCanonicalOfferings(ctx, overview, id); err != nil {
		return nil, err
	}
	return overview, nil
}

func (a *API) getHousingCompanyOverview(ctx context.Context, id uuid.UUID) (*TargetOverview, error) {
	var companyID uuid.UUID
	var name, address, city, postal, businessID string
	var buildYear, floorCount, apartmentCount *int32
	err := a.pool.QueryRow(ctx, `
SELECT
    housing_company_id,
    COALESCE(housing_company_name, ''),
    COALESCE(housing_company_address_norm, ''),
    COALESCE(housing_company_city_norm, ''),
    COALESCE(housing_company_postal_norm, ''),
    COALESCE(housing_company_business_id, ''),
    housing_company_build_year,
    housing_company_floor_count,
    housing_company_apartment_count
FROM public.housing_companies
WHERE housing_company_id = $1`, id).Scan(&companyID, &name, &address, &city, &postal, &businessID, &buildYear, &floorCount, &apartmentCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	overview := &TargetOverview{
		Title:    firstNonEmpty(name, address, "Housing company"),
		Subtitle: strings.TrimSpace(strings.Join(nonEmpty(address, postal, city), " ")),
		Fields: []TargetOverviewField{
			{Label: "Business ID", Value: businessID},
			{Label: "Build year", Value: formatOptionalInt32(buildYear, "")},
			{Label: "Floors", Value: formatOptionalInt32(floorCount, "")},
			{Label: "Apartments", Value: formatOptionalInt32(apartmentCount, "")},
		},
	}
	if err := a.appendTargetSourceLinks(ctx, overview, CanonicalTargetRef{Type: "housing_company", ID: id.String()}); err != nil {
		return nil, err
	}
	if err := a.appendHousingCompanyCanonicalOfferings(ctx, overview, id); err != nil {
		return nil, err
	}
	return overview, nil
}

func (a *API) appendHousingCompanyCanonicalOfferings(ctx context.Context, overview *TargetOverview, housingCompanyID uuid.UUID) error {
	rows, err := a.pool.Query(ctx, `
SELECT
    po.property_offering_id,
    COALESCE(po.property_offering_headline, pu.property_unit_room_layout, pu.property_unit_address_norm, po.property_offering_id::text) AS title,
    COALESCE(pu.property_unit_room_layout, '') AS room_layout,
    pu.property_unit_area_value,
    po.property_offering_asking_price,
    po.property_offering_last_seen_at
FROM public.property_units pu
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
WHERE pu.housing_company_id = $1
ORDER BY po.property_offering_last_seen_at DESC NULLS LAST, po.property_offering_asking_price ASC NULLS LAST, title
LIMIT 200`, housingCompanyID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var offeringID uuid.UUID
		var title, roomLayout string
		var areaM2 *float64
		var askingPrice *int64
		var lastSeenAt *time.Time
		if err := rows.Scan(&offeringID, &title, &roomLayout, &areaM2, &askingPrice, &lastSeenAt); err != nil {
			return err
		}
		details := strings.Join(nonEmpty(roomLayout, formatOptionalFloat(areaM2, " m2"), formatOptionalInt(askingPrice, " EUR"), formatOptionalTime(lastSeenAt)), " / ")
		overview.Related = append(overview.Related, TargetOverviewRelated{
			Label:  "Canonical listing",
			Title:  firstNonEmpty(title, details, offeringID.String()),
			Target: CanonicalTargetRef{Type: "offering", ID: offeringID.String()},
		})
	}
	return rows.Err()
}

func (a *API) appendBuildingCanonicalOfferings(ctx context.Context, overview *TargetOverview, buildingID uuid.UUID) error {
	rows, err := a.pool.Query(ctx, `
SELECT
    po.property_offering_id,
    COALESCE(po.property_offering_headline, pu.property_unit_room_layout, pu.property_unit_address_norm, po.property_offering_id::text) AS title,
    COALESCE(pu.property_unit_room_layout, '') AS room_layout,
    pu.property_unit_area_value,
    po.property_offering_asking_price,
    po.property_offering_last_seen_at
FROM public.property_units pu
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
WHERE pu.physical_building_id = $1
ORDER BY po.property_offering_last_seen_at DESC NULLS LAST, po.property_offering_asking_price ASC NULLS LAST, title
LIMIT 200`, buildingID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var offeringID uuid.UUID
		var title, roomLayout string
		var areaM2 *float64
		var askingPrice *int64
		var lastSeenAt *time.Time
		if err := rows.Scan(&offeringID, &title, &roomLayout, &areaM2, &askingPrice, &lastSeenAt); err != nil {
			return err
		}
		details := strings.Join(nonEmpty(roomLayout, formatOptionalFloat(areaM2, " m2"), formatOptionalInt(askingPrice, " EUR"), formatOptionalTime(lastSeenAt)), " / ")
		overview.Related = append(overview.Related, TargetOverviewRelated{
			Label:  "Canonical listing",
			Title:  firstNonEmpty(title, details, offeringID.String()),
			Target: CanonicalTargetRef{Type: "offering", ID: offeringID.String()},
		})
	}
	return rows.Err()
}

func (a *API) appendTargetSourceLinks(ctx context.Context, overview *TargetOverview, target CanonicalTargetRef) error {
	links, err := a.listTargetSources(ctx, target)
	if err != nil {
		return err
	}
	overview.Sources = append(overview.Sources, links...)
	return nil
}

func (a *API) appendOfferingSourceLinks(ctx context.Context, overview *TargetOverview, offeringID uuid.UUID) error {
	rows, err := a.pool.Query(ctx, `
WITH linked_listings AS (
    SELECT sl.*
    FROM public.property_offering_sources pos
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    WHERE pos.property_offering_id = $1
        AND pos.property_offering_source_link_status <> 'rejected'
),
source_links AS (
    SELECT
        'Listing'::text AS label,
        sl.sale_listing_source_provider AS provider,
        sl.sale_listing_source_kind AS kind,
        sl.sale_listing_native_id AS source_id,
        COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS title,
        COALESCE(sl.sale_listing_url, '') AS url,
        sl.sale_listing_last_seen_at AS last_seen_at
    FROM linked_listings sl
    UNION ALL
    SELECT DISTINCT
        'Building page'::text AS label,
        'shortcut'::text AS provider,
        'building'::text AS kind,
        sb.shortcut_building_external_id::text AS source_id,
        COALESCE(sb.shortcut_building_housing_company, sb.shortcut_building_address, sb.shortcut_building_external_id::text) AS title,
        sb.shortcut_building_url AS url,
        COALESCE(sb.shortcut_building_updated_at, sb.shortcut_building_processed_at) AS last_seen_at
    FROM linked_listings sl
    JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    UNION ALL
    SELECT DISTINCT
        'Building page'::text AS label,
        'frontdoor'::text AS provider,
        'building'::text AS kind,
        fb.frontdoor_building_id::text AS source_id,
        COALESCE(fb.frontdoor_building_company_name, concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number), fb.frontdoor_building_id::text) AS title,
        fb.frontdoor_building_url AS url,
        fb.frontdoor_building_last_seen_at AS last_seen_at
    FROM linked_listings sl
    JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
)
SELECT label, provider, kind, COALESCE(source_id, ''), COALESCE(title, ''), COALESCE(url, ''), last_seen_at
FROM source_links
ORDER BY label, provider, last_seen_at DESC NULLS LAST, title`, offeringID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var link TargetSourceLink
		if err := rows.Scan(&link.Label, &link.Provider, &link.Kind, &link.SourceID, &link.Title, &link.URL, &link.LastSeenAt); err != nil {
			return err
		}
		overview.Sources = append(overview.Sources, link)
	}
	return rows.Err()
}

func (a *API) appendHousingCompanySourceLinks(ctx context.Context, overview *TargetOverview, housingCompanyID uuid.UUID) error {
	rows, err := a.pool.Query(ctx, `
WITH linked_listings AS (
    SELECT DISTINCT sl.*
    FROM public.property_units pu
    JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    WHERE pu.housing_company_id = $1
),
source_links AS (
    SELECT
        'Listing'::text AS label,
        sl.sale_listing_source_provider AS provider,
        sl.sale_listing_source_kind AS kind,
        sl.sale_listing_native_id AS source_id,
        COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS title,
        COALESCE(sl.sale_listing_url, '') AS url,
        sl.sale_listing_last_seen_at AS last_seen_at
    FROM linked_listings sl
    UNION ALL
    SELECT DISTINCT
        'Building page'::text AS label,
        'shortcut'::text AS provider,
        'building'::text AS kind,
        sb.shortcut_building_external_id::text AS source_id,
        COALESCE(sb.shortcut_building_housing_company, sb.shortcut_building_address, sb.shortcut_building_external_id::text) AS title,
        sb.shortcut_building_url AS url,
        COALESCE(sb.shortcut_building_updated_at, sb.shortcut_building_processed_at) AS last_seen_at
    FROM linked_listings sl
    JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    UNION ALL
    SELECT DISTINCT
        'Building page'::text AS label,
        'frontdoor'::text AS provider,
        'building'::text AS kind,
        fb.frontdoor_building_id::text AS source_id,
        COALESCE(fb.frontdoor_building_company_name, concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number), fb.frontdoor_building_id::text) AS title,
        fb.frontdoor_building_url AS url,
        fb.frontdoor_building_last_seen_at AS last_seen_at
    FROM linked_listings sl
    JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
)
SELECT label, provider, kind, COALESCE(source_id, ''), COALESCE(title, ''), COALESCE(url, ''), last_seen_at
FROM source_links
ORDER BY label, provider, last_seen_at DESC NULLS LAST, title
LIMIT 500`, housingCompanyID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var link TargetSourceLink
		if err := rows.Scan(&link.Label, &link.Provider, &link.Kind, &link.SourceID, &link.Title, &link.URL, &link.LastSeenAt); err != nil {
			return err
		}
		overview.Sources = append(overview.Sources, link)
	}
	return rows.Err()
}

func parseCanonicalTarget(targetType string, targetID string) (CanonicalTargetRef, error) {
	targetType = strings.TrimSpace(targetType)
	switch targetType {
	case "offering", "unit", "building", "housing_company", "listing", "document", "transaction":
	default:
		return CanonicalTargetRef{}, fmt.Errorf("unsupported target_type %q", targetType)
	}
	id, err := uuid.Parse(strings.TrimSpace(targetID))
	if err != nil {
		return CanonicalTargetRef{}, fmt.Errorf("target_id must be a UUID")
	}
	return CanonicalTargetRef{Type: targetType, ID: id.String()}, nil
}

func (a *API) listTargetSources(ctx context.Context, target CanonicalTargetRef) ([]TargetSourceLink, error) {
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx, `
SELECT
    source_provider,
    source_kind,
    source_table,
    COALESCE(source_id::text, ''),
    source_id_value,
    COALESCE(source_external_id, ''),
    COALESCE(source_url, ''),
    link_status,
    last_seen_at
FROM public.property_target_sources
WHERE target_type = $1
    AND target_id = $2
    AND link_status <> 'rejected'
ORDER BY source_kind, source_provider, last_seen_at DESC NULLS LAST, source_id_value
LIMIT 500`, target.Type, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TargetSourceLink{}
	for rows.Next() {
		var link TargetSourceLink
		if err := rows.Scan(&link.Provider, &link.Kind, &link.SourceTable, &link.SourceID, &link.SourceIDValue, &link.ExternalID, &link.URL, &link.LinkStatus, &link.LastSeenAt); err != nil {
			return nil, err
		}
		link.Label = sourceLinkLabel(link.Kind)
		link.Title = firstNonEmpty(link.ExternalID, link.SourceIDValue)
		out = append(out, link)
	}
	return out, rows.Err()
}

func (a *API) listTargetChildren(ctx context.Context, target CanonicalTargetRef) ([]TargetBuildingSummary, []TargetUnitSummary, []TargetOfferingSummary, error) {
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, nil, nil, err
	}
	switch target.Type {
	case "housing_company":
		buildings, err := a.listCompanyBuildings(ctx, targetID)
		if err != nil {
			return nil, nil, nil, err
		}
		units, err := a.listCompanyUnits(ctx, targetID)
		if err != nil {
			return nil, nil, nil, err
		}
		offerings, err := a.listCompanyOfferings(ctx, targetID)
		return buildings, units, offerings, err
	case "building":
		units, err := a.listBuildingUnits(ctx, targetID)
		if err != nil {
			return nil, nil, nil, err
		}
		offerings, err := a.listBuildingOfferings(ctx, targetID)
		return nil, units, offerings, err
	case "unit":
		offerings, err := a.listUnitOfferings(ctx, targetID)
		return nil, nil, offerings, err
	default:
		return nil, nil, nil, nil
	}
}

func (a *API) listCompanyBuildings(ctx context.Context, housingCompanyID uuid.UUID) ([]TargetBuildingSummary, error) {
	rows, err := a.pool.Query(ctx, `
SELECT
    pb.physical_building_id,
    pb.housing_company_id,
    COALESCE(pb.physical_building_address_norm, hc.housing_company_address_norm, ''),
    COALESCE(pb.physical_building_city_norm, hc.housing_company_city_norm, ''),
    COALESCE(pb.physical_building_postal_norm, hc.housing_company_postal_norm, ''),
    COALESCE(hc.housing_company_name, ''),
    pb.physical_building_build_year,
    pb.physical_building_latitude,
    pb.physical_building_longitude,
    count(DISTINCT pu.property_unit_id)::bigint,
    count(DISTINCT po.property_offering_id)::bigint
FROM public.physical_buildings pb
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pb.housing_company_id
LEFT JOIN public.property_units pu ON pu.physical_building_id = pb.physical_building_id
LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
WHERE pb.housing_company_id = $1
GROUP BY pb.physical_building_id, hc.housing_company_id
ORDER BY COALESCE(pb.physical_building_address_norm, hc.housing_company_address_norm, ''), pb.physical_building_id
LIMIT 500`, housingCompanyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TargetBuildingSummary{}
	for rows.Next() {
		var row TargetBuildingSummary
		var id uuid.UUID
		var housingID *uuid.UUID
		if err := rows.Scan(&id, &housingID, &row.Address, &row.City, &row.Postal, &row.Title, &row.BuildYear, &row.Lat, &row.Lng, &row.UnitCount, &row.OfferingCount); err != nil {
			return nil, err
		}
		row.Target = CanonicalTargetRef{Type: "building", ID: id.String()}
		row.Title = firstNonEmpty(row.Title, row.Address, id.String())
		if housingID != nil {
			row.HousingTarget = &CanonicalTargetRef{Type: "housing_company", ID: housingID.String()}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (a *API) listCompanyUnits(ctx context.Context, housingCompanyID uuid.UUID) ([]TargetUnitSummary, error) {
	return a.listUnits(ctx, "pu.housing_company_id = $1", housingCompanyID)
}

func (a *API) listBuildingUnits(ctx context.Context, buildingID uuid.UUID) ([]TargetUnitSummary, error) {
	return a.listUnits(ctx, "pu.physical_building_id = $1", buildingID)
}

func (a *API) listUnits(ctx context.Context, predicate string, id uuid.UUID) ([]TargetUnitSummary, error) {
	rows, err := a.pool.Query(ctx, fmt.Sprintf(`
SELECT
    pu.property_unit_id,
    pu.physical_building_id,
    pu.housing_company_id,
    COALESCE(pu.property_unit_address_norm, ''),
    COALESCE(pu.property_unit_room_layout, ''),
    pu.property_unit_area_value,
    COALESCE(pu.property_unit_floor_level::text, ''),
    count(po.property_offering_id)::bigint
FROM public.property_units pu
LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
WHERE %s
GROUP BY pu.property_unit_id
ORDER BY COALESCE(pu.property_unit_address_norm, ''), COALESCE(pu.property_unit_floor_level::text, ''), pu.property_unit_id
LIMIT 500`, predicate), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TargetUnitSummary{}
	for rows.Next() {
		var row TargetUnitSummary
		var id, housingID uuid.UUID
		var buildingID *uuid.UUID
		if err := rows.Scan(&id, &buildingID, &housingID, &row.Address, &row.Layout, &row.AreaM2, &row.Floor, &row.OfferingCount); err != nil {
			return nil, err
		}
		row.Target = CanonicalTargetRef{Type: "unit", ID: id.String()}
		row.HousingTarget = &CanonicalTargetRef{Type: "housing_company", ID: housingID.String()}
		row.Title = firstNonEmpty(row.Layout, row.Address, id.String())
		if buildingID != nil {
			row.BuildingTarget = &CanonicalTargetRef{Type: "building", ID: buildingID.String()}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (a *API) listCompanyOfferings(ctx context.Context, housingCompanyID uuid.UUID) ([]TargetOfferingSummary, error) {
	return a.listOfferings(ctx, "pu.housing_company_id = $1", housingCompanyID)
}

func (a *API) listBuildingOfferings(ctx context.Context, buildingID uuid.UUID) ([]TargetOfferingSummary, error) {
	return a.listOfferings(ctx, "pu.physical_building_id = $1", buildingID)
}

func (a *API) listUnitOfferings(ctx context.Context, unitID uuid.UUID) ([]TargetOfferingSummary, error) {
	return a.listOfferings(ctx, "pu.property_unit_id = $1", unitID)
}

func (a *API) listOfferings(ctx context.Context, predicate string, id uuid.UUID) ([]TargetOfferingSummary, error) {
	rows, err := a.pool.Query(ctx, fmt.Sprintf(`
SELECT
    po.property_offering_id,
    pu.property_unit_id,
    pu.physical_building_id,
    pu.housing_company_id,
    COALESCE(po.property_offering_headline, pu.property_unit_room_layout, pu.property_unit_address_norm, po.property_offering_id::text),
    COALESCE(pu.property_unit_room_layout, ''),
    pu.property_unit_area_value,
    po.property_offering_asking_price,
    po.property_offering_last_seen_at
FROM public.property_units pu
JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
WHERE %s
ORDER BY po.property_offering_last_seen_at DESC NULLS LAST, po.property_offering_asking_price ASC NULLS LAST, po.property_offering_id
LIMIT 500`, predicate), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []TargetOfferingSummary{}
	for rows.Next() {
		var row TargetOfferingSummary
		var id, unitID, housingID uuid.UUID
		var buildingID *uuid.UUID
		if err := rows.Scan(&id, &unitID, &buildingID, &housingID, &row.Title, &row.Layout, &row.AreaM2, &row.AskingPriceEUR, &row.LastSeenAt); err != nil {
			return nil, err
		}
		row.Target = CanonicalTargetRef{Type: "offering", ID: id.String()}
		row.UnitTarget = CanonicalTargetRef{Type: "unit", ID: unitID.String()}
		row.HousingTarget = &CanonicalTargetRef{Type: "housing_company", ID: housingID.String()}
		if buildingID != nil {
			row.BuildingTarget = &CanonicalTargetRef{Type: "building", ID: buildingID.String()}
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (a *API) listResolvedValues(ctx context.Context, target CanonicalTargetRef) ([]ResolvedValue, error) {
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx, `
SELECT
    value.target_type,
    value.target_id,
    value.dimension_key,
    value.value,
    value.value_kind,
    COALESCE(value.unit, ''),
    value.confidence,
    value.selected_reason,
    value.conflict_status,
    value.supporting_claim_ids,
    value.rejected_claim_ids,
    value.resolved_at,
    claim.property_dimension_claim_id,
    claim.claim_scope,
    claim.source_table,
    claim.source_id,
    COALESCE(claim.source_field, ''),
    claim.projection_version,
    claim.source_observed_at,
    claim.created_at,
    claim.confidence,
    claim.source_reliability,
    claim.evidence
FROM public.property_dimension_values value
LEFT JOIN public.property_dimension_claims claim ON claim.property_dimension_claim_id = value.selected_claim_id
WHERE value.target_type = $1
    AND value.target_id = $2
ORDER BY value.dimension_key`, target.Type, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ResolvedValue{}
	for rows.Next() {
		var value ResolvedValue
		var targetID uuid.UUID
		var selectedID *uuid.UUID
		var selectedScope, selectedTable, selectedSourceField, selectedProjectionVersion string
		var selectedSourceID *uuid.UUID
		var selectedObservedAt, selectedCreatedAt *time.Time
		var selectedConfidence, selectedReliability *float64
		var selectedEvidence json.RawMessage
		var supportingIDs, rejectedIDs []uuid.UUID
		if err := rows.Scan(&value.Target.Type, &targetID, &value.DimensionKey, &value.Value, &value.ValueKind, &value.Unit, &value.Confidence, &value.SelectedReason, &value.ConflictStatus, &supportingIDs, &rejectedIDs, &value.ResolvedAt, &selectedID, &selectedScope, &selectedTable, &selectedSourceID, &selectedSourceField, &selectedProjectionVersion, &selectedObservedAt, &selectedCreatedAt, &selectedConfidence, &selectedReliability, &selectedEvidence); err != nil {
			return nil, err
		}
		value.Target.ID = targetID.String()
		value.SupportingClaimIDs = uuidStrings(supportingIDs)
		value.RejectedClaimIDs = uuidStrings(rejectedIDs)
		if selectedID != nil {
			sourceID := ""
			if selectedSourceID != nil {
				sourceID = selectedSourceID.String()
			}
			value.SelectedEvidence = &EvidenceRef{ID: selectedID.String(), Kind: "claim", Scope: selectedScope, SourceTable: selectedTable, SourceID: sourceID, SourceField: selectedSourceField, ProjectionVersion: selectedProjectionVersion, ObservedAt: selectedObservedAt, CreatedAt: selectedCreatedAt, Confidence: selectedConfidence, SourceReliability: selectedReliability, Evidence: selectedEvidence}
		}
		out = append(out, value)
	}
	return out, rows.Err()
}

func (a *API) listSourceClaims(ctx context.Context, target CanonicalTargetRef) ([]SourceClaim, error) {
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx, `
SELECT
    property_dimension_claim_id,
    target_type,
    target_id,
    dimension_key,
    value,
    value_kind,
    COALESCE(unit, ''),
    claim_scope,
    source_table,
    source_id,
    COALESCE(source_field, ''),
    projection_version,
    source_observed_at,
    valid_from::text,
    valid_until::text,
    confidence,
    source_reliability,
    evidence,
    created_at,
    updated_at
FROM public.property_dimension_claims
WHERE target_type = $1
    AND target_id = $2
ORDER BY dimension_key, claim_scope DESC, source_observed_at DESC NULLS LAST, created_at DESC`, target.Type, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SourceClaim{}
	for rows.Next() {
		var claim SourceClaim
		var id, targetID, sourceID uuid.UUID
		if err := rows.Scan(&id, &claim.Target.Type, &targetID, &claim.DimensionKey, &claim.Value, &claim.ValueKind, &claim.Unit, &claim.ClaimScope, &claim.SourceTable, &sourceID, &claim.SourceField, &claim.ProjectionVersion, &claim.ObservedAt, &claim.ValidFrom, &claim.ValidUntil, &claim.Confidence, &claim.SourceReliability, &claim.Evidence, &claim.CreatedAt, &claim.UpdatedAt); err != nil {
			return nil, err
		}
		claim.ID = id.String()
		claim.Target.ID = targetID.String()
		claim.SourceID = sourceID.String()
		out = append(out, claim)
	}
	return out, rows.Err()
}

func (a *API) listRenovationEvents(ctx context.Context, target CanonicalTargetRef) ([]RenovationEvent, error) {
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx, `
SELECT
    property_renovation_event_id,
    target_type,
    target_id,
    event_scope,
    source_table,
    source_id,
    COALESCE(source_field, ''),
    projection_version,
    category,
    COALESCE(component, ''),
    status,
    COALESCE(stage, ''),
    COALESCE(scope, ''),
    COALESCE(responsibility, ''),
    year,
    start_year,
    end_year,
    cost_estimate_eur,
    COALESCE(summary, ''),
    evidence,
    confidence,
    source_reliability,
    source_observed_at,
    created_at
FROM public.property_renovation_events
WHERE target_type = $1
    AND target_id = $2
ORDER BY category, status, year NULLS LAST, source_observed_at DESC NULLS LAST, created_at DESC`, target.Type, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []RenovationEvent{}
	for rows.Next() {
		var event RenovationEvent
		var id, targetID, sourceID uuid.UUID
		if err := rows.Scan(&id, &event.Target.Type, &targetID, &event.EventScope, &event.SourceTable, &sourceID, &event.SourceField, &event.ProjectionVersion, &event.Category, &event.Component, &event.Status, &event.Stage, &event.Scope, &event.Responsibility, &event.Year, &event.StartYear, &event.EndYear, &event.CostEstimateEUR, &event.Summary, &event.Evidence, &event.Confidence, &event.SourceReliability, &event.ObservedAt, &event.CreatedAt); err != nil {
			return nil, err
		}
		event.ID = id.String()
		event.Target.ID = targetID.String()
		event.SourceID = sourceID.String()
		out = append(out, event)
	}
	return out, rows.Err()
}

func (a *API) listTargetDocuments(ctx context.Context, target CanonicalTargetRef) ([]properties.PropertyDocumentSummary, error) {
	column, err := documentTargetColumn(target.Type)
	if err != nil {
		return []properties.PropertyDocumentSummary{}, nil
	}
	targetID, err := uuid.Parse(target.ID)
	if err != nil {
		return nil, err
	}
	rows, err := a.pool.Query(ctx, fmt.Sprintf(`
SELECT
    property_document_id,
    COALESCE(property_offering_id::text, ''),
    COALESCE(property_unit_id::text, ''),
    COALESCE(physical_building_id::text, ''),
    COALESCE(housing_company_id::text, ''),
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_extraction_status,
    COALESCE(property_document_extraction_error, ''),
    property_document_uploaded_at,
    property_document_extracted_at
FROM public.property_documents
WHERE %s = $1
ORDER BY property_document_uploaded_at DESC`, column), targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []properties.PropertyDocumentSummary{}
	for rows.Next() {
		var document properties.PropertyDocumentSummary
		var uploadedAt time.Time
		var extractedAt *time.Time
		if err := rows.Scan(&document.ID, &document.OfferingID, &document.UnitID, &document.PhysicalBuildingID, &document.HousingCompanyID, &document.Type, &document.Filename, &document.MimeType, &document.SizeBytes, &document.SHA256, &document.ExtractionStatus, &document.ExtractionError, &uploadedAt, &extractedAt); err != nil {
			return nil, err
		}
		document.UploadedAt = uploadedAt.Format(time.RFC3339)
		if extractedAt != nil {
			document.ExtractedAt = extractedAt.Format(time.RFC3339)
		}
		document.DownloadURL = "/api/v1/property-documents/" + document.ID + "/download"
		out = append(out, document)
	}
	return out, rows.Err()
}

func (a *API) createPropertyDocument(ctx context.Context, filename string, mimeType string, data []byte) (properties.PropertyDocumentSummary, error) {
	if len(data) == 0 || strings.TrimSpace(mimeType) != "application/pdf" {
		return properties.PropertyDocumentSummary{}, properties.ErrPropertyDocumentInvalid
	}
	if len(data) > 25*1024*1024 {
		return properties.PropertyDocumentSummary{}, properties.ErrPropertyDocumentTooLarge
	}
	hash := sha256.Sum256(data)
	var id uuid.UUID
	if err := a.pool.QueryRow(ctx, `
INSERT INTO public.property_documents (
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_bytes
) VALUES ('manager_certificate', $1, 'application/pdf', $2, $3, $4)
RETURNING property_document_id`, strings.TrimSpace(filename), int64(len(data)), hex.EncodeToString(hash[:]), data).Scan(&id); err != nil {
		return properties.PropertyDocumentSummary{}, err
	}
	return a.propertiesService.PropertyDocumentSummary(ctx, id.String())
}

func (a *API) attachPropertyDocument(ctx context.Context, documentID string, target *CanonicalTargetRef) (properties.PropertyDocumentSummary, error) {
	docID, err := uuid.Parse(strings.TrimSpace(documentID))
	if err != nil {
		return properties.PropertyDocumentSummary{}, properties.ErrNotFound
	}
	offeringID, unitID, buildingID, housingCompanyID := (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil), (*uuid.UUID)(nil)
	if target != nil {
		parsed, err := parseCanonicalTarget(target.Type, target.ID)
		if err != nil {
			return properties.PropertyDocumentSummary{}, err
		}
		parsedID, err := uuid.Parse(parsed.ID)
		if err != nil {
			return properties.PropertyDocumentSummary{}, err
		}
		switch parsed.Type {
		case "offering":
			offeringID = &parsedID
		case "unit":
			unitID = &parsedID
		case "building":
			buildingID = &parsedID
		case "housing_company":
			housingCompanyID = &parsedID
		default:
			return properties.PropertyDocumentSummary{}, fmt.Errorf("documents can attach only to offering, unit, building, or housing_company")
		}
	}
	tag, err := a.pool.Exec(ctx, `
UPDATE public.property_documents
SET property_offering_id = $2,
    property_unit_id = $3,
    physical_building_id = $4,
    housing_company_id = $5,
    property_document_updated_at = now()
WHERE property_document_id = $1`, docID, offeringID, unitID, buildingID, housingCompanyID)
	if err != nil {
		return properties.PropertyDocumentSummary{}, err
	}
	if tag.RowsAffected() == 0 {
		return properties.PropertyDocumentSummary{}, properties.ErrNotFound
	}
	return a.propertiesService.PropertyDocumentSummary(ctx, docID.String())
}

func (a *API) enqueueTargetResolution(ctx context.Context, target CanonicalTargetRef) (QueuedCanonicalJob, error) {
	payload, err := json.Marshal(map[string]string{"target_type": target.Type, "target_id": target.ID})
	if err != nil {
		return QueuedCanonicalJob{}, fmt.Errorf("marshal target resolution payload: %w", err)
	}
	store := syncjobs.NewStore(a.logger, a.pool)
	result, err := store.Enqueue(ctx, syncjobs.EnqueueRequest{Provider: "canonical", Kind: consumers.TaskTypeCanonicalResolveDimensionTarget, EntityID: target.Type + ":" + target.ID, Priority: int32(taskqueue.PriorityHigh), MaxAttempts: 3, Payload: payload})
	if err != nil {
		return QueuedCanonicalJob{}, err
	}
	return QueuedCanonicalJob{JobID: result.Job.SyncJobID.String(), Queued: result.Enqueued}, nil
}

func documentTargetColumn(targetType string) (string, error) {
	switch targetType {
	case "offering":
		return "property_offering_id", nil
	case "unit":
		return "property_unit_id", nil
	case "building":
		return "physical_building_id", nil
	case "housing_company":
		return "housing_company_id", nil
	default:
		return "", fmt.Errorf("unsupported document target type")
	}
}

func sourceLinkLabel(kind string) string {
	switch kind {
	case "building":
		return "Building page"
	case "ad", "announcement":
		return "Listing"
	default:
		return "Source"
	}
}

func uuidStrings(values []uuid.UUID) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.String())
	}
	return out
}

func nonEmpty(values ...string) []string {
	out := []string{}
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, strings.TrimSpace(value))
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func formatOptionalInt(value *int64, suffix string) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d%s", *value, suffix)
}

func formatOptionalInt32(value *int32, suffix string) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d%s", *value, suffix)
}

func formatOptionalFloat(value *float64, suffix string) string {
	if value == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", *value), "0"), ".") + suffix
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
