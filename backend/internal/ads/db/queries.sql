-- name: SearchUnifiedEntities :many
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        sa.shortcut_ad_id::text AS native_id,
        ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id,
        COALESCE(sa.shortcut_ad_data #>> '{address,formattedAddress}', sb.shortcut_building_address, sa.shortcut_ad_id::text) AS headline,
        COALESCE(sa.shortcut_ad_data #>> '{address,formattedAddress}', sb.shortcut_building_address) AS address,
        COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}') AS city,
        COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,priceSell}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,price}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(sa.shortcut_ad_data #>> '{adData,size}', '')::float8, 0::float8) AS area,
        sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout,
        sa.shortcut_ad_url AS url,
        sa.shortcut_ad_last_seen_at AS last_seen_at,
        concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, sa.shortcut_ad_data #>> '{address,formattedAddress}', sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable
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
        concat_ws(' ', sb.shortcut_building_id::text, sb.shortcut_building_external_id::text, sb.shortcut_building_url, sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_building_type, sb.shortcut_building_building_subtype) AS searchable
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        fa.frontdoor_ad_external_id AS native_id,
        ('frontdoor:ad:' || fa.frontdoor_ad_external_id) AS canonical_id,
        COALESCE(fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}', fa.frontdoor_ad_external_id) AS headline,
        COALESCE(fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}') AS address,
        COALESCE(fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}') AS city,
        COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{debfFreePrice}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{preparsed,price}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(fa.frontdoor_ad_data #>> '{preparsed,area}', '')::float8, 0::float8) AS area,
        fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS room_layout,
        fa.frontdoor_ad_url AS url,
        fa.frontdoor_ad_last_seen_at AS last_seen_at,
        concat_ws(' ', fa.frontdoor_ad_external_id, fa.frontdoor_ad_url, fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}', fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}', fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}') AS searchable
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
        concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
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
        concat_ws(' ', fb.frontdoor_building_id::text, fb.frontdoor_building_url, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_company_name, fb.frontdoor_building_street_address, fb.frontdoor_building_house_number, fb.frontdoor_building_postcode, fb.frontdoor_building_post_area, fb.frontdoor_building_municipality) AS searchable
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
        COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}') AS city,
        COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,priceSell}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,price}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(sa.shortcut_ad_data #>> '{adData,size}', '')::float8, 0::float8) AS area,
        concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, sa.shortcut_ad_data #>> '{address,formattedAddress}', sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company) AS searchable
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
        concat_ws(' ', sb.shortcut_building_id::text, sb.shortcut_building_external_id::text, sb.shortcut_building_url, sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_building_type, sb.shortcut_building_building_subtype) AS searchable
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        COALESCE(fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}') AS city,
        COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{debfFreePrice}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{preparsed,price}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(fa.frontdoor_ad_data #>> '{preparsed,area}', '')::float8, 0::float8) AS area,
        concat_ws(' ', fa.frontdoor_ad_external_id, fa.frontdoor_ad_url, fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}', fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}', fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}') AS searchable
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END AS price,
        COALESCE(fba.frontdoor_building_announcement_area, 0::float8) AS area,
        concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure) AS searchable
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
        concat_ws(' ', fb.frontdoor_building_id::text, fb.frontdoor_building_url, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_company_name, fb.frontdoor_building_street_address, fb.frontdoor_building_house_number, fb.frontdoor_building_postcode, fb.frontdoor_building_post_area, fb.frontdoor_building_municipality) AS searchable
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
  AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8);

-- name: GetShortcutAdUnifiedDetail :one
SELECT
    sa.shortcut_ad_id,
    sa.shortcut_ad_url,
    sa.shortcut_ad_type,
    sa.shortcut_ad_last_seen_at,
    sa.shortcut_building_id,
    sa.shortcut_ad_data #>> '{address,formattedAddress}' AS ad_address,
    sa.shortcut_ad_data #>> '{address,city,name}' AS ad_city,
    COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}') AS ad_postal,
    sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS ad_room_layout,
    COALESCE(
        NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,priceSell}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint,
        NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,price}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint
    ) AS ad_price,
    COALESCE(NULLIF(sa.shortcut_ad_data #>> '{adData,size}', '')::float8, 0::float8) AS ad_area,
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
    fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}' AS ad_address,
    COALESCE(fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}') AS ad_city,
    COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}') AS ad_postal,
    COALESCE(
        NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{debfFreePrice}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint,
        NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{preparsed,price}', '[^0-9\.-]', '', 'g'), '')::numeric::bigint
    ) AS ad_price,
    COALESCE(NULLIF(fa.frontdoor_ad_data #>> '{preparsed,area}', '')::float8, 0::float8) AS ad_area,
    fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS ad_room_layout,
    fa.frontdoor_ad_data #>> '{property,apartmentType}' AS ad_property_type,
    fa.frontdoor_ad_data #>> '{property,condition}' AS ad_condition,
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
