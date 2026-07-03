package properties

import (
	"context"
	"encoding/json"
	"fmt"

	"koditon/internal/db"
)

func (s *Service) searchListings(ctx context.Context, params SearchParams, listingType string) ([]listingSearchRow, error) {
	if listingType == "rental" {
		return []listingSearchRow{}, nil
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
		return 0, nil
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
WITH visible AS (
    SELECT
        doc.*,
        postgis.ST_SnapToGrid(postgis.ST_SetSRID(postgis.ST_MakePoint(doc.longitude, doc.latitude), 4326), 0.000001) AS marker_geom,
        postgis.ST_AsEWKT(postgis.ST_SnapToGrid(postgis.ST_SetSRID(postgis.ST_MakePoint(doc.longitude, doc.latitude), 4326), 0.000001)) AS marker_key
    FROM public.listing_search_documents doc
    WHERE doc.listing_status = 'active'
        AND doc.latitude IS NOT NULL
        AND doc.longitude IS NOT NULL
        AND ($1 = 'all' OR doc.source_providers @> ARRAY[$1::text])
        AND ($2 = 'all' OR doc.source_kinds @> ARRAY[$2::text])
        AND ($8::text IS NULL OR lower(doc.search_text) LIKE ('%' || lower(trim($8::text)) || '%'))
        AND ($9::text IS NULL OR lower(COALESCE(doc.city, '')) LIKE ('%' || lower(trim($9::text)) || '%'))
        AND ($10::text IS NULL OR NULLIF(regexp_replace(trim(COALESCE(doc.postal, '')), '[^0-9]+', '', 'g'), '') = NULLIF(regexp_replace(trim(COALESCE($10::text, '')), '[^0-9]+', '', 'g'), ''))
        AND ($11::bigint IS NULL OR doc.asking_price >= $11::bigint)
        AND ($12::bigint IS NULL OR doc.asking_price <= $12::bigint)
        AND ($13::double precision IS NULL OR doc.area_m2 >= $13::double precision)
        AND ($14::double precision IS NULL OR doc.area_m2 <= $14::double precision)
        AND ($15::double precision IS NULL OR doc.price_per_m2 >= $15::double precision)
        AND ($16::double precision IS NULL OR doc.price_per_m2 <= $16::double precision)
        AND ($17::integer IS NULL OR doc.rooms_count = $17::integer)
        AND ($18::integer IS NULL OR doc.build_year >= $18::integer)
        AND ($19::integer IS NULL OR doc.build_year <= $19::integer)
        AND ($20::text IS NULL OR doc.property_type_code = $20::text)
        AND ($21::text IS NULL OR lower(COALESCE(doc.condition, '')) LIKE ('%' || lower(trim($21::text)) || '%'))
        AND ($22::text IS NULL OR lower(COALESCE(doc.energy_class, '')) LIKE ('%' || lower(trim($22::text)) || '%'))
        AND ($23::boolean IS NULL OR doc.elevator IS NOT DISTINCT FROM $23::boolean)
        AND ($24::boolean IS NULL OR doc.sauna IS NOT DISTINCT FROM $24::boolean)
        AND ($25::boolean IS NULL OR doc.balcony IS NOT DISTINCT FROM $25::boolean)
        AND ($26::boolean IS NULL OR doc.plot_owned IS NOT DISTINCT FROM $26::boolean)
        AND ($27::boolean IS NULL OR doc.new_development IS NOT DISTINCT FROM $27::boolean)
        AND (
            $28::boolean IS NULL
            OR EXISTS (
                SELECT 1
                FROM public.price_links pl
                WHERE pl.target_type = 'listing'
                    AND pl.target_id = doc.property_offering_id
                    AND pl.link_status <> 'rejected'
            ) IS NOT DISTINCT FROM $28::boolean
        )
        AND (
            $3::double precision IS NULL
            OR postgis.ST_Intersects(
                postgis.ST_SetSRID(postgis.ST_MakePoint(doc.longitude, doc.latitude), 4326),
                postgis.ST_MakeEnvelope($4::double precision, $3::double precision, $6::double precision, $5::double precision, 4326)
            )
        )
),
grouped AS (
    SELECT
        marker_geom,
        marker_key,
        count(DISTINCT property_offering_id)::bigint AS offering_count,
        0::bigint AS housing_company_count,
        min(address) AS address,
        min(city) AS city,
        min(postal) AS postal,
        min(asking_price) AS min_price,
        max(asking_price) AS max_price,
        min(area_m2) AS min_area,
        max(area_m2) AS max_area,
        max(last_seen_at) AS last_seen_at,
        array_agg(DISTINCT source ORDER BY source) AS providers,
        array_agg(DISTINCT kind ORDER BY kind) AS kinds,
        (array_agg(DISTINCT listing_id::text))[1:8] AS listing_ids,
        NULL::text AS housing_company_id
    FROM visible
    GROUP BY marker_geom, marker_key
),
listing_cards AS (
    SELECT
        marker_key,
        jsonb_agg(listing ORDER BY last_seen_at DESC NULLS LAST, asking_price ASC NULLS LAST) AS listings
    FROM (
        SELECT
            visible.marker_key,
            visible.last_seen_at,
            visible.asking_price,
            jsonb_build_object(
                'id', visible.listing_id::text,
                'headline', visible.headline,
                'address', visible.address,
                'city', visible.city,
                'postal', visible.postal,
                'layout', visible.room_layout,
                'area_m2', visible.area_m2,
                'price', visible.asking_price,
                'price_per_m2', visible.price_per_m2,
                'build_year', visible.build_year,
                'last_seen_at', visible.last_seen_at,
                'providers', visible.source_providers,
                'kinds', visible.source_kinds
            ) AS listing,
            row_number() OVER (
                PARTITION BY visible.marker_key
                ORDER BY visible.last_seen_at DESC NULLS LAST, visible.asking_price ASC NULLS LAST
            ) AS listing_rank
        FROM visible
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
        NULLIF(trim(doc.city), '') AS city_norm,
        NULLIF(regexp_replace(trim(COALESCE(doc.postal, '')), '[^0-9]+', '', 'g'), '') AS postal,
        postgis.ST_SetSRID(postgis.ST_MakePoint(doc.longitude, doc.latitude), 4326) AS geom
    FROM public.listing_search_documents doc
    WHERE doc.listing_status = 'active'
        AND doc.latitude IS NOT NULL
        AND doc.longitude IS NOT NULL
        AND ($1 = 'all' OR doc.source_providers @> ARRAY[$1::text])
        AND ($2 = 'all' OR doc.source_kinds @> ARRAY[$2::text])
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
