-- name: SearchAdsReports :many
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        sa.shortcut_ad_id::text AS entity_id,
        COALESCE(
            sa.shortcut_ad_data #>> '{address,formattedAddress}',
            sb.shortcut_building_address,
            sa.shortcut_ad_id::text
        ) AS headline,
        COALESCE(sa.shortcut_ad_data #>> '{address,formattedAddress}', sb.shortcut_building_address) AS address,
        COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}') AS city,
        COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,priceSell}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,price}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(sa.shortcut_ad_data #>> '{adData,size}', '')::float8, 0::float8) AS area,
        sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout,
        sa.shortcut_ad_url AS url,
        sa.shortcut_ad_last_seen_at AS last_seen_at,
        concat_ws(' ',
            sa.shortcut_ad_id::text,
            sa.shortcut_ad_url,
            sa.shortcut_ad_data #>> '{address,formattedAddress}',
            sa.shortcut_ad_data #>> '{address,city,name}',
            sa.shortcut_ad_data #>> '{address,zipCode,value}',
            sa.shortcut_ad_data #>> '{address,zipCode,name}',
            sa.shortcut_ad_data #>> '{adData,roomConfiguration}',
            sb.shortcut_building_address,
            sb.shortcut_building_housing_company
        ) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        fa.frontdoor_ad_external_id AS entity_id,
        COALESCE(
            fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}',
            fa.frontdoor_ad_data #>> '{property,address}',
            fa.frontdoor_ad_external_id
        ) AS headline,
        COALESCE(fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}') AS address,
        COALESCE(fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}') AS city,
        COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{debfFreePrice}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{preparsed,price}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(fa.frontdoor_ad_data #>> '{preparsed,area}', '')::float8, 0::float8) AS area,
        fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS room_layout,
        fa.frontdoor_ad_url AS url,
        fa.frontdoor_ad_last_seen_at AS last_seen_at,
        concat_ws(' ',
            fa.frontdoor_ad_external_id,
            fa.frontdoor_ad_url,
            fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}',
            fa.frontdoor_ad_data #>> '{property,address}',
            fa.frontdoor_ad_data #>> '{property,municipality}',
            fa.frontdoor_ad_data #>> '{property,city}',
            fa.frontdoor_ad_data #>> '{property,postalCode}',
            fa.frontdoor_ad_data #>> '{property,addressPostalCode}',
            fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}'
        ) AS searchable
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        fba.frontdoor_building_announcement_id::text AS entity_id,
        COALESCE(
            fba.frontdoor_building_announcement_address_line1,
            fba.frontdoor_building_announcement_friendly_id,
            fba.frontdoor_building_announcement_external_id::text,
            fba.frontdoor_building_announcement_id::text
        ) AS headline,
        concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2) AS address,
        COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        CASE
            WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL
            ELSE fba.frontdoor_building_announcement_search_price::bigint
        END AS price,
        COALESCE(fba.frontdoor_building_announcement_area, 0::float8) AS area,
        fba.frontdoor_building_announcement_room_structure AS room_layout,
        fb.frontdoor_building_url AS url,
        fba.frontdoor_building_announcement_last_seen_at AS last_seen_at,
        concat_ws(' ',
            fba.frontdoor_building_announcement_id::text,
            fba.frontdoor_building_announcement_external_id::text,
            fba.frontdoor_building_announcement_friendly_id,
            fba.frontdoor_building_announcement_address_line1,
            fba.frontdoor_building_announcement_address_line2,
            fba.frontdoor_building_announcement_location,
            fb.frontdoor_building_postcode,
            fb.frontdoor_building_municipality,
            fb.frontdoor_building_post_area,
            fb.frontdoor_building_url,
            fba.frontdoor_building_announcement_room_structure
        ) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
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
    entity_id,
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
    entity_id
LIMIT sqlc.arg(limit_count)::int
OFFSET sqlc.arg(offset_count)::int;

