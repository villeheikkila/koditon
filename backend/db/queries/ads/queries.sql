-- name: SearchUnifiedEntities :many
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        sa.shortcut_ad_id::text AS native_id,
        ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id,
        COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text) AS headline,
        COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address) AS address,
        sa.shortcut_ad_city AS city,
        sa.shortcut_ad_postal AS postal,
        sa.shortcut_ad_price AS price,
        COALESCE(sa.shortcut_ad_area_value, 0::float8) AS area,
        sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout,
        sa.shortcut_ad_url AS url,
        sa.shortcut_ad_last_seen_at AS last_seen_at,
        concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable,
        sa.shortcut_ad_type AS listing_type,
        (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    UNION ALL
    SELECT
        'shortcut'::text AS source,
        'building'::text AS kind,
        sb.shortcut_building_id::text AS native_id,
        ('shortcut:building:' || sb.shortcut_building_id::text) AS canonical_id,
        COALESCE(sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_external_id::text) AS headline,
        sb.shortcut_building_address AS address,
        NULL::text AS city,
        NULL::text AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        NULL::text AS room_layout,
        sb.shortcut_building_url AS url,
        COALESCE(sb.shortcut_building_updated_at, sb.shortcut_building_processed_at, now()) AS last_seen_at,
        concat_ws(' ', sb.shortcut_building_id::text, sb.shortcut_building_external_id::text, sb.shortcut_building_url, sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_building_type, sb.shortcut_building_building_subtype) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        fa.frontdoor_ad_external_id AS native_id,
        ('frontdoor:ad:' || fa.frontdoor_ad_external_id) AS canonical_id,
        COALESCE(fa.frontdoor_ad_street_address, fa.frontdoor_ad_external_id) AS headline,
        fa.frontdoor_ad_street_address AS address,
        fa.frontdoor_ad_city AS city,
        fa.frontdoor_ad_postal AS postal,
        fa.frontdoor_ad_price AS price,
        COALESCE(fa.frontdoor_ad_area_value, 0::float8) AS area,
        fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS room_layout,
        fa.frontdoor_ad_url AS url,
        fa.frontdoor_ad_last_seen_at AS last_seen_at,
        fa.frontdoor_ad_search_text AS searchable,
        NULL::text AS listing_type,
        fa.frontdoor_ad_publishing_time AS published_at
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        fba.frontdoor_building_announcement_id::text AS native_id,
        ('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text) AS canonical_id,
        COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text) AS headline,
        concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS address,
        COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price,
        COALESCE(fba.frontdoor_building_announcement_area, 0::float8) AS area,
        fba.frontdoor_building_announcement_room_structure AS room_layout,
        fb.frontdoor_building_url AS url,
        fba.frontdoor_building_announcement_last_seen_at AS last_seen_at,
        concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'building'::text AS kind,
        fb.frontdoor_building_id::text AS native_id,
        ('frontdoor:building:' || fb.frontdoor_building_id::text) AS canonical_id,
        COALESCE(fb.frontdoor_building_company_name, concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number), fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_id::text) AS headline,
        concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number) AS address,
        COALESCE(fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        NULL::text AS room_layout,
        fb.frontdoor_building_url AS url,
        COALESCE(fb.frontdoor_building_last_seen_at, now()) AS last_seen_at,
        concat_ws(' ', fb.frontdoor_building_id::text, fb.frontdoor_building_url, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_company_name, fb.frontdoor_building_street_address, fb.frontdoor_building_house_number, fb.frontdoor_building_postcode, fb.frontdoor_building_post_area, fb.frontdoor_building_municipality) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.frontdoor_buildings fb
), filtered AS (
    SELECT *
    FROM unified u
    WHERE (sqlc.arg(source_filter) = 'all' OR u.source = sqlc.arg(source_filter))
      AND (sqlc.arg(kind_filter) = 'all' OR u.kind = sqlc.arg(kind_filter))
      AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
      AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
      AND (sqlc.narg('postal_filter')::text IS NULL OR trim(sqlc.narg('postal_filter')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal_filter')::text)) || '%'))
      AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
      AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
      AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
      AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
      AND (sqlc.narg('listing_type_filter')::text IS NULL OR u.listing_type IS NULL OR u.listing_type = sqlc.narg('listing_type_filter')::text)
      AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
      AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz)
)
SELECT
    source,
    kind,
    native_id,
    canonical_id,
    headline,
    address,
    city,
    postal,
    price,
    area,
    room_layout,
    url,
    last_seen_at
FROM filtered
ORDER BY
    CASE WHEN sqlc.arg(sort_mode) = 'price_asc' THEN price END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'price_desc' THEN price END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'area_asc' THEN area END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'area_desc' THEN area END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'seen_desc' THEN last_seen_at END DESC NULLS LAST,
    last_seen_at DESC,
    source,
    kind,
    native_id
LIMIT sqlc.arg(limit_count)::int
OFFSET sqlc.arg(offset_count)::int;

-- name: CountUnifiedEntities :one
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        sa.shortcut_ad_city AS city,
        sa.shortcut_ad_postal AS postal,
        sa.shortcut_ad_price AS price,
        COALESCE(sa.shortcut_ad_area_value, 0::float8) AS area,
        concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable,
        sa.shortcut_ad_type AS listing_type,
        (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    UNION ALL
    SELECT
        'shortcut'::text AS source,
        'building'::text AS kind,
        NULL::text AS city,
        NULL::text AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        concat_ws(' ', sb.shortcut_building_id::text, sb.shortcut_building_external_id::text, sb.shortcut_building_url, sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_building_type, sb.shortcut_building_building_subtype) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        fa.frontdoor_ad_city AS city,
        fa.frontdoor_ad_postal AS postal,
        fa.frontdoor_ad_price AS price,
        COALESCE(fa.frontdoor_ad_area_value, 0::float8) AS area,
        fa.frontdoor_ad_search_text AS searchable,
        NULL::text AS listing_type,
        fa.frontdoor_ad_publishing_time AS published_at
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price,
        COALESCE(fba.frontdoor_building_announcement_area, 0::float8) AS area,
        concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'building'::text AS kind,
        COALESCE(fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        concat_ws(' ', fb.frontdoor_building_id::text, fb.frontdoor_building_url, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_company_name, fb.frontdoor_building_street_address, fb.frontdoor_building_house_number, fb.frontdoor_building_postcode, fb.frontdoor_building_post_area, fb.frontdoor_building_municipality) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.frontdoor_buildings fb
)
SELECT COUNT(*)::bigint AS count
FROM unified u
WHERE (sqlc.arg(source_filter) = 'all' OR u.source = sqlc.arg(source_filter))
  AND (sqlc.arg(kind_filter) = 'all' OR u.kind = sqlc.arg(kind_filter))
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
  AND (sqlc.narg('postal_filter')::text IS NULL OR trim(sqlc.narg('postal_filter')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal_filter')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('listing_type_filter')::text IS NULL OR u.listing_type IS NULL OR u.listing_type = sqlc.narg('listing_type_filter')::text)
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz);

-- name: FindCrossSourceAdMatches :many
SELECT
    sa.shortcut_ad_id,
    fa.frontdoor_ad_external_id,
    sa.shortcut_ad_address_key AS address_key,
    sa.shortcut_ad_street_address AS shortcut_street,
    fa.frontdoor_ad_street_address AS frontdoor_street,
    sa.shortcut_ad_postal AS shortcut_postal,
    fa.frontdoor_ad_postal AS frontdoor_postal,
    sa.shortcut_ad_city AS shortcut_city,
    fa.frontdoor_ad_city AS frontdoor_city,
    sa.shortcut_ad_price AS shortcut_price,
    fa.frontdoor_ad_price AS frontdoor_price,
    sa.shortcut_ad_area_value AS shortcut_area,
    fa.frontdoor_ad_area_value AS frontdoor_area
FROM public.shortcut_ads sa
JOIN public.frontdoor_ads fa ON sa.shortcut_ad_address_key = fa.frontdoor_ad_address_key
WHERE sa.shortcut_ad_address_key IS NOT NULL
  AND sa.shortcut_ad_address_key <> ''
  AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(sa.shortcut_ad_city, fa.frontdoor_ad_city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
  AND (
      sqlc.narg('max_price_delta')::bigint IS NULL
      OR (
          sa.shortcut_ad_price IS NOT NULL
          AND fa.frontdoor_ad_price IS NOT NULL
          AND abs(sa.shortcut_ad_price - fa.frontdoor_ad_price) <= sqlc.narg('max_price_delta')::bigint
      )
  )
  AND (
      sqlc.narg('max_area_delta')::float8 IS NULL
      OR (
          sa.shortcut_ad_area_value IS NOT NULL
          AND fa.frontdoor_ad_area_value IS NOT NULL
          AND abs(sa.shortcut_ad_area_value - fa.frontdoor_ad_area_value) <= sqlc.narg('max_area_delta')::float8
      )
  )
ORDER BY
    abs(COALESCE(sa.shortcut_ad_price, 0) - COALESCE(fa.frontdoor_ad_price, 0)) ASC,
    abs(COALESCE(sa.shortcut_ad_area_value, 0) - COALESCE(fa.frontdoor_ad_area_value, 0)) ASC,
    sa.shortcut_ad_last_seen_at DESC,
    fa.frontdoor_ad_last_seen_at DESC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetShortcutAdUnifiedDetail :one
SELECT
    sa.shortcut_ad_id,
    sa.shortcut_ad_url,
    sa.shortcut_ad_type,
    sa.shortcut_ad_last_seen_at,
    sa.shortcut_building_id,
    sa.shortcut_ad_street_address AS ad_address,
    sa.shortcut_ad_city AS ad_city,
    sa.shortcut_ad_postal AS ad_postal,
    sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS ad_room_layout,
    sa.shortcut_ad_price AS ad_price,
    COALESCE(sa.shortcut_ad_area_value, 0::float8) AS ad_area,
    sa.shortcut_ad_description_text,
    sa.shortcut_ad_availability_text,
    sa.shortcut_ad_renovations_done_text,
    sa.shortcut_ad_renovations_planned_text,
    sa.shortcut_ad_additional_info_text,
    sa.shortcut_ad_charges_text,
    sa.shortcut_ad_maintenance_charge_monthly,
    sa.shortcut_ad_total_charge_monthly,
    sa.shortcut_ad_water_charge,
    sa.shortcut_ad_debt_free_price,
    sa.shortcut_ad_debt_share_amount,
    sa.shortcut_ad_price_per_m2,
    sa.shortcut_ad_floor_level,
    sa.shortcut_ad_total_floors,
    sa.shortcut_ad_build_year,
    sa.shortcut_ad_condition,
    sa.shortcut_ad_energy_class,
    sa.shortcut_ad_plot_type,
    sa.shortcut_ad_elevator,
    sa.shortcut_ad_sauna,
    sa.shortcut_ad_rooms_count,
    sa.shortcut_ad_data,
    sb.shortcut_building_external_id,
    sb.shortcut_building_url,
    sb.shortcut_building_address,
    sb.shortcut_building_housing_company,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id) AS building_listing_count,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id) AS building_rental_count
FROM public.shortcut_ads sa
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
WHERE sa.shortcut_ad_id = sqlc.arg(ad_id)
LIMIT 1;

-- name: GetShortcutBuildingUnifiedDetail :one
SELECT
    sb.shortcut_building_id,
    sb.shortcut_building_external_id,
    sb.shortcut_building_url,
    sb.shortcut_building_address,
    sb.shortcut_building_housing_company,
    sb.shortcut_building_building_type,
    sb.shortcut_building_building_subtype,
    sb.shortcut_building_construction_year,
    sb.shortcut_building_floor_count,
    sb.shortcut_building_apartment_count,
    sb.shortcut_building_heating_system,
    sb.shortcut_building_building_material,
    sb.shortcut_building_plot_type,
    sb.shortcut_building_wall_structure,
    sb.shortcut_building_heat_source,
    sb.shortcut_building_has_elevator,
    sb.shortcut_building_has_sauna,
    sb.shortcut_building_latitude,
    sb.shortcut_building_longitude,
    sb.shortcut_building_updated_at,
    sb.shortcut_building_processed_at,
    sb.shortcut_building_page_not_found,
    (SELECT COUNT(*)::bigint FROM public.shortcut_ads sa WHERE sa.shortcut_building_id = sb.shortcut_building_id) AS ad_count,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id) AS listing_count,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id) AS rental_count,
    jsonb_build_object(
        'building_id', sb.shortcut_building_id,
        'external_id', sb.shortcut_building_external_id,
        'address', sb.shortcut_building_address,
        'housing_company', sb.shortcut_building_housing_company,
        'building_type', sb.shortcut_building_building_type,
        'building_subtype', sb.shortcut_building_building_subtype,
        'construction_year', sb.shortcut_building_construction_year,
        'floor_count', sb.shortcut_building_floor_count,
        'apartment_count', sb.shortcut_building_apartment_count,
        'heating_system', sb.shortcut_building_heating_system,
        'building_material', sb.shortcut_building_building_material,
        'plot_type', sb.shortcut_building_plot_type,
        'wall_structure', sb.shortcut_building_wall_structure,
        'heat_source', sb.shortcut_building_heat_source,
        'has_elevator', sb.shortcut_building_has_elevator,
        'has_sauna', sb.shortcut_building_has_sauna,
        'latitude', sb.shortcut_building_latitude,
        'longitude', sb.shortcut_building_longitude,
        'updated_at', sb.shortcut_building_updated_at,
        'processed_at', sb.shortcut_building_processed_at,
        'page_not_found', sb.shortcut_building_page_not_found
    ) AS raw_json
