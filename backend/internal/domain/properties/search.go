package properties

import (
	"context"
	"encoding/json"
	"fmt"
)

const searchSaleListingsSQL = `
SELECT
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.sale_listing_canonical_id,
    po.property_offering_id::text,
    COALESCE(sl.sale_listing_url, ''),
    COALESCE(sl.sale_listing_headline, ''),
    COALESCE(sl.sale_listing_street_address, ''),
    COALESCE(sl.sale_listing_city, ''),
    COALESCE(sl.sale_listing_postal, ''),
    sl.sale_listing_asking_price,
    sl.sale_listing_area_value,
    COALESCE(sl.sale_listing_room_layout, ''),
    sl.sale_listing_price_per_m2,
    sl.sale_listing_debt_free_price,
    sl.sale_listing_debt_share_amount,
    sl.sale_listing_rooms_count,
    sl.sale_listing_floor_level,
    sl.sale_listing_total_floors,
    sl.sale_listing_build_year,
    sl.sale_listing_condition,
    sl.sale_listing_energy_class,
    sl.sale_listing_energy_efficiency_label,
    sl.sale_listing_last_seen_at::text,
    sl.sale_listing_published_at::text,
    COALESCE(sl.sale_listing_street_address, ''),
    source_badges.source_providers
FROM public.property_offerings po
JOIN public.property_source_offerings sl ON sl.sale_listing_id = po.primary_sale_listing_id
JOIN LATERAL (
    SELECT array_agg(provider ORDER BY provider)::text[] AS source_providers
    FROM (
        SELECT DISTINCT source_sl.sale_listing_source_provider AS provider
        FROM public.property_offering_sources source_pos
        JOIN public.property_source_offerings source_sl ON source_sl.sale_listing_id = source_pos.sale_listing_id
        WHERE source_pos.property_offering_id = po.property_offering_id
            AND source_pos.property_offering_source_link_status <> 'rejected'
    ) providers
) source_badges ON true
WHERE EXISTS (
    SELECT 1
    FROM public.property_offering_sources active_pos
    WHERE active_pos.property_offering_id = po.property_offering_id
        AND active_pos.property_offering_source_link_status <> 'rejected'
)
  AND (
    $4 = 'all'
    OR EXISTS (
        SELECT 1
        FROM public.property_offering_sources source_pos
        JOIN public.property_source_offerings source_sl ON source_sl.sale_listing_id = source_pos.sale_listing_id
        WHERE source_pos.property_offering_id = po.property_offering_id
            AND source_pos.property_offering_source_link_status <> 'rejected'
            AND source_sl.sale_listing_source_provider = $4
    )
  )
  AND ($5::text IS NULL OR trim($5::text) = '' OR lower(concat_ws(' ', sl.sale_listing_search_text, sl.sale_listing_description_text)) LIKE ('%' || lower(trim($5::text)) || '%'))
  AND ($6::text IS NULL OR trim($6::text) = '' OR lower(COALESCE(sl.sale_listing_city, '')) LIKE ('%' || lower(trim($6::text)) || '%'))
  AND ($7::text IS NULL OR trim($7::text) = '' OR lower(COALESCE(sl.sale_listing_postal, '')) LIKE ('%' || lower(trim($7::text)) || '%'))
  AND ($8::bigint IS NULL OR sl.sale_listing_asking_price >= $8::bigint)
  AND ($9::bigint IS NULL OR sl.sale_listing_asking_price <= $9::bigint)
  AND ($10::float8 IS NULL OR sl.sale_listing_area_value >= $10::float8)
  AND ($11::float8 IS NULL OR sl.sale_listing_area_value <= $11::float8)
  AND ($12::timestamptz IS NULL OR sl.sale_listing_published_at >= $12::timestamptz)
  AND ($13::timestamptz IS NULL OR sl.sale_listing_published_at <= $13::timestamptz)
  AND ($14::float8 IS NULL OR sl.sale_listing_price_per_m2 >= $14::float8)
  AND ($15::float8 IS NULL OR sl.sale_listing_price_per_m2 <= $15::float8)
  AND ($16::int4 IS NULL OR sl.sale_listing_rooms_count = $16::int4)
  AND ($17::int4 IS NULL OR sl.sale_listing_floor_level = $17::int4)
  AND ($18::int4 IS NULL OR sl.sale_listing_build_year >= $18::int4)
  AND ($19::int4 IS NULL OR sl.sale_listing_build_year <= $19::int4)
  AND ($20::text IS NULL OR trim($20::text) = '' OR lower(COALESCE(sl.sale_listing_condition, '')) LIKE ('%' || lower(trim($20::text)) || '%'))
  AND ($21::text IS NULL OR trim($21::text) = '' OR lower(COALESCE(sl.sale_listing_energy_class, '')) LIKE ('%' || lower(trim($21::text)) || '%'))
  AND ($22 = 'all' OR sl.sale_listing_source_kind = $22)
ORDER BY
    CASE WHEN $1 = 'price_asc' THEN sl.sale_listing_asking_price END ASC NULLS LAST,
    CASE WHEN $1 = 'price_desc' THEN sl.sale_listing_asking_price END DESC NULLS LAST,
    CASE WHEN $1 = 'area_asc' THEN sl.sale_listing_area_value END ASC NULLS LAST,
    CASE WHEN $1 = 'area_desc' THEN sl.sale_listing_area_value END DESC NULLS LAST,
    CASE WHEN $1 = 'price_m2_asc' THEN sl.sale_listing_price_per_m2 END ASC NULLS LAST,
    CASE WHEN $1 = 'price_m2_desc' THEN sl.sale_listing_price_per_m2 END DESC NULLS LAST,
    CASE WHEN $1 = 'build_year_desc' THEN sl.sale_listing_build_year END DESC NULLS LAST,
    CASE WHEN $1 = 'seen_desc' THEN sl.sale_listing_last_seen_at END DESC NULLS LAST,
    sl.sale_listing_last_seen_at DESC
LIMIT $3::int OFFSET $2::int`

