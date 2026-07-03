package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"

	"koditon/internal/domain/ads"
)

type entityDetailInput struct {
	ID string `query:"id" required:"true" doc:"Canonical ID or source URL"`
}

type detailFieldOutput struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type rawPayloadOutput struct {
	Pretty        string `json:"pretty"`
	OriginalBytes int    `json:"original_bytes"`
}

type entitySourceRecordOutput struct {
	ListingID            string                  `json:"listing_id"`
	CanonicalID          string                  `json:"canonical_id"`
	Source               string                  `json:"source"`
	Kind                 string                  `json:"kind"`
	NativeID             string                  `json:"native_id"`
	Headline             string                  `json:"headline,omitempty"`
	Address              string                  `json:"address,omitempty"`
	City                 string                  `json:"city,omitempty"`
	Postal               string                  `json:"postal,omitempty"`
	AskingPrice          *int64                  `json:"asking_price,omitempty"`
	Area                 *float64                `json:"area,omitempty"`
	URL                  string                  `json:"url,omitempty"`
	ExternalURLAvailable bool                    `json:"external_url_available"`
	LastSeenAt           *time.Time              `json:"last_seen_at,omitempty"`
	LinkStatus           string                  `json:"link_status,omitempty"`
	LinkMethod           string                  `json:"link_method,omitempty"`
	LinkScore            *int32                  `json:"link_score,omitempty"`
	PriceMatch           *entityPriceMatchOutput `json:"price_match,omitempty"`
	Insights             []entityInsightOutput   `json:"insights,omitempty"`
}

type entityPriceMatchOutput struct {
	TransactionID string     `json:"transaction_id"`
	Scope         string     `json:"scope,omitempty"`
	Status        string     `json:"status,omitempty"`
	Method        string     `json:"method,omitempty"`
	Score         *int32     `json:"score,omitempty"`
	PriceEUR      *int64     `json:"price_eur,omitempty"`
	Description   string     `json:"description,omitempty"`
	Type          string     `json:"type,omitempty"`
	Category      string     `json:"category,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`
}

