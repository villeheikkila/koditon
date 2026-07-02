package properties

import (
	"context"
	"encoding/json"
	"fmt"

	"koditon/internal/db"
)

func (s *Service) searchListings(ctx context.Context, params SearchParams, listingType string) ([]listingSearchRow, error) {
	if listingType == "rental" {
		rows, err := s.queries.SearchRentalListings(ctx, rentalSearchParams(params))
		if err != nil {
			return nil, fmt.Errorf("search rental listings: %w", err)
		}
		out := make([]listingSearchRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, listingSearchRowFromRental(row))
		}
		return out, nil
	}
	rows, err := s.queries.SearchSaleListings(ctx, saleSearchParams(params))
	if err != nil {
		return nil, fmt.Errorf("search sale listings: %w", err)
	}
	out := make([]listingSearchRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, listingSearchRowFromSale(row))
	}
	return out, nil
}

func (s *Service) countListings(ctx context.Context, params SearchParams, listingType string) (int64, error) {
	if listingType == "rental" {
		count, err := s.queries.CountRentalListings(ctx, rentalCountParams(params))
		if err != nil {
			return 0, fmt.Errorf("count rental listings: %w", err)
		}
		return int64Value(count), nil
	}
	count, err := s.queries.CountSaleListings(ctx, saleCountParams(params))
	if err != nil {
		return 0, fmt.Errorf("count sale listings: %w", err)
	}
	return int64Value(count), nil
}

func saleSearchParams(params SearchParams) db.SearchSaleListingsParams {
	offset := (params.Page - 1) * params.PageSize
	limit := params.PageSize
	return db.SearchSaleListingsParams{Source: &params.Source, QueryText: emptyToNil(params.Query), City: emptyToNil(params.City), Postal: emptyToNil(params.Postal), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore, MinPricePerM2: params.MinPricePerM2, MaxPricePerM2: params.MaxPricePerM2, Rooms: params.Rooms, Floor: params.Floor, MinBuildYear: params.MinBuildYear, MaxBuildYear: params.MaxBuildYear, Condition: emptyToNil(params.Condition), EnergyClass: emptyToNil(params.EnergyClass), Kind: &params.Kind, SortMode: &params.Sort, OffsetCount: &offset, LimitCount: &limit}
}

func saleCountParams(params SearchParams) db.CountSaleListingsParams {
	return db.CountSaleListingsParams{Source: &params.Source, QueryText: emptyToNil(params.Query), City: emptyToNil(params.City), Postal: emptyToNil(params.Postal), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore, MinPricePerM2: params.MinPricePerM2, MaxPricePerM2: params.MaxPricePerM2, Rooms: params.Rooms, Floor: params.Floor, MinBuildYear: params.MinBuildYear, MaxBuildYear: params.MaxBuildYear, Condition: emptyToNil(params.Condition), EnergyClass: emptyToNil(params.EnergyClass), Kind: &params.Kind}
}

func rentalSearchParams(params SearchParams) db.SearchRentalListingsParams {
	offset := (params.Page - 1) * params.PageSize
	return db.SearchRentalListingsParams{Source: &params.Source, QueryText: emptyToNil(params.Query), City: emptyToNil(params.City), Postal: emptyToNil(params.Postal), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore, SortMode: &params.Sort, OffsetCount: offset, LimitCount: params.PageSize}
}

func rentalCountParams(params SearchParams) db.CountRentalListingsParams {
	return db.CountRentalListingsParams{Source: &params.Source, QueryText: emptyToNil(params.Query), City: emptyToNil(params.City), Postal: emptyToNil(params.Postal), MinPrice: params.MinPrice, MaxPrice: params.MaxPrice, MinArea: params.MinArea, MaxArea: params.MaxArea, PublishedAfter: params.PublishedAfter, PublishedBefore: params.PublishedBefore}
}