const searchRentalsSQL = `
WITH unified AS (
    SELECT 'shortcut'::text AS source, 'ad'::text AS kind, sa.shortcut_ad_id::text AS native_id, ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id, sa.shortcut_ad_url AS url, COALESCE(raw.street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text) AS headline, COALESCE(raw.street_address, sb.shortcut_building_address) AS address, raw.city, raw.postal, raw.price, raw.area, sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout, sa.shortcut_ad_last_seen_at AS last_seen_at, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    CROSS JOIN LATERAL (
        SELECT
            public.fnc__shortcut_ad_street_address(sa.shortcut_ad_data) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerDay}')) AS price,
            COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')) AS area
    ) raw
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT 'frontdoor'::text AS source, 'announcement'::text AS kind, fba.frontdoor_building_announcement_id::text AS native_id, ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id, fb.frontdoor_building_url AS url, COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text) AS headline, concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS address, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, fba.frontdoor_building_announcement_room_structure AS room_layout, fba.frontdoor_building_announcement_last_seen_at AS last_seen_at, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT source, kind, native_id, canonical_id, ('r_' || substr(md5(canonical_id), 1, 16)) AS public_id, url, headline, address, city, postal, price, area, room_layout, NULL::float8, NULL::bigint, NULL::bigint, NULL::int4, NULL::int4, NULL::int4, NULL::int4, NULL::text, NULL::text, NULL::text, last_seen_at::text, published_at::text, address, ARRAY[source]::text[]
FROM unified u
WHERE ($4 = 'all' OR u.source = $4)
  AND ($5::text IS NULL OR trim($5::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim($5::text)) || '%'))
  AND ($6::text IS NULL OR trim($6::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim($6::text)) || '%'))
  AND ($7::text IS NULL OR trim($7::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim($7::text)) || '%'))
  AND ($8::bigint IS NULL OR u.price >= $8::bigint)
  AND ($9::bigint IS NULL OR u.price <= $9::bigint)
  AND ($10::float8 IS NULL OR u.area >= $10::float8)
  AND ($11::float8 IS NULL OR u.area <= $11::float8)
  AND ($12::timestamptz IS NULL OR u.published_at >= $12::timestamptz)
  AND ($13::timestamptz IS NULL OR u.published_at <= $13::timestamptz)
ORDER BY
    CASE WHEN $1 = 'price_asc' THEN price END ASC NULLS LAST,
    CASE WHEN $1 = 'price_desc' THEN price END DESC NULLS LAST,
    CASE WHEN $1 = 'area_asc' THEN area END ASC NULLS LAST,
    CASE WHEN $1 = 'area_desc' THEN area END DESC NULLS LAST,
    CASE WHEN $1 = 'seen_desc' THEN last_seen_at END DESC NULLS LAST,
    last_seen_at DESC
LIMIT $3::int OFFSET $2::int`