FROM public.shortcut_buildings sb
WHERE sb.shortcut_building_id = sqlc.arg(building_id)
LIMIT 1;

-- name: GetFrontdoorAdUnifiedDetail :one
SELECT
    fa.frontdoor_ad_id,
    fa.frontdoor_ad_external_id,
    fa.frontdoor_ad_url,
    fa.frontdoor_ad_last_seen_at,
    fa.frontdoor_ad_page_not_found,
    fa.frontdoor_ad_street_address AS ad_address,
    fa.frontdoor_ad_city AS ad_city,
    fa.frontdoor_ad_postal AS ad_postal,
    fa.frontdoor_ad_price AS ad_price,
    COALESCE(fa.frontdoor_ad_area_value, 0::float8) AS ad_area,
    fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS ad_room_layout,
    fa.frontdoor_ad_data #>> '{property,apartmentType}' AS ad_property_type,
    fa.frontdoor_ad_data #>> '{property,condition}' AS ad_condition,
    fa.frontdoor_ad_description_text,
    fa.frontdoor_ad_availability_text,
    fa.frontdoor_ad_renovations_done_text,
    fa.frontdoor_ad_renovations_planned_text,
    fa.frontdoor_ad_additional_info_text,
    fa.frontdoor_ad_charges_text,
    fa.frontdoor_ad_maintenance_charge_monthly,
    fa.frontdoor_ad_total_charge_monthly,
    fa.frontdoor_ad_water_charge,
    fa.frontdoor_ad_debt_free_price,
    fa.frontdoor_ad_debt_share_amount,
    fa.frontdoor_ad_price_per_m2,
    fa.frontdoor_ad_floor_level,
    fa.frontdoor_ad_total_floors,
    fa.frontdoor_ad_build_year,
    fa.frontdoor_ad_energy_class,
    fa.frontdoor_ad_plot_type,
    fa.frontdoor_ad_elevator,
    fa.frontdoor_ad_sauna,
    fa.frontdoor_ad_rooms_count,
    fa.frontdoor_ad_data