type entityInsightOutput struct {
	Key         string  `json:"key"`
	Value       string  `json:"value"`
	Direction   string  `json:"direction,omitempty"`
	Severity    string  `json:"severity,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
	SourceField string  `json:"source_field,omitempty"`
	Text        string  `json:"text,omitempty"`
}

type entityDetailOutput struct {
	Body struct {
		// Canonical
		CanonicalID          string     `json:"canonical_id"`
		Source               string     `json:"source"`
		Kind                 string     `json:"kind"`
		NativeID             string     `json:"native_id"`
		Headline             string     `json:"headline"`
		URL                  string     `json:"url,omitempty"`
		ExternalURLAvailable bool       `json:"external_url_available"`
		LastSeenAt           *time.Time `json:"last_seen_at,omitempty"`

		// Location
		StreetAddress string `json:"street_address,omitempty"`
		City          string `json:"city,omitempty"`
		Postal        string `json:"postal,omitempty"`

		// Pricing
		AskingPrice         *int64   `json:"asking_price,omitempty"`
		DebtFreePrice       *int64   `json:"debt_free_price,omitempty"`
		DebtShareAmount     *int64   `json:"debt_share_amount,omitempty"`
		PricePerSquareMeter *float64 `json:"price_per_m2,omitempty"`

		// Property details
		AreaM2      *float64 `json:"area_m2,omitempty"`
		RoomLayout  string   `json:"room_layout,omitempty"`
		RoomsCount  *int32   `json:"rooms_count,omitempty"`
		FloorLevel  *int32   `json:"floor_level,omitempty"`
		TotalFloors *int32   `json:"total_floors,omitempty"`
		BuildYear   *int32   `json:"build_year,omitempty"`
		Condition   string   `json:"condition,omitempty"`
		EnergyClass string   `json:"energy_class,omitempty"`
		PlotType    string   `json:"plot_type,omitempty"`
		Elevator    *bool    `json:"elevator,omitempty"`
		Sauna       *bool    `json:"sauna,omitempty"`

		// Monthly charges
		MaintenanceChargeMonthly *float64 `json:"maintenance_charge_monthly,omitempty"`
		TotalChargeMonthly       *float64 `json:"total_charge_monthly,omitempty"`
		WaterCharge              *float64 `json:"water_charge,omitempty"`

		// Text sections
		DescriptionText        string `json:"description_text,omitempty"`
		AvailabilityText       string `json:"availability_text,omitempty"`
		RenovationsDoneText    string `json:"renovations_done_text,omitempty"`
		RenovationsPlannedText string `json:"renovations_planned_text,omitempty"`
		AdditionalInfoText     string `json:"additional_info_text,omitempty"`
		ChargesText            string `json:"charges_text,omitempty"`

		// Source-specific & related
		ListingID      string                     `json:"listing_id,omitempty"`
		OfferingID     string                     `json:"offering_id,omitempty"`
		SourceCount    int                        `json:"source_count,omitempty"`
		SourceRecords  []entitySourceRecordOutput `json:"source_records,omitempty"`
		PriceMatch     *entityPriceMatchOutput    `json:"price_match,omitempty"`
		Insights       []entityInsightOutput      `json:"insights,omitempty"`
		CanonicalExtra []detailFieldOutput        `json:"canonical_extra,omitempty"`
		SourceSpecific []detailFieldOutput        `json:"source_specific,omitempty"`
		Related        []detailFieldOutput        `json:"related,omitempty"`
		Raw            *rawPayloadOutput          `json:"raw,omitempty"`
	}
}

func (a *API) entityDetailHandler(ctx context.Context, input *entityDetailInput) (*entityDetailOutput, error) {
	canonicalID, err := ads.ResolveInput(input.ID, a.cfg.Shortcut.SitemapBase, a.cfg.Frontdoor.SitemapBase)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid ID or URL: " + input.ID)
	}

	detail, err := a.adsService.DetailByCanonicalID(ctx, canonicalID)
	if err != nil {
		if errors.Is(err, ads.ErrNotFound) {
			return nil, huma.Error404NotFound("entity not found")
		}
		a.logger.ErrorContext(ctx, "entity detail lookup failed", "canonical_id", canonicalID, "error", err)
		return nil, huma.Error500InternalServerError("failed to fetch entity detail")
	}

	n := detail.Normalized
	out := &entityDetailOutput{}
	b := &out.Body

	b.CanonicalID = detail.Canonical.CanonicalID
	b.Source = detail.Canonical.Source
	b.Kind = detail.Canonical.Kind
	b.NativeID = detail.Canonical.NativeID
	b.Headline = detail.Canonical.Headline
	b.URL = detail.Canonical.URL
	b.ExternalURLAvailable = detail.Canonical.ExternalURLAvailable
	if !detail.Canonical.LastSeenAt.IsZero() {
		t := detail.Canonical.LastSeenAt
		b.LastSeenAt = &t
	}

	b.StreetAddress = firstNonEmpty(n.StreetAddress, detail.Canonical.Address)
	b.City = firstNonEmpty(n.City, detail.Canonical.City)
	b.Postal = firstNonEmpty(n.Postal, detail.Canonical.Postal)

	b.AskingPrice = n.AskingPrice
	b.DebtFreePrice = n.DebtFreePrice
	b.DebtShareAmount = n.DebtShareAmount
	b.PricePerSquareMeter = n.PricePerSquareMeter

	b.AreaM2 = n.AreaM2
	b.RoomLayout = n.RoomLayout
	b.RoomsCount = n.RoomsCount
	b.FloorLevel = n.FloorLevel
	b.TotalFloors = n.TotalFloors
	b.BuildYear = n.BuildYear
	b.Condition = n.Condition
	b.EnergyClass = n.EnergyClass
	b.PlotType = n.PlotType
	b.Elevator = n.Elevator
	b.Sauna = n.Sauna

	b.MaintenanceChargeMonthly = n.MaintenanceChargeMonthly
	b.TotalChargeMonthly = n.TotalChargeMonthly
	b.WaterCharge = n.WaterCharge

	b.DescriptionText = n.DescriptionText
	b.AvailabilityText = n.AvailabilityText
	b.RenovationsDoneText = n.RenovationsDoneText
	b.RenovationsPlannedText = n.RenovationsPlannedText
	b.AdditionalInfoText = n.AdditionalInfoText
	b.ChargesText = n.ChargesText

	b.CanonicalExtra = toDetailFields(detail.CanonicalExtra)
	b.SourceSpecific = toDetailFields(detail.SourceSpecific)
	b.Related = toDetailFields(detail.Related)
	grouping, err := a.entityGrouping(ctx, canonicalID)
	if err != nil {
		a.logger.ErrorContext(ctx, "entity grouping lookup failed", "canonical_id", canonicalID, "error", err)
		return nil, huma.Error500InternalServerError("failed to fetch entity grouping")
	}
	b.ListingID = grouping.ListingID
	b.OfferingID = grouping.OfferingID
	b.SourceCount = grouping.SourceCount
	b.SourceRecords = grouping.SourceRecords
	b.PriceMatch = grouping.PriceMatch
	b.Insights = grouping.Insights
	if detail.Raw.Pretty != "" || detail.Raw.OriginalBytes > 0 {
		b.Raw = &rawPayloadOutput{Pretty: detail.Raw.Pretty, OriginalBytes: detail.Raw.OriginalBytes}
	}

	return out, nil
}

type entityGroupingOutput struct {
	ListingID     string
	OfferingID    string
	SourceCount   int
	SourceRecords []entitySourceRecordOutput
	PriceMatch    *entityPriceMatchOutput
	Insights      []entityInsightOutput
}

func (a *API) entityGrouping(ctx context.Context, canonicalID string) (entityGroupingOutput, error) {
	var listingID uuid.UUID
	var offeringID uuid.UUID
	var primarySourceListingID *uuid.UUID
	err := a.pool.QueryRow(ctx, `
SELECT listing_id, property_offering_id, primary_source_listing_id
FROM public.listing_search_documents
WHERE canonical_id = $1
    OR listing_id::text = $1
    OR property_offering_id::text = $1
ORDER BY last_seen_at DESC NULLS LAST, refreshed_at DESC
LIMIT 1`, canonicalID).Scan(&listingID, &offeringID, &primarySourceListingID)
	if err != nil {
		return entityGroupingOutput{}, nil
	}
	out := entityGroupingOutput{ListingID: listingID.String(), OfferingID: offeringID.String()}
	metaListingID := listingID
	if primarySourceListingID != nil {
		metaListingID = *primarySourceListingID
	}
	meta, err := a.entityListingMetadata(ctx, metaListingID, &offeringID)
	if err != nil {
		return entityGroupingOutput{}, err
	}
	out.PriceMatch = meta.PriceMatch
	out.Insights = meta.Insights
	rows, err := a.pool.Query(ctx, `
SELECT
    COALESCE(doc.primary_source_listing_id::text, doc.listing_id::text),
    doc.canonical_id,
    COALESCE(evidence.provider, doc.source),
    CASE evidence.source_kind
        WHEN 'frontdoor_building_announcement' THEN 'announcement'
        WHEN 'frontdoor_ad' THEN 'ad'
        WHEN 'shortcut_ad' THEN 'ad'
        ELSE doc.kind
    END,
    COALESCE(evidence.external_id, doc.native_id),
    COALESCE(doc.headline, ''),
    COALESCE(doc.address, ''),
    COALESCE(doc.city, ''),
    COALESCE(doc.postal, ''),
    doc.asking_price,
    doc.area_m2,
    COALESCE(evidence.url, doc.url, ''),
    (COALESCE(evidence.url, doc.url) IS NOT NULL AND doc.last_seen_at >= now() - interval '7 days'),
    doc.last_seen_at,
    entity_evidence.link_status,
    entity_evidence.link_method,
    (entity_evidence.confidence * 100)::int4,
    price_match.transaction_id,
    COALESCE(price_match.match_scope, ''),
    COALESCE(price_match.match_status, ''),
    COALESCE(price_match.match_method, ''),
    price_match.match_score,
    COALESCE(price_match.price_eur, 0)::bigint,
    COALESCE(price_match.description, ''),
    COALESCE(price_match.type, ''),
    COALESCE(price_match.category, ''),
    price_match.updated_at,
    COALESCE(insight_rows.insights_json, '[]'::jsonb)
FROM public.listing_search_documents doc
JOIN public.entity_evidence entity_evidence ON entity_evidence.listing_id = doc.listing_id
    AND entity_evidence.link_status <> 'rejected'
JOIN public.evidence_sources evidence ON evidence.evidence_source_id = entity_evidence.evidence_source_id
LEFT JOIN LATERAL (
    SELECT
        pt.prices_transaction_id AS transaction_id,
        match_source.match_scope,
        match_source.match_status,
        match_source.match_method,
        match_source.match_score,
        pt.prices_transaction_price AS price_eur,
        pt.prices_transaction_description AS description,
        pt.prices_transaction_type AS type,
        pt.prices_transaction_category AS category,
        pt.prices_transaction_updated_at AS updated_at
    FROM (
        SELECT
            pl.prices_transaction_id,
            pl.target_type AS match_scope,
            pl.link_status AS match_status,
            pl.link_method AS match_method,
            pl.link_score::int4 AS match_score,
            CASE WHEN pl.target_type = 'source_listing' THEN 0 ELSE 1 END AS priority
        FROM public.price_links pl
        WHERE pl.link_status <> 'rejected'
            AND (
                (doc.primary_source_listing_id IS NOT NULL AND pl.target_type = 'source_listing' AND pl.target_id = doc.primary_source_listing_id)
                OR (pl.target_type = 'listing' AND pl.target_id = doc.property_offering_id)
            )
    ) match_source
    JOIN origin.prices_transactions pt ON pt.prices_transaction_id = match_source.prices_transaction_id
    ORDER BY match_source.priority, match_source.match_score DESC NULLS LAST, pt.prices_transaction_updated_at DESC
    LIMIT 1
) price_match ON true
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object(
        'key', observation.observation_key,
        'value', observation.value #>> '{}',
        'direction', observation.direction,
        'severity', observation.severity,
        'confidence', observation.confidence,
        'source_field', COALESCE(observation.evidence ->> 'source_field', ''),
        'text', COALESCE(observation.text, '')
    ) ORDER BY observation.severity DESC, observation.observation_key) AS insights_json
    FROM public.target_observations observation
    WHERE observation.source_type = 'source_listing'
        AND observation.source_id = doc.primary_source_listing_id
        AND observation.superseded_at IS NULL
) insight_rows ON true
WHERE doc.listing_id = $1
ORDER BY doc.last_seen_at DESC NULLS LAST, evidence.provider, evidence.external_id`, listingID)
	if err != nil {
		return entityGroupingOutput{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var record entitySourceRecordOutput
		var transactionID *uuid.UUID
		var priceMatch entityPriceMatchOutput
		var insightsJSON json.RawMessage
		if err := rows.Scan(&record.ListingID, &record.CanonicalID, &record.Source, &record.Kind, &record.NativeID, &record.Headline, &record.Address, &record.City, &record.Postal, &record.AskingPrice, &record.Area, &record.URL, &record.ExternalURLAvailable, &record.LastSeenAt, &record.LinkStatus, &record.LinkMethod, &record.LinkScore, &transactionID, &priceMatch.Scope, &priceMatch.Status, &priceMatch.Method, &priceMatch.Score, &priceMatch.PriceEUR, &priceMatch.Description, &priceMatch.Type, &priceMatch.Category, &priceMatch.UpdatedAt, &insightsJSON); err != nil {
			return entityGroupingOutput{}, err
		}
		if transactionID != nil {
			priceMatch.TransactionID = transactionID.String()
			record.PriceMatch = &priceMatch
		}
		record.Insights = parseEntityInsights(insightsJSON)
		out.SourceRecords = append(out.SourceRecords, record)
	}
	if err := rows.Err(); err != nil {
		return entityGroupingOutput{}, err
	}
	out.SourceCount = len(out.SourceRecords)
	return out, nil
}

type entityListingMetadataOutput struct {
	PriceMatch *entityPriceMatchOutput
	Insights   []entityInsightOutput
}

func (a *API) entityListingMetadata(ctx context.Context, listingID uuid.UUID, offeringID *uuid.UUID) (entityListingMetadataOutput, error) {
	var transactionID *uuid.UUID
	var priceMatch entityPriceMatchOutput
	var insightsJSON json.RawMessage
	var offeringArg any
	if offeringID != nil {
		offeringArg = *offeringID
	}
	err := a.pool.QueryRow(ctx, `
