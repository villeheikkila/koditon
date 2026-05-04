package properties

import (
	"context"
	"encoding/json"
	"fmt"
)

const transactionMatchPostalsSQL = `
WITH latest AS (
    SELECT DISTINCT ON (c.sale_listing_id, c.prices_transaction_id)
        c.*
    FROM public.sale_listing_prices_transaction_match_candidates c
    ORDER BY c.sale_listing_id, c.prices_transaction_id, c.sale_listing_prices_transaction_match_created_at DESC
),
potential AS (
    SELECT
        latest.*,
        sl.sale_listing_postal_norm AS postal
    FROM latest
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = latest.sale_listing_id
    WHERE latest.sale_listing_prices_transaction_match_status = ANY(ARRAY['candidate'::text, 'ambiguous'::text])
        AND sl.prices_transaction_id IS NULL
        AND sl.sale_listing_postal_norm IS NOT NULL
        AND NOT EXISTS (
            SELECT 1
            FROM public.property_source_offerings linked
            WHERE linked.prices_transaction_id = latest.prices_transaction_id
        )
)
SELECT
    postal,
    COALESCE(ppc.postal_postal_code_name_fi, '') AS postal_name_fi,
    COALESCE(pm.postal_municipality_name_fi, '') AS municipality_name,
    count(*)::bigint AS candidate_count,
    count(DISTINCT sale_listing_id)::bigint AS listing_count,
    count(DISTINCT prices_transaction_id)::bigint AS transaction_count,
    count(*) FILTER (WHERE sale_listing_prices_transaction_match_confidence = 'high')::bigint AS high_count,
    count(*) FILTER (WHERE sale_listing_prices_transaction_match_confidence = 'medium')::bigint AS medium_count,
    count(*) FILTER (WHERE sale_listing_prices_transaction_match_confidence = 'low')::bigint AS low_count,
    count(*) FILTER (WHERE sale_listing_prices_transaction_match_status = 'ambiguous')::bigint AS ambiguous_count,
    COALESCE(max(sale_listing_prices_transaction_match_created_at)::text, '') AS latest_at
FROM potential
LEFT JOIN public.postal_postal_codes ppc ON ppc.postal_postal_code_code = potential.postal
LEFT JOIN public.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
GROUP BY postal, ppc.postal_postal_code_name_fi, pm.postal_municipality_name_fi
ORDER BY candidate_count DESC, postal
LIMIT $1`

const transactionMatchCandidatesSQL = `
WITH latest AS (
    SELECT DISTINCT ON (c.sale_listing_id, c.prices_transaction_id)
        c.*
    FROM public.sale_listing_prices_transaction_match_candidates c
    ORDER BY c.sale_listing_id, c.prices_transaction_id, c.sale_listing_prices_transaction_match_created_at DESC
)
SELECT
    latest.sale_listing_prices_transaction_match_candidate_id::text,
    latest.sale_listing_prices_transaction_match_status,
    latest.sale_listing_prices_transaction_match_score::int4,
    latest.sale_listing_prices_transaction_match_confidence,
    latest.sale_listing_prices_transaction_match_price_delta_percent,
    latest.sale_listing_prices_transaction_match_reasons,
    COALESCE(latest.sale_listing_prices_transaction_match_created_at::text, ''),
    pos.property_offering_id::text,
    sl.sale_listing_canonical_id,
    sl.sale_listing_source_provider,
    COALESCE(sl.sale_listing_url, ''),
    COALESCE(sl.sale_listing_headline, ''),
    COALESCE(sl.sale_listing_street_address, ''),
    COALESCE(sl.sale_listing_city, ''),
    COALESCE(sl.sale_listing_postal_norm, ''),
    COALESCE(sl.sale_listing_room_layout, ''),
    COALESCE(sl.sale_listing_condition, ''),
    COALESCE(public.fnc__condition_match_code(sl.sale_listing_condition), ''),
    sl.sale_listing_area_value,
    sl.sale_listing_asking_price,
    sl.sale_listing_price_per_m2,
    sl.sale_listing_build_year,
    sl.sale_listing_floor_level,
    sl.sale_listing_total_floors,
    sl.sale_listing_elevator,
    COALESCE(sl.sale_listing_energy_efficiency_match_code, ''),
    COALESCE(sl.sale_listing_energy_efficiency_label, ''),
    COALESCE(sl.sale_listing_plot_type_raw, ''),
    sl.sale_listing_plot_owned,
    COALESCE(sl.sale_listing_first_seen_at::text, ''),
    COALESCE(sl.sale_listing_last_seen_at::text, ''),
    pt.prices_transaction_id::text,
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    pt.prices_transaction_area,
    pt.prices_transaction_price,
    pt.prices_transaction_price_per_square_meter,
    pt.prices_transaction_build_year,
    COALESCE(pt.prices_transaction_floor, ''),
    pt.prices_transaction_elevator,
    COALESCE(pt.prices_transaction_condition, ''),
    COALESCE(public.fnc__condition_match_code(pt.prices_transaction_condition), ''),
    COALESCE(pt.prices_transaction_plot, ''),
    COALESCE(pt.prices_transaction_plot_owned, public.fnc__plot_owned(pt.prices_transaction_plot)),
    COALESCE(pt.prices_transaction_energy_class, ''),
    COALESCE(public.fnc__prices_transaction_energy_match_code(pt.prices_transaction_energy_class), ''),
    COALESCE(pt.prices_transaction_period_identifier, ''),
    COALESCE(pt.prices_transaction_created_at::text, '')
FROM latest
JOIN public.property_source_offerings sl ON sl.sale_listing_id = latest.sale_listing_id
JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
    AND pos.property_offering_source_link_status <> 'rejected'
JOIN public.prices_transactions pt ON pt.prices_transaction_id = latest.prices_transaction_id
WHERE ($3::uuid IS NOT NULL OR latest.sale_listing_prices_transaction_match_status = ANY(ARRAY['candidate'::text, 'ambiguous'::text]))
    AND ($3::uuid IS NOT NULL OR sl.prices_transaction_id IS NULL)
    AND ($1::text IS NULL OR sl.sale_listing_postal_norm = public.fnc__normalize_postal($1::text))
    AND ($2::text IS NULL OR latest.sale_listing_prices_transaction_match_status = $2::text)
    AND ($3::uuid IS NULL OR pt.prices_transaction_id = $3::uuid)
    AND ($3::uuid IS NOT NULL OR NOT EXISTS (
        SELECT 1
        FROM public.property_source_offerings linked
        WHERE linked.prices_transaction_id = latest.prices_transaction_id
    ))
ORDER BY
    latest.sale_listing_prices_transaction_match_score DESC,
    latest.sale_listing_prices_transaction_match_price_delta_percent ASC NULLS LAST,
    sl.sale_listing_postal_norm,
    sl.sale_listing_street_address
LIMIT $4`