FROM public.frontdoor_ads fa
WHERE fa.frontdoor_ad_external_id = sqlc.arg(external_id)
LIMIT 1;

-- name: GetFrontdoorAnnouncementUnifiedDetail :one
SELECT
    fba.frontdoor_building_announcement_id,
    fba.frontdoor_building_announcement_external_id,
    fba.frontdoor_building_announcement_friendly_id,
    fba.frontdoor_building_announcement_last_seen_at,
    fba.frontdoor_building_announcement_address_line1,
    fba.frontdoor_building_announcement_address_line2,
    fba.frontdoor_building_announcement_location,
    fba.frontdoor_building_announcement_search_price,
    fba.frontdoor_building_announcement_area,
    fba.frontdoor_building_announcement_room_structure,
    fba.frontdoor_building_announcement_property_type,
    fba.frontdoor_building_announcement_property_subtype,
    fba.frontdoor_building_announcement_published,
    fb.frontdoor_building_id,
    fb.frontdoor_building_url,
    fb.frontdoor_building_housing_company_id,
    fb.frontdoor_building_housing_company_friendly_id,
    fb.frontdoor_building_company_name,
    fb.frontdoor_building_street_address,
    fb.frontdoor_building_house_number,
    fb.frontdoor_building_postcode,
    fb.frontdoor_building_post_area,
    fb.frontdoor_building_municipality,
    jsonb_build_object(
        'announcement_id', fba.frontdoor_building_announcement_id,
        'external_id', fba.frontdoor_building_announcement_external_id,
        'friendly_id', fba.frontdoor_building_announcement_friendly_id,
        'address_line1', fba.frontdoor_building_announcement_address_line1,
        'address_line2', fba.frontdoor_building_announcement_address_line2,
        'location', fba.frontdoor_building_announcement_location,
        'search_price', fba.frontdoor_building_announcement_search_price,
        'area', fba.frontdoor_building_announcement_area,
        'room_structure', fba.frontdoor_building_announcement_room_structure,
        'property_type', fba.frontdoor_building_announcement_property_type,
        'property_subtype', fba.frontdoor_building_announcement_property_subtype,
        'published', fba.frontdoor_building_announcement_published,
        'building', jsonb_build_object(
            'building_id', fb.frontdoor_building_id,
            'building_url', fb.frontdoor_building_url,
            'housing_company_id', fb.frontdoor_building_housing_company_id,
            'housing_company_friendly_id', fb.frontdoor_building_housing_company_friendly_id,
            'company_name', fb.frontdoor_building_company_name,
            'street_address', fb.frontdoor_building_street_address,
            'house_number', fb.frontdoor_building_house_number,
            'postcode', fb.frontdoor_building_postcode,
            'post_area', fb.frontdoor_building_post_area,
            'municipality', fb.frontdoor_building_municipality
        )
    ) AS raw_json
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE fba.frontdoor_building_announcement_id = sqlc.arg(announcement_id)
LIMIT 1;