SELECT
    price_match.transaction_id,
    COALESCE(price_match.match_scope, ''),
    COALESCE(price_match.match_status, ''),
    COALESCE(price_match.match_method, ''),
    price_match.match_score,
    COALESCE(price_match.price_eur, 0)::bigint,
    COALESCE(price_match.description, ''),
    COALESCE(price_match.type, ''),
    COALESCE(price_match.category, ''),
    price_match.updated_at,
    COALESCE(insight_rows.insights_json, '[]'::jsonb)
FROM (SELECT $1::uuid AS sale_listing_id, $2::uuid AS property_offering_id) selected
LEFT JOIN LATERAL (
    SELECT
        pt.prices_transaction_id AS transaction_id,
        match_source.match_scope,
        match_source.match_status,
        match_source.match_method,
        match_source.match_score,
        pt.prices_transaction_price AS price_eur,
        pt.prices_transaction_description AS description,
        pt.prices_transaction_type AS type,
        pt.prices_transaction_category AS category,
        pt.prices_transaction_updated_at AS updated_at
    FROM (
        SELECT
            pl.prices_transaction_id,
            pl.target_type AS match_scope,
            pl.link_status AS match_status,
            pl.link_method AS match_method,
            pl.link_score::int4 AS match_score,
            CASE WHEN pl.target_type = 'source_listing' THEN 0 ELSE 1 END AS priority
        FROM public.price_links pl
        WHERE pl.link_status <> 'rejected'
            AND (
                (pl.target_type = 'source_listing' AND pl.target_id = selected.sale_listing_id)
                OR (selected.property_offering_id IS NOT NULL AND pl.target_type = 'listing' AND pl.target_id = selected.property_offering_id)
            )
    ) match_source
    JOIN origin.prices_transactions pt ON pt.prices_transaction_id = match_source.prices_transaction_id
    ORDER BY match_source.priority, match_source.match_score DESC NULLS LAST, pt.prices_transaction_updated_at DESC
    LIMIT 1
) price_match ON true
LEFT JOIN LATERAL (
    SELECT jsonb_agg(jsonb_build_object(
        'key', observation.observation_key,
        'value', observation.value #>> '{}',
        'direction', observation.direction,
        'severity', observation.severity,
        'confidence', observation.confidence,
        'source_field', COALESCE(observation.evidence ->> 'source_field', ''),
        'text', COALESCE(observation.text, '')
    ) ORDER BY observation.severity DESC, observation.observation_key) AS insights_json
    FROM public.target_observations observation
    WHERE observation.source_type = 'source_listing'
        AND observation.source_id = selected.sale_listing_id
        AND observation.superseded_at IS NULL
) insight_rows ON true`, listingID, offeringArg).Scan(&transactionID, &priceMatch.Scope, &priceMatch.Status, &priceMatch.Method, &priceMatch.Score, &priceMatch.PriceEUR, &priceMatch.Description, &priceMatch.Type, &priceMatch.Category, &priceMatch.UpdatedAt, &insightsJSON)
	if err != nil {
		return entityListingMetadataOutput{}, err
	}
	out := entityListingMetadataOutput{Insights: parseEntityInsights(insightsJSON)}
	if transactionID != nil {
		priceMatch.TransactionID = transactionID.String()
		out.PriceMatch = &priceMatch
	}
	return out, nil
}

func parseEntityInsights(raw json.RawMessage) []entityInsightOutput {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil
	}
	var out []entityInsightOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func toDetailFields(fields []ads.DetailField) []detailFieldOutput {
	if len(fields) == 0 {
		return nil
	}
	out := make([]detailFieldOutput, len(fields))
	for i, f := range fields {
		out[i] = detailFieldOutput{Label: f.Label, Value: f.Value}
	}
	return out
}