func (s *Service) TransactionMatchPostals(ctx context.Context, limit int32) ([]TransactionMatchPostalSummary, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, transactionMatchPostalsSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list transaction match postals: %w", err)
	}
	defer rows.Close()
	out := []TransactionMatchPostalSummary{}
	for rows.Next() {
		var item TransactionMatchPostalSummary
		if err := rows.Scan(&item.Postal, &item.NameFi, &item.MunicipalityName, &item.CandidateCount, &item.ListingCount, &item.TransactionCount, &item.HighCount, &item.MediumCount, &item.LowCount, &item.AmbiguousCount, &item.LatestAt); err != nil {
			return nil, fmt.Errorf("scan transaction match postal: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transaction match postals: %w", err)
	}
	return out, nil
}

func (s *Service) TransactionMatchCandidates(ctx context.Context, postal string, status string, transactionID string, limit int32) ([]TransactionMatchCandidate, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if status != "candidate" && status != "ambiguous" {
		status = ""
	}
	rows, err := s.db.Query(ctx, transactionMatchCandidatesSQL, emptyToNil(postal), emptyToNil(status), emptyToNil(transactionID), limit)
	if err != nil {
		return nil, fmt.Errorf("list transaction match candidates: %w", err)
	}
	defer rows.Close()
	out := []TransactionMatchCandidate{}
	for rows.Next() {
		var item TransactionMatchCandidate
		var reasons json.RawMessage
		var listingArea, listingPricePerM2, priceDelta *float64
		var listingAskingPrice *int64
		var listingBuildYear, floorLevel, totalFloors *int32
		var listingElevator *bool
		var listingPlotOwned, transactionPlotOwned *bool
		var transactionPrice, transactionPricePerM2 int32
		if err := rows.Scan(
			&item.ID,
			&item.Status,
			&item.Score,
			&item.Confidence,
			&priceDelta,
			&reasons,
			&item.CreatedAt,
			&item.Listing.ID,
			&item.Listing.CanonicalID,
			&item.Listing.SourceProvider,
			&item.Listing.URL,
			&item.Listing.Headline,
			&item.Listing.StreetAddress,
			&item.Listing.City,
			&item.Listing.Postal,
			&item.Listing.RoomLayout,
			&item.Listing.Condition,
			&item.Listing.ConditionMatchCode,
			&listingArea,
			&listingAskingPrice,
			&listingPricePerM2,
			&listingBuildYear,
			&floorLevel,
			&totalFloors,
			&listingElevator,
			&item.Listing.EnergyMatchCode,
			&item.Listing.EnergyLabel,
			&item.Listing.PlotOwnershipRaw,
			&listingPlotOwned,
			&item.Listing.FirstSeenAt,
			&item.Listing.LastSeenAt,
			&item.Transaction.ID,
			&item.Transaction.Description,
			&item.Transaction.Type,
			&item.Transaction.Category,
			&item.Transaction.AreaM2,
			&transactionPrice,
			&transactionPricePerM2,
			&item.Transaction.BuildYear,
			&item.Transaction.Floor,
			&item.Transaction.Elevator,
			&item.Transaction.Condition,
			&item.Transaction.ConditionMatchCode,
			&item.Transaction.Plot,
			&transactionPlotOwned,
			&item.Transaction.EnergyClass,
			&item.Transaction.EnergyMatchCode,
			&item.Transaction.PeriodIdentifier,
			&item.Transaction.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction match candidate: %w", err)
		}
		item.PriceDeltaPercent = priceDelta
		item.Reasons = reasons
		item.Listing.AreaM2 = listingArea
		item.Listing.AskingPrice = listingAskingPrice
		item.Listing.PricePerM2 = listingPricePerM2
		item.Listing.BuildYear = listingBuildYear
		item.Listing.FloorLevel = floorLevel
		item.Listing.TotalFloors = totalFloors
		item.Listing.Elevator = listingElevator
		item.Listing.PlotOwned = listingPlotOwned
		item.Listing.Condition = displayCondition(item.Listing.Condition)
		item.Listing.EnergyLabel = displayEnergyClass(item.Listing.EnergyLabel, item.Listing.EnergyMatchCode)
		item.Transaction.Price = int64(transactionPrice)
		item.Transaction.PricePerSquareMeter = int64(transactionPricePerM2)
		item.Transaction.PlotOwned = transactionPlotOwned
		item.Transaction.Condition = displayCondition(item.Transaction.Condition)
		item.Transaction.EnergyClass = displayEnergyClass(item.Transaction.EnergyClass, item.Transaction.EnergyMatchCode)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transaction match candidates: %w", err)
	}
	return out, nil
}