const countSaleListingsSQL = `
SELECT count(*)::bigint
FROM public.property_offerings po
JOIN public.property_source_offerings sl ON sl.sale_listing_id = po.primary_sale_listing_id
WHERE EXISTS (
    SELECT 1
    FROM public.property_offering_sources active_pos
    WHERE active_pos.property_offering_id = po.property_offering_id
        AND active_pos.property_offering_source_link_status <> 'rejected'
)
  AND (
    $1 = 'all'
    OR EXISTS (
        SELECT 1
        FROM public.property_offering_sources source_pos
        JOIN public.property_source_offerings source_sl ON source_sl.sale_listing_id = source_pos.sale_listing_id
        WHERE source_pos.property_offering_id = po.property_offering_id
            AND source_pos.property_offering_source_link_status <> 'rejected'
            AND source_sl.sale_listing_source_provider = $1
    )
  )
  AND ($2::text IS NULL OR trim($2::text) = '' OR lower(concat_ws(' ', sl.sale_listing_search_text, sl.sale_listing_description_text)) LIKE ('%' || lower(trim($2::text)) || '%'))
  AND ($3::text IS NULL OR trim($3::text) = '' OR lower(COALESCE(sl.sale_listing_city, '')) LIKE ('%' || lower(trim($3::text)) || '%'))
  AND ($4::text IS NULL OR trim($4::text) = '' OR lower(COALESCE(sl.sale_listing_postal, '')) LIKE ('%' || lower(trim($4::text)) || '%'))
  AND ($5::bigint IS NULL OR sl.sale_listing_asking_price >= $5::bigint)
  AND ($6::bigint IS NULL OR sl.sale_listing_asking_price <= $6::bigint)
  AND ($7::float8 IS NULL OR sl.sale_listing_area_value >= $7::float8)
  AND ($8::float8 IS NULL OR sl.sale_listing_area_value <= $8::float8)
  AND ($9::timestamptz IS NULL OR sl.sale_listing_published_at >= $9::timestamptz)
  AND ($10::timestamptz IS NULL OR sl.sale_listing_published_at <= $10::timestamptz)
  AND ($11::float8 IS NULL OR sl.sale_listing_price_per_m2 >= $11::float8)
  AND ($12::float8 IS NULL OR sl.sale_listing_price_per_m2 <= $12::float8)
  AND ($13::int4 IS NULL OR sl.sale_listing_rooms_count = $13::int4)
  AND ($14::int4 IS NULL OR sl.sale_listing_floor_level = $14::int4)
  AND ($15::int4 IS NULL OR sl.sale_listing_build_year >= $15::int4)
  AND ($16::int4 IS NULL OR sl.sale_listing_build_year <= $16::int4)
  AND ($17::text IS NULL OR trim($17::text) = '' OR lower(COALESCE(sl.sale_listing_condition, '')) LIKE ('%' || lower(trim($17::text)) || '%'))
  AND ($18::text IS NULL OR trim($18::text) = '' OR lower(COALESCE(sl.sale_listing_energy_class, '')) LIKE ('%' || lower(trim($18::text)) || '%'))
  AND ($19 = 'all' OR sl.sale_listing_source_kind = $19)`

const countRentalsSQL = `
WITH unified AS (
    SELECT 'shortcut'::text AS source, raw.city, raw.postal, raw.price, raw.area, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    CROSS JOIN LATERAL (
        SELECT
            public.fnc__shortcut_ad_street_address(sa.shortcut_ad_data) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerDay}')) AS price,
            COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')) AS area
    ) raw
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT 'frontdoor'::text AS source, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT count(*)::bigint
FROM unified u
WHERE ($1 = 'all' OR u.source = $1)
  AND ($2::text IS NULL OR trim($2::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim($2::text)) || '%'))
  AND ($3::text IS NULL OR trim($3::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim($3::text)) || '%'))
  AND ($4::text IS NULL OR trim($4::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim($4::text)) || '%'))
  AND ($5::bigint IS NULL OR u.price >= $5::bigint)
  AND ($6::bigint IS NULL OR u.price <= $6::bigint)
  AND ($7::float8 IS NULL OR u.area >= $7::float8)
  AND ($8::float8 IS NULL OR u.area <= $8::float8)
  AND ($9::timestamptz IS NULL OR u.published_at >= $9::timestamptz)
  AND ($10::timestamptz IS NULL OR u.published_at <= $10::timestamptz)`