-- name: CountAdsReports :one
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}') AS city,
        COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,priceSell}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,price}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(sa.shortcut_ad_data #>> '{adData,size}', '')::float8, 0::float8) AS area,
        concat_ws(' ',
            sa.shortcut_ad_id::text,
            sa.shortcut_ad_url,
            sa.shortcut_ad_data #>> '{address,formattedAddress}',
            sa.shortcut_ad_data #>> '{address,city,name}',
            sa.shortcut_ad_data #>> '{address,zipCode,value}',
            sa.shortcut_ad_data #>> '{address,zipCode,name}',
            sa.shortcut_ad_data #>> '{adData,roomConfiguration}',
            sb.shortcut_building_address,
            sb.shortcut_building_housing_company
        ) AS searchable
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        COALESCE(fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}') AS city,
        COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}') AS postal,
        COALESCE(
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{debfFreePrice}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint,
            NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{preparsed,price}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint
        ) AS price,
        COALESCE(NULLIF(fa.frontdoor_ad_data #>> '{preparsed,area}', '')::float8, 0::float8) AS area,
        concat_ws(' ',
            fa.frontdoor_ad_external_id,
            fa.frontdoor_ad_url,
            fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}',
            fa.frontdoor_ad_data #>> '{property,address}',
            fa.frontdoor_ad_data #>> '{property,municipality}',
            fa.frontdoor_ad_data #>> '{property,city}',
            fa.frontdoor_ad_data #>> '{property,postalCode}',
            fa.frontdoor_ad_data #>> '{property,addressPostalCode}',
            fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}'
        ) AS searchable
    FROM public.frontdoor_ads fa
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        CASE
            WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL
            ELSE fba.frontdoor_building_announcement_search_price::bigint
        END AS price,
        COALESCE(fba.frontdoor_building_announcement_area, 0::float8) AS area,
        concat_ws(' ',
            fba.frontdoor_building_announcement_id::text,
            fba.frontdoor_building_announcement_external_id::text,
            fba.frontdoor_building_announcement_friendly_id,
            fba.frontdoor_building_announcement_address_line1,
            fba.frontdoor_building_announcement_address_line2,
            fba.frontdoor_building_announcement_location,
            fb.frontdoor_building_postcode,
            fb.frontdoor_building_municipality,
            fb.frontdoor_building_post_area,
            fb.frontdoor_building_url,
            fba.frontdoor_building_announcement_room_structure
        ) AS searchable
    FROM public.frontdoor_building_announcements fba
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
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

-- name: GetShortcutAdReportDetail :one
SELECT
    sa.shortcut_ad_id,
    sa.shortcut_ad_url,
    sa.shortcut_ad_type,
    sa.shortcut_ad_last_seen_at,
    sa.shortcut_building_id,
    sa.shortcut_ad_data #>> '{address,formattedAddress}' AS ad_address,
    sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS ad_room_layout,
    COALESCE(
        NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,priceSell}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint,
        NULLIF(regexp_replace(sa.shortcut_ad_data #>> '{priceData,price}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint
    ) AS ad_price,
    COALESCE(NULLIF(sa.shortcut_ad_data #>> '{adData,size}', '')::float8, 0::float8) AS ad_area,
    sb.shortcut_building_external_id,
    sb.shortcut_building_url,
    sb.shortcut_building_address,
    sb.shortcut_building_housing_company,
    COALESCE((SELECT COUNT(*)::bigint FROM public.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id), 0)::bigint AS building_listing_count,
    COALESCE((SELECT COUNT(*)::bigint FROM public.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id), 0)::bigint AS building_rental_count
FROM public.shortcut_ads sa
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
WHERE sa.shortcut_ad_id = sqlc.arg(ad_id)
LIMIT 1;

-- name: GetFrontdoorAdReportDetail :one
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
        NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{debfFreePrice}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint,
        NULLIF(regexp_replace(fa.frontdoor_ad_data #>> '{preparsed,price}', '[^0-9\\.-]', '', 'g'), '')::numeric::bigint
    ) AS ad_price,
    COALESCE(NULLIF(fa.frontdoor_ad_data #>> '{preparsed,area}', '')::float8, 0::float8) AS ad_area,
    fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}' AS ad_room_layout,
    fa.frontdoor_ad_data #>> '{property,apartmentType}' AS ad_property_type,
    fa.frontdoor_ad_data #>> '{property,condition}' AS ad_condition
FROM public.frontdoor_ads fa
WHERE fa.frontdoor_ad_external_id = sqlc.arg(external_id)
LIMIT 1;

-- name: GetFrontdoorAnnouncementReportDetail :one
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
    fb.frontdoor_building_municipality
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE fba.frontdoor_building_announcement_id = sqlc.arg(announcement_id)
LIMIT 1;