-- name: GetFrontdoorBuildingUnifiedDetail :one
SELECT
    fb.frontdoor_building_id,
    fb.frontdoor_building_url,
    fb.frontdoor_building_last_seen_at,
    fb.frontdoor_building_company_name,
    fb.frontdoor_building_business_id,
    fb.frontdoor_building_apartment_count,
    fb.frontdoor_building_floor_count,
    fb.frontdoor_building_build_year,
    fb.frontdoor_building_has_elevator,
    fb.frontdoor_building_has_sauna,
    fb.frontdoor_building_energy_certificate_code,
    fb.frontdoor_building_heating,
    fb.frontdoor_building_street_address,
    fb.frontdoor_building_house_number,
    fb.frontdoor_building_postcode,
    fb.frontdoor_building_post_area,
    fb.frontdoor_building_municipality,
    fb.frontdoor_building_latitude,
    fb.frontdoor_building_longitude,
    fb.frontdoor_building_housing_company_id,
    fb.frontdoor_building_housing_company_friendly_id,
    (SELECT COUNT(*)::bigint FROM public.frontdoor_building_announcements fba WHERE fba.frontdoor_building_id = fb.frontdoor_building_id) AS announcement_count,
    fb.frontdoor_building_data
FROM public.frontdoor_buildings fb
WHERE fb.frontdoor_building_id = sqlc.arg(building_id)
LIMIT 1;