const resolveRentalPublicIDSQL = `
WITH unified AS (
    SELECT ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id
    FROM public.shortcut_ads sa
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id
    FROM public.frontdoor_building_announcements fba
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT canonical_id
FROM unified
WHERE ('r_' || substr(md5(canonical_id), 1, 16)) = $1
LIMIT 1`

const resolveBuildingPublicIDSQL = `
WITH unified AS (
    SELECT ('shortcut:building:' || sb.shortcut_building_id::text) AS canonical_id
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT ('frontdoor:building:' || fb.frontdoor_building_id::text) AS canonical_id
    FROM public.frontdoor_buildings fb
)
SELECT canonical_id
FROM unified
WHERE ('b_' || substr(md5(canonical_id), 1, 16)) = $1
LIMIT 1`

func (s *Service) searchListings(ctx context.Context, params SearchParams, listingType string) ([]listingSearchRow, error) {
	query := searchSaleListingsSQL
	args := []any{params.Sort, (params.Page - 1) * params.PageSize, params.PageSize, params.Source, emptyToNil(params.Query), emptyToNil(params.City), emptyToNil(params.Postal), params.MinPrice, params.MaxPrice, params.MinArea, params.MaxArea, params.PublishedAfter, params.PublishedBefore, params.MinPricePerM2, params.MaxPricePerM2, params.Rooms, params.Floor, params.MinBuildYear, params.MaxBuildYear, emptyToNil(params.Condition), emptyToNil(params.EnergyClass), params.Kind}
	if listingType == "rental" {
		query = searchRentalsSQL
		args = args[:13]
	}
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search %s listings: %w", listingType, err)
	}
	defer rows.Close()
	out := []listingSearchRow{}
	for rows.Next() {
		var row listingSearchRow
		if err := rows.Scan(&row.Source, &row.Kind, &row.NativeID, &row.CanonicalID, &row.PublicID, &row.URL, &row.Headline, &row.Address, &row.City, &row.Postal, &row.Price, &row.Area, &row.RoomLayout, &row.PricePerM2, &row.DebtFreePrice, &row.DebtShareAmount, &row.RoomsCount, &row.FloorLevel, &row.TotalFloors, &row.BuildYear, &row.Condition, &row.EnergyClass, &row.EnergyEfficiencyLabel, &row.LastSeenAt, &row.PublishedAt, &row.BuildingKeyAddress, &row.SourceProviders); err != nil {
			return nil, fmt.Errorf("scan %s listing: %w", listingType, err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s listings: %w", listingType, err)
	}
	return out, nil
}

func (s *Service) countListings(ctx context.Context, params SearchParams, listingType string) (int64, error) {
	query := countSaleListingsSQL
	args := []any{params.Source, emptyToNil(params.Query), emptyToNil(params.City), emptyToNil(params.Postal), params.MinPrice, params.MaxPrice, params.MinArea, params.MaxArea, params.PublishedAfter, params.PublishedBefore, params.MinPricePerM2, params.MaxPricePerM2, params.Rooms, params.Floor, params.MinBuildYear, params.MaxBuildYear, emptyToNil(params.Condition), emptyToNil(params.EnergyClass), params.Kind}
	if listingType == "rental" {
		query = countRentalsSQL
		args = args[:10]
	}
	var count int64
	err := s.db.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count %s listings: %w", listingType, err)
	}
	return count, nil
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
        pb.housing_company_build_year,
        po.property_offering_id,
        po.property_offering_headline,
        po.property_offering_asking_price,
        po.property_offering_price_per_m2,
        pu.property_unit_area_value,
        pu.property_unit_room_layout,
        po.property_offering_last_seen_at,
        sl.sale_listing_street_address,
        sl.sale_listing_city,
        sl.sale_listing_postal,
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
    JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
        AND pos.property_offering_source_link_status <> 'rejected'
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    WHERE pb.housing_company_geom IS NOT NULL
        AND ($1 = 'all' OR sl.sale_listing_source_provider = $1)
        AND ($2 = 'all' OR sl.sale_listing_source_kind = $2)
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
LIMIT $7::int`, source, kind, bounds.MinLat, bounds.MinLng, bounds.MaxLat, bounds.MaxLng, limit)
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
