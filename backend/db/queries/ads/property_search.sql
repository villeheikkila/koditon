-- name: SearchSaleListings :many
SELECT
    pso.sale_listing_source_provider AS source,
    pso.sale_listing_source_kind AS kind,
    pso.sale_listing_native_id AS native_id,
    pso.sale_listing_canonical_id AS canonical_id,
    l.listing_id::text AS public_id,
    pso.sale_listing_url AS url,
    pso.sale_listing_headline AS headline,
    pso.sale_listing_street_address AS address,
    COALESCE(pso.sale_listing_city, pso.sale_listing_city_norm, '') AS city,
    COALESCE(pso.sale_listing_postal, pso.sale_listing_postal_norm, '') AS postal,
    pso.sale_listing_asking_price AS price,
    COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, pso.sale_listing_area_value) AS area,
    COALESCE(NULLIF(unit_profile.dimensions #>> '{layout,room_layout}', ''), pso.sale_listing_room_layout, '') AS room_layout,
    pso.sale_listing_price_per_m2 AS price_per_m2,
    pso.sale_listing_debt_free_price AS debt_free_price,
    pso.sale_listing_debt_share_amount AS debt_share_amount,
    COALESCE((unit_profile.dimensions #>> '{layout,room_count}')::int4, pso.sale_listing_rooms_count) AS rooms_count,
    COALESCE((unit_profile.dimensions #>> '{unit,floor_level}')::int4, pso.sale_listing_floor_level) AS floor_level,
    COALESCE((unit_profile.dimensions #>> '{unit,total_floors}')::int4, (building_profile.dimensions #>> '{building,floor_count}')::int4, pso.sale_listing_total_floors) AS total_floors,
    COALESCE((building_profile.dimensions #>> '{building,build_year}')::int4, (housing_profile.dimensions #>> '{housing_company,build_year}')::int4, hc.housing_company_build_year, pso.sale_listing_build_year) AS build_year,
    COALESCE(NULLIF(unit_profile.dimensions #>> '{condition,unit_condition}', ''), pso.sale_listing_condition) AS condition,
    COALESCE(NULLIF(building_profile.dimensions #>> '{building,energy_class}', ''), NULLIF(housing_profile.dimensions #>> '{housing_company,energy_class}', ''), pso.sale_listing_energy_class) AS energy_class,
    pso.sale_listing_energy_efficiency_label AS energy_efficiency_label,
    pso.sale_listing_last_seen_at::text AS last_seen_at,
    pso.sale_listing_published_at::text AS published_at,
    pso.sale_listing_street_address AS building_key_address,
    source_badges.source_providers
FROM public.listings l
JOIN public.property_source_offerings pso ON pso.sale_listing_id = l.primary_source_listing_id
JOIN public.property_units pu ON pu.property_unit_id = l.unit_id
LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
LEFT JOIN public.dimension_profiles unit_profile ON unit_profile.target_type = 'unit'
    AND unit_profile.target_id = pu.property_unit_id
LEFT JOIN public.dimension_profiles building_profile ON building_profile.target_type = 'building'
    AND building_profile.target_id = pu.physical_building_id
LEFT JOIN public.dimension_profiles housing_profile ON housing_profile.target_type = 'housing_company'
    AND housing_profile.target_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
JOIN LATERAL (
    SELECT array_agg(provider ORDER BY provider)::text[] AS source_providers
    FROM (
        SELECT DISTINCT source_sl.sale_listing_source_provider AS provider
        FROM public.target_sources source_link
        JOIN public.property_source_offerings source_sl ON source_sl.sale_listing_id = source_link.source_id
        WHERE source_link.target_type = 'listing'
            AND source_link.target_id = l.listing_id
            AND source_link.source_type = 'source_listing'
            AND source_link.link_status <> 'rejected'
    ) providers
) source_badges ON true
WHERE EXISTS (
    SELECT 1
    FROM public.target_sources active_link
    WHERE active_link.target_type = 'listing'
        AND active_link.target_id = l.listing_id
        AND active_link.source_type = 'source_listing'
        AND active_link.link_status <> 'rejected'
)
  AND (
    sqlc.arg('source') = 'all'
    OR EXISTS (
        SELECT 1
        FROM public.target_sources source_link
        JOIN public.property_source_offerings source_sl ON source_sl.sale_listing_id = source_link.source_id
        WHERE source_link.target_type = 'listing'
            AND source_link.target_id = l.listing_id
            AND source_link.source_type = 'source_listing'
            AND source_link.link_status <> 'rejected'
            AND source_sl.sale_listing_source_provider = sqlc.arg('source')
    )
  )
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(concat_ws(' ', pso.sale_listing_search_text, pso.sale_listing_description_text)) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city')::text IS NULL OR trim(sqlc.narg('city')::text) = '' OR lower(COALESCE(pso.sale_listing_city, pso.sale_listing_city_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
  AND (sqlc.narg('postal')::text IS NULL OR trim(sqlc.narg('postal')::text) = '' OR lower(COALESCE(pso.sale_listing_postal, pso.sale_listing_postal_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR pso.sale_listing_asking_price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR pso.sale_listing_asking_price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, pso.sale_listing_area_value) >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, pso.sale_listing_area_value) <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR pso.sale_listing_published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR pso.sale_listing_published_at <= sqlc.narg('published_before')::timestamptz)
  AND (sqlc.narg('min_price_per_m2')::float8 IS NULL OR pso.sale_listing_price_per_m2 >= sqlc.narg('min_price_per_m2')::float8)
  AND (sqlc.narg('max_price_per_m2')::float8 IS NULL OR pso.sale_listing_price_per_m2 <= sqlc.narg('max_price_per_m2')::float8)
  AND (sqlc.narg('rooms')::int4 IS NULL OR COALESCE((unit_profile.dimensions #>> '{layout,room_count}')::int4, pso.sale_listing_rooms_count) = sqlc.narg('rooms')::int4)
  AND (sqlc.narg('floor')::int4 IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,floor_level}')::int4, pso.sale_listing_floor_level) = sqlc.narg('floor')::int4)
  AND (sqlc.narg('min_build_year')::int4 IS NULL OR COALESCE((building_profile.dimensions #>> '{building,build_year}')::int4, (housing_profile.dimensions #>> '{housing_company,build_year}')::int4, hc.housing_company_build_year, pso.sale_listing_build_year) >= sqlc.narg('min_build_year')::int4)
  AND (sqlc.narg('max_build_year')::int4 IS NULL OR COALESCE((building_profile.dimensions #>> '{building,build_year}')::int4, (housing_profile.dimensions #>> '{housing_company,build_year}')::int4, hc.housing_company_build_year, pso.sale_listing_build_year) <= sqlc.narg('max_build_year')::int4)
  AND (sqlc.narg('condition')::text IS NULL OR trim(sqlc.narg('condition')::text) = '' OR lower(COALESCE(COALESCE(NULLIF(unit_profile.dimensions #>> '{condition,unit_condition}', ''), pso.sale_listing_condition), '')) LIKE ('%' || lower(trim(sqlc.narg('condition')::text)) || '%'))
  AND (sqlc.narg('energy_class')::text IS NULL OR trim(sqlc.narg('energy_class')::text) = '' OR lower(COALESCE(COALESCE(NULLIF(building_profile.dimensions #>> '{building,energy_class}', ''), NULLIF(housing_profile.dimensions #>> '{housing_company,energy_class}', ''), pso.sale_listing_energy_class), '')) LIKE ('%' || lower(trim(sqlc.narg('energy_class')::text)) || '%'))
  AND (sqlc.arg('kind') = 'all' OR pso.sale_listing_source_kind = sqlc.arg('kind'))
ORDER BY
    CASE WHEN sqlc.arg('sort_mode') = 'price_asc' THEN pso.sale_listing_asking_price END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_desc' THEN pso.sale_listing_asking_price END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'area_asc' THEN COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, pso.sale_listing_area_value) END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'area_desc' THEN COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, pso.sale_listing_area_value) END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_m2_asc' THEN pso.sale_listing_price_per_m2 END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_m2_desc' THEN pso.sale_listing_price_per_m2 END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'build_year_desc' THEN COALESCE((building_profile.dimensions #>> '{building,build_year}')::int4, (housing_profile.dimensions #>> '{housing_company,build_year}')::int4, hc.housing_company_build_year, pso.sale_listing_build_year) END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'seen_desc' THEN pso.sale_listing_last_seen_at END DESC NULLS LAST,
    pso.sale_listing_last_seen_at DESC
LIMIT sqlc.arg('limit_count')::int OFFSET sqlc.arg('offset_count')::int;

-- name: SearchRentalListings :many
WITH unified AS (
    SELECT 'shortcut'::text AS source, 'ad'::text AS kind, sa.shortcut_ad_id::text AS native_id, ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id, sa.shortcut_ad_url AS url, COALESCE(raw.street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text) AS headline, COALESCE(raw.street_address, sb.shortcut_building_address) AS address, raw.city, raw.postal, raw.price, raw.area, sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout, sa.shortcut_ad_last_seen_at AS last_seen_at, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable
    FROM origin.shortcut_ads sa
    LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    CROSS JOIN LATERAL (
        SELECT
            COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerDay}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area
    ) raw
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT 'frontdoor'::text AS source, 'announcement'::text AS kind, fba.frontdoor_building_announcement_id::text AS native_id, ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id, fb.frontdoor_building_url AS url, COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text) AS headline, concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS address, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, fba.frontdoor_building_announcement_room_structure AS room_layout, fba.frontdoor_building_announcement_last_seen_at AS last_seen_at, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM origin.frontdoor_building_announcements fba
    JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT
    source,
    kind,
    native_id,
    canonical_id,
    ('r_' || substr(md5(canonical_id), 1, 16)) AS public_id,
    url,
    headline,
    address,
    city,
    postal,
    price,
    area,
    room_layout,
    NULL::float8 AS price_per_m2,
    NULL::bigint AS debt_free_price,
    NULL::bigint AS debt_share_amount,
    NULL::int4 AS rooms_count,
    NULL::int4 AS floor_level,
    NULL::int4 AS total_floors,
    NULL::int4 AS build_year,
    NULL::text AS condition,
    NULL::text AS energy_class,
    NULL::text AS energy_efficiency_label,
    last_seen_at::text AS last_seen_at,
    published_at::text AS published_at,
    address AS building_key_address,
    ARRAY[source]::text[] AS source_providers
FROM unified u
WHERE (sqlc.arg('source') = 'all' OR u.source = sqlc.arg('source'))
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city')::text IS NULL OR trim(sqlc.narg('city')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
  AND (sqlc.narg('postal')::text IS NULL OR trim(sqlc.narg('postal')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz)
ORDER BY
    CASE WHEN sqlc.arg('sort_mode') = 'price_asc' THEN price END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'price_desc' THEN price END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'area_asc' THEN area END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'area_desc' THEN area END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode') = 'seen_desc' THEN last_seen_at END DESC NULLS LAST,
    last_seen_at DESC
LIMIT sqlc.arg('limit_count')::int OFFSET sqlc.arg('offset_count')::int;

-- name: CountSaleListings :one
SELECT count(*)::bigint
FROM public.listings l
JOIN public.property_source_offerings sl ON sl.sale_listing_id = l.primary_source_listing_id
JOIN public.property_units pu ON pu.property_unit_id = l.unit_id
LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
LEFT JOIN public.dimension_profiles unit_profile ON unit_profile.target_type = 'unit'
    AND unit_profile.target_id = pu.property_unit_id
LEFT JOIN public.dimension_profiles building_profile ON building_profile.target_type = 'building'
    AND building_profile.target_id = pu.physical_building_id
LEFT JOIN public.dimension_profiles housing_profile ON housing_profile.target_type = 'housing_company'
    AND housing_profile.target_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
WHERE EXISTS (
    SELECT 1
    FROM public.target_sources active_link
    WHERE active_link.target_type = 'listing'
        AND active_link.target_id = l.listing_id
        AND active_link.source_type = 'source_listing'
        AND active_link.link_status <> 'rejected'
)
  AND (
    sqlc.arg('source') = 'all'
    OR EXISTS (
        SELECT 1
        FROM public.target_sources source_link
        JOIN public.property_source_offerings source_sl ON source_sl.sale_listing_id = source_link.source_id
        WHERE source_link.target_type = 'listing'
            AND source_link.target_id = l.listing_id
            AND source_link.source_type = 'source_listing'
            AND source_link.link_status <> 'rejected'
            AND source_sl.sale_listing_source_provider = sqlc.arg('source')
    )
  )
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(concat_ws(' ', sl.sale_listing_search_text, sl.sale_listing_description_text)) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city')::text IS NULL OR trim(sqlc.narg('city')::text) = '' OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
  AND (sqlc.narg('postal')::text IS NULL OR trim(sqlc.narg('postal')::text) = '' OR lower(COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR sl.sale_listing_asking_price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR sl.sale_listing_asking_price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, sl.sale_listing_area_value) >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,area_m2}')::float8, sl.sale_listing_area_value) <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR sl.sale_listing_published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR sl.sale_listing_published_at <= sqlc.narg('published_before')::timestamptz)
  AND (sqlc.narg('min_price_per_m2')::float8 IS NULL OR sl.sale_listing_price_per_m2 >= sqlc.narg('min_price_per_m2')::float8)
  AND (sqlc.narg('max_price_per_m2')::float8 IS NULL OR sl.sale_listing_price_per_m2 <= sqlc.narg('max_price_per_m2')::float8)
  AND (sqlc.narg('rooms')::int4 IS NULL OR COALESCE((unit_profile.dimensions #>> '{layout,room_count}')::int4, sl.sale_listing_rooms_count) = sqlc.narg('rooms')::int4)
  AND (sqlc.narg('floor')::int4 IS NULL OR COALESCE((unit_profile.dimensions #>> '{unit,floor_level}')::int4, sl.sale_listing_floor_level) = sqlc.narg('floor')::int4)
  AND (sqlc.narg('min_build_year')::int4 IS NULL OR COALESCE((building_profile.dimensions #>> '{building,build_year}')::int4, (housing_profile.dimensions #>> '{housing_company,build_year}')::int4, hc.housing_company_build_year, sl.sale_listing_build_year) >= sqlc.narg('min_build_year')::int4)
  AND (sqlc.narg('max_build_year')::int4 IS NULL OR COALESCE((building_profile.dimensions #>> '{building,build_year}')::int4, (housing_profile.dimensions #>> '{housing_company,build_year}')::int4, hc.housing_company_build_year, sl.sale_listing_build_year) <= sqlc.narg('max_build_year')::int4)
  AND (sqlc.narg('condition')::text IS NULL OR trim(sqlc.narg('condition')::text) = '' OR lower(COALESCE(COALESCE(NULLIF(unit_profile.dimensions #>> '{condition,unit_condition}', ''), sl.sale_listing_condition), '')) LIKE ('%' || lower(trim(sqlc.narg('condition')::text)) || '%'))
  AND (sqlc.narg('energy_class')::text IS NULL OR trim(sqlc.narg('energy_class')::text) = '' OR lower(COALESCE(COALESCE(NULLIF(building_profile.dimensions #>> '{building,energy_class}', ''), NULLIF(housing_profile.dimensions #>> '{housing_company,energy_class}', ''), sl.sale_listing_energy_class), '')) LIKE ('%' || lower(trim(sqlc.narg('energy_class')::text)) || '%'))
  AND (sqlc.arg('kind') = 'all' OR sl.sale_listing_source_kind = sqlc.arg('kind'));

-- name: CountRentalListings :one
WITH unified AS (
    SELECT 'shortcut'::text AS source, raw.city, raw.postal, raw.price, raw.area, (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at, trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable
    FROM origin.shortcut_ads sa
    LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    CROSS JOIN LATERAL (
        SELECT
            COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerDay}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area
    ) raw
    WHERE sa.shortcut_ad_type = 'rental'
    UNION ALL
    SELECT 'frontdoor'::text AS source, COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city, fb.frontdoor_building_postcode AS postal, CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price, fba.frontdoor_building_announcement_area AS area, NULL::timestamptz AS published_at, concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
    FROM origin.frontdoor_building_announcements fba
    JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL
)
SELECT count(*)::bigint
FROM unified u
WHERE (sqlc.arg('source') = 'all' OR u.source = sqlc.arg('source'))
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city')::text IS NULL OR trim(sqlc.narg('city')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
  AND (sqlc.narg('postal')::text IS NULL OR trim(sqlc.narg('postal')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz);

-- name: ListRentalCanonicalIDs :many
SELECT ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id
FROM origin.shortcut_ads sa
WHERE sa.shortcut_ad_type = 'rental'
UNION ALL
SELECT ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id
FROM origin.frontdoor_building_announcements fba
WHERE fba.frontdoor_building_announcement_rent_period IS NOT NULL OR fba.frontdoor_building_announcement_rental_unique_no IS NOT NULL;

-- name: ListBuildingCanonicalIDs :many
SELECT ('shortcut:building:' || sb.shortcut_building_id::text) AS canonical_id
FROM origin.shortcut_buildings sb
UNION ALL
SELECT ('frontdoor:building:' || fb.frontdoor_building_id::text) AS canonical_id
FROM origin.frontdoor_buildings fb;