func listingSearchRowFromSale(row db.SearchSaleListingsRow) listingSearchRow {
	return listingSearchRow{Source: row.Source, Kind: row.Kind, NativeID: row.NativeID, CanonicalID: row.CanonicalID, PublicID: valueOrEmpty(row.PublicID), URL: valueOrEmpty(row.Url), Headline: valueOrEmpty(row.Headline), Address: valueOrEmpty(row.Address), City: valueOrEmpty(row.City), Postal: valueOrEmpty(row.Postal), Price: row.Price, Area: row.Area, RoomLayout: valueOrEmpty(row.RoomLayout), PricePerM2: row.PricePerM2, DebtFreePrice: row.DebtFreePrice, DebtShareAmount: row.DebtShareAmount, RoomsCount: row.RoomsCount, FloorLevel: row.FloorLevel, TotalFloors: row.TotalFloors, BuildYear: row.BuildYear, Condition: row.Condition, EnergyClass: row.EnergyClass, EnergyEfficiencyLabel: row.EnergyEfficiencyLabel, LastSeenAt: row.LastSeenAt, PublishedAt: row.PublishedAt, BuildingKeyAddress: valueOrEmpty(row.BuildingKeyAddress), SourceProviders: row.SourceProviders}
}

func listingSearchRowFromRental(row db.SearchRentalListingsRow) listingSearchRow {
	return listingSearchRow{Source: valueOrEmpty(row.Source), Kind: valueOrEmpty(row.Kind), NativeID: valueOrEmpty(row.NativeID), CanonicalID: valueOrEmpty(row.CanonicalID), PublicID: valueOrEmpty(row.PublicID), URL: valueOrEmpty(row.Url), Headline: valueOrEmpty(row.Headline), Address: valueOrEmpty(row.Address), City: valueOrEmpty(row.City), Postal: valueOrEmpty(row.Postal), Price: row.Price, Area: row.Area, RoomLayout: valueOrEmpty(row.RoomLayout), PricePerM2: row.PricePerM2, DebtFreePrice: row.DebtFreePrice, DebtShareAmount: row.DebtShareAmount, RoomsCount: row.RoomsCount, FloorLevel: row.FloorLevel, TotalFloors: row.TotalFloors, BuildYear: row.BuildYear, Condition: row.Condition, EnergyClass: row.EnergyClass, EnergyEfficiencyLabel: row.EnergyEfficiencyLabel, LastSeenAt: row.LastSeenAt, PublishedAt: row.PublishedAt, BuildingKeyAddress: valueOrEmpty(row.BuildingKeyAddress), SourceProviders: row.SourceProviders}
}

func (s *Service) SaleListingMap(ctx context.Context, bounds MapBounds) (SaleListingMap, error) {
	source := normalizeSource(bounds.Source)
	kind := normalizeListingKind(bounds.Kind)
	limit := bounds.Limit
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	rows, err := s.db.Query(ctx, `
WITH visible_base AS (
    SELECT
        pb.housing_company_id,
        postgis.ST_SnapToGrid(pb.housing_company_geom, 0.000001) AS marker_geom,
        postgis.ST_AsEWKT(postgis.ST_SnapToGrid(pb.housing_company_geom, 0.000001)) AS marker_key,
        pb.housing_company_address_norm,
        pb.housing_company_city_norm,
        pb.housing_company_postal_norm,
        COALESCE((building_profile.dimensions #>> '{building,build_year}')::integer, (housing_profile.dimensions #>> '{housing_company,build_year}')::integer, pb.housing_company_build_year, sl.sale_listing_build_year) AS housing_company_build_year,
        po.property_offering_id,
        po.property_offering_headline,
        po.property_offering_asking_price,
        po.property_offering_price_per_m2,
        COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::double precision, pu.property_unit_area_value, sl.sale_listing_area_value) AS property_unit_area_value,
        COALESCE(NULLIF(unit_profile.dimensions #>> '{layout,room_layout}', ''), pu.property_unit_room_layout, sl.sale_listing_room_layout) AS property_unit_room_layout,
        COALESCE((unit_profile.dimensions #>> '{layout,room_count}')::integer, pu.property_unit_rooms_count, sl.sale_listing_rooms_count) AS property_unit_rooms_count,
        COALESCE(NULLIF(unit_profile.dimensions #>> '{condition,unit_condition}', ''), sl.sale_listing_condition) AS property_unit_condition,
        COALESCE(NULLIF(building_profile.dimensions #>> '{building,energy_class}', ''), NULLIF(housing_profile.dimensions #>> '{housing_company,energy_class}', ''), sl.sale_listing_energy_class) AS building_energy_class,
        po.property_offering_last_seen_at,
        sl.sale_listing_street_address,
        COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm) AS sale_listing_city,
        COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm) AS sale_listing_postal,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        row_number() OVER (
            PARTITION BY po.property_offering_id
            ORDER BY
                CASE
                    WHEN sl.frontdoor_ad_id IS NOT NULL THEN 0
                    WHEN sl.shortcut_ad_id IS NOT NULL THEN 1
                    ELSE 2
                END,
                sl.sale_listing_last_seen_at DESC NULLS LAST,
                sl.sale_listing_created_at DESC
        ) AS source_rank
    FROM public.housing_companies pb
    JOIN public.property_units pu ON pu.housing_company_id = pb.housing_company_id
    JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = po.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    LEFT JOIN public.dimension_profiles unit_profile ON unit_profile.target_type = 'unit'
        AND unit_profile.target_id = pu.property_unit_id
    LEFT JOIN public.dimension_profiles building_profile ON building_profile.target_type = 'building'
        AND building_profile.target_id = pu.physical_building_id
    LEFT JOIN public.dimension_profiles housing_profile ON housing_profile.target_type = 'housing_company'
        AND housing_profile.target_id = pb.housing_company_id
    WHERE pb.housing_company_geom IS NOT NULL
        AND ($1 = 'all' OR sl.sale_listing_source_provider = $1)
        AND ($2 = 'all' OR sl.sale_listing_source_kind = $2)
        AND ($8::text IS NULL OR lower(concat_ws(' ', sl.sale_listing_search_text, sl.sale_listing_description_text, sl.sale_listing_street_address, pb.housing_company_address_norm, pb.housing_company_name)) LIKE ('%' || lower(trim($8::text)) || '%'))
        AND (
            $9::text IS NULL
            OR lower(COALESCE(pb.housing_company_city_norm, '')) LIKE ('%' || lower(trim($9::text)) || '%')
            OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')) LIKE ('%' || lower(trim($9::text)) || '%')
        )
        AND ($10::text IS NULL OR public.fnc__normalize_postal(COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, pb.housing_company_postal_norm, '')) = public.fnc__normalize_postal($10::text))
        AND ($11::bigint IS NULL OR COALESCE(po.property_offering_asking_price, sl.sale_listing_asking_price) >= $11::bigint)
        AND ($12::bigint IS NULL OR COALESCE(po.property_offering_asking_price, sl.sale_listing_asking_price) <= $12::bigint)
        AND ($13::double precision IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::double precision, pu.property_unit_area_value, sl.sale_listing_area_value) >= $13::double precision)
        AND ($14::double precision IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::double precision, pu.property_unit_area_value, sl.sale_listing_area_value) <= $14::double precision)
        AND ($15::double precision IS NULL OR COALESCE(po.property_offering_price_per_m2, sl.sale_listing_price_per_m2) >= $15::double precision)
        AND ($16::double precision IS NULL OR COALESCE(po.property_offering_price_per_m2, sl.sale_listing_price_per_m2) <= $16::double precision)
        AND ($17::integer IS NULL OR COALESCE((unit_profile.dimensions #>> '{layout,room_count}')::integer, pu.property_unit_rooms_count, sl.sale_listing_rooms_count) = $17::integer)
        AND ($18::integer IS NULL OR COALESCE((building_profile.dimensions #>> '{building,build_year}')::integer, (housing_profile.dimensions #>> '{housing_company,build_year}')::integer, pb.housing_company_build_year, sl.sale_listing_build_year) >= $18::integer)
        AND ($19::integer IS NULL OR COALESCE((building_profile.dimensions #>> '{building,build_year}')::integer, (housing_profile.dimensions #>> '{housing_company,build_year}')::integer, pb.housing_company_build_year, sl.sale_listing_build_year) <= $19::integer)
        AND ($20::text IS NULL OR sl.sale_listing_property_type_code = public.fnc__sale_listing_property_type_code($20::text) OR lower(COALESCE(sl.sale_listing_property_type_raw, '')) LIKE ('%' || lower(trim($20::text)) || '%'))
        AND ($21::text IS NULL OR public.fnc__condition_match_code(COALESCE(NULLIF(unit_profile.dimensions #>> '{condition,unit_condition}', ''), sl.sale_listing_condition)) = public.fnc__condition_match_code($21::text) OR lower(COALESCE(NULLIF(unit_profile.dimensions #>> '{condition,unit_condition}', ''), sl.sale_listing_condition, '')) LIKE ('%' || lower(trim($21::text)) || '%'))
        AND ($22::text IS NULL OR public.fnc__energy_efficiency_match_code(COALESCE(NULLIF(building_profile.dimensions #>> '{building,energy_class}', ''), NULLIF(housing_profile.dimensions #>> '{housing_company,energy_class}', ''), sl.sale_listing_energy_class)) = public.fnc__energy_efficiency_match_code($22::text) OR lower(concat_ws(' ', COALESCE(NULLIF(building_profile.dimensions #>> '{building,energy_class}', ''), NULLIF(housing_profile.dimensions #>> '{housing_company,energy_class}', ''), sl.sale_listing_energy_class), sl.sale_listing_energy_efficiency_label)) LIKE ('%' || lower(trim($22::text)) || '%'))
        AND ($23::boolean IS NULL OR sl.sale_listing_elevator IS NOT DISTINCT FROM $23::boolean)
        AND ($24::boolean IS NULL OR sl.sale_listing_sauna IS NOT DISTINCT FROM $24::boolean)
        AND ($25::boolean IS NULL OR sl.sale_listing_balcony IS NOT DISTINCT FROM $25::boolean)
        AND ($26::boolean IS NULL OR sl.sale_listing_plot_owned IS NOT DISTINCT FROM $26::boolean)
        AND ($27::boolean IS NULL OR sl.sale_listing_new_development IS NOT DISTINCT FROM $27::boolean)
        AND (
            $28::boolean IS NULL
            OR EXISTS (
                SELECT 1
                FROM public.price_links pl
                WHERE pl.target_type = 'listing'
                    AND pl.target_id = po.property_offering_id
                    AND pl.link_status <> 'rejected'
            ) IS NOT DISTINCT FROM $28::boolean
        )
        AND (
            $3::double precision IS NULL
            OR postgis.ST_Intersects(
                pb.housing_company_geom,
                postgis.ST_MakeEnvelope($4::double precision, $3::double precision, $6::double precision, $5::double precision, 4326)
            )
        )
),
visible AS (
    SELECT *
    FROM visible_base
    WHERE source_rank = 1
),
source_summary AS (
    SELECT
        property_offering_id,
        array_agg(DISTINCT sale_listing_source_provider ORDER BY sale_listing_source_provider) AS offering_providers,
        array_agg(DISTINCT sale_listing_source_kind ORDER BY sale_listing_source_kind) AS offering_kinds
    FROM visible_base
    GROUP BY property_offering_id
),
grouped AS (
    SELECT
        marker_geom,
        marker_key,
        count(DISTINCT property_offering_id)::bigint AS offering_count,
        count(DISTINCT housing_company_id)::bigint AS housing_company_count,
        min(housing_company_address_norm) AS address,
        min(housing_company_city_norm) AS city,
        min(housing_company_postal_norm) AS postal,
        min(property_offering_asking_price) AS min_price,
        max(property_offering_asking_price) AS max_price,
        min(property_unit_area_value) AS min_area,
        max(property_unit_area_value) AS max_area,
        max(property_offering_last_seen_at) AS last_seen_at,
        array_agg(DISTINCT sale_listing_source_provider ORDER BY sale_listing_source_provider) AS providers,
        array_agg(DISTINCT sale_listing_source_kind ORDER BY sale_listing_source_kind) AS kinds,
        (array_agg(DISTINCT property_offering_id::text))[1:8] AS listing_ids,
        min(housing_company_id::text) AS housing_company_id
    FROM visible
    GROUP BY marker_geom, marker_key
),
listing_cards AS (
    SELECT
        marker_key,
        jsonb_agg(listing ORDER BY property_offering_last_seen_at DESC NULLS LAST, property_offering_asking_price ASC NULLS LAST) AS listings
    FROM (
        SELECT
            visible.marker_key,
            visible.property_offering_last_seen_at,
            visible.property_offering_asking_price,
            jsonb_build_object(
                'id', visible.property_offering_id::text,
                'headline', visible.property_offering_headline,
                'address', COALESCE(NULLIF(visible.sale_listing_street_address, ''), visible.housing_company_address_norm),
                'city', COALESCE(NULLIF(visible.sale_listing_city, ''), visible.housing_company_city_norm),
                'postal', COALESCE(NULLIF(visible.sale_listing_postal, ''), visible.housing_company_postal_norm),
                'layout', visible.property_unit_room_layout,
                'area_m2', visible.property_unit_area_value,
                'price', visible.property_offering_asking_price,
                'price_per_m2', visible.property_offering_price_per_m2,
                'build_year', visible.housing_company_build_year,
                'last_seen_at', visible.property_offering_last_seen_at,
                'providers', COALESCE(source_summary.offering_providers, ARRAY[]::text[]),
                'kinds', COALESCE(source_summary.offering_kinds, ARRAY[]::text[])
            ) AS listing,
            row_number() OVER (
                PARTITION BY visible.marker_key
                ORDER BY visible.property_offering_last_seen_at DESC NULLS LAST, visible.property_offering_asking_price ASC NULLS LAST
            ) AS listing_rank
        FROM visible
        LEFT JOIN source_summary ON source_summary.property_offering_id = visible.property_offering_id
    ) ranked
    WHERE listing_rank <= 24
    GROUP BY marker_key
)
SELECT
    postgis.ST_Y(marker_geom)::double precision AS lat,
    postgis.ST_X(marker_geom)::double precision AS lng,
    offering_count,
    COALESCE(address, ''),
    COALESCE(city, ''),
    COALESCE(postal, ''),
    min_price,
    max_price,
    min_area,
    max_area,
    last_seen_at,
    providers,
    kinds,
    listing_ids,
    COALESCE(listing_cards.listings, '[]'::jsonb),
    housing_company_id,
    housing_company_count
FROM grouped
LEFT JOIN listing_cards USING (marker_key)
ORDER BY last_seen_at DESC NULLS LAST, offering_count DESC
LIMIT $7::int`, source, kind, bounds.MinLat, bounds.MinLng, bounds.MaxLat, bounds.MaxLng, limit, emptyToNil(bounds.Query), emptyToNil(bounds.City), emptyToNil(bounds.Postal), bounds.MinPrice, bounds.MaxPrice, bounds.MinArea, bounds.MaxArea, bounds.MinPricePerM2, bounds.MaxPricePerM2, bounds.Rooms, bounds.MinBuildYear, bounds.MaxBuildYear, emptyToNil(bounds.PropertyType), emptyToNil(bounds.Condition), emptyToNil(bounds.EnergyClass), bounds.Elevator, bounds.Sauna, bounds.Balcony, bounds.PlotOwned, bounds.NewDevelopment, bounds.HasTransaction)
	if err != nil {
		return SaleListingMap{}, fmt.Errorf("query sale listing map: %w", err)
	}
	defer rows.Close()
	out := SaleListingMap{Markers: []SaleListingMapMarker{}}
	for rows.Next() {
		var marker SaleListingMapMarker
		var listingsJSON []byte
		if err := rows.Scan(&marker.Lat, &marker.Lng, &marker.Count, &marker.Address, &marker.City, &marker.Postal, &marker.MinPrice, &marker.MaxPrice, &marker.MinAreaM2, &marker.MaxAreaM2, &marker.LastSeenAt, &marker.Providers, &marker.Kinds, &marker.ListingIDs, &listingsJSON, &marker.HousingCompanyID, &marker.HousingCompanyCount); err != nil {
			return SaleListingMap{}, fmt.Errorf("scan sale listing map marker: %w", err)
		}
		if err := json.Unmarshal(listingsJSON, &marker.Listings); err != nil {
			return SaleListingMap{}, fmt.Errorf("decode sale listing map listings: %w", err)
		}
		out.Markers = append(out.Markers, marker)
	}
	if err := rows.Err(); err != nil {
		return SaleListingMap{}, fmt.Errorf("iterate sale listing map markers: %w", err)
	}
	return out, nil
}

func (s *Service) SaleListingMapFilterOptions(ctx context.Context, sourceValue string, kindValue string) (SaleListingMapFilterOptions, error) {
	source := normalizeSource(sourceValue)
	kind := normalizeListingKind(kindValue)
	rows, err := s.db.Query(ctx, `
WITH source_rows AS (
    SELECT
        NULLIF(trim(pb.housing_company_city_norm), '') AS city_norm,
        NULLIF(public.fnc__normalize_postal(pb.housing_company_postal_norm), '') AS postal,
        pb.housing_company_geom AS geom
    FROM public.housing_companies pb
    JOIN public.property_units pu ON pu.housing_company_id = pb.housing_company_id
    JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = po.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    WHERE pb.housing_company_geom IS NOT NULL
        AND ($1 = 'all' OR sl.sale_listing_source_provider = $1)
        AND ($2 = 'all' OR sl.sale_listing_source_kind = $2)
),
city_rows AS (
    SELECT
        'city'::text AS option_kind,
        initcap(city_norm) AS value,
        initcap(city_norm) AS label,
        ''::text AS meta,
        avg(postgis.ST_Y(geom))::double precision AS lat,
        avg(postgis.ST_X(geom))::double precision AS lng
    FROM source_rows
    WHERE city_norm IS NOT NULL
    GROUP BY city_norm
),
postal_rows AS (
    SELECT
        'postal'::text AS option_kind,
        postal AS value,
        postal AS label,
        concat_ws(' ', NULLIF(ppc.postal_postal_code_name_fi, ''), NULLIF(pm.postal_municipality_name_fi, '')) AS meta,
        avg(postgis.ST_Y(source_rows.geom))::double precision AS lat,
        avg(postgis.ST_X(source_rows.geom))::double precision AS lng
    FROM source_rows
    LEFT JOIN origin.postal_postal_codes ppc ON ppc.postal_postal_code_code = source_rows.postal
    LEFT JOIN origin.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
    WHERE postal IS NOT NULL
    GROUP BY postal, ppc.postal_postal_code_name_fi, pm.postal_municipality_name_fi
)
SELECT option_kind, value, label, COALESCE(meta, '') AS meta, lat, lng
FROM city_rows
UNION ALL
SELECT option_kind, value, label, COALESCE(meta, '') AS meta, lat, lng
FROM postal_rows
ORDER BY option_kind, value`, source, kind)
	if err != nil {
		return SaleListingMapFilterOptions{}, fmt.Errorf("query sale listing map filter options: %w", err)
	}
	defer rows.Close()
	out := SaleListingMapFilterOptions{Cities: []SaleListingMapFilterOption{}, Postals: []SaleListingMapFilterOption{}}
	for rows.Next() {
		var optionKind string
		var option SaleListingMapFilterOption
		if err := rows.Scan(&optionKind, &option.Value, &option.Label, &option.Meta, &option.Lat, &option.Lng); err != nil {
			return SaleListingMapFilterOptions{}, fmt.Errorf("scan sale listing map filter option: %w", err)
		}
		if optionKind == "city" {
			out.Cities = append(out.Cities, option)
		} else {
			out.Postals = append(out.Postals, option)
		}
	}
	if err := rows.Err(); err != nil {
		return SaleListingMapFilterOptions{}, fmt.Errorf("iterate sale listing map filter options: %w", err)
	}
	return out, nil
}
