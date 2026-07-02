-- name: CountGroupedOfferings :one
WITH source_rows AS (
    SELECT
        source_link.target_id AS property_offering_id,
        sl.sale_listing_id
    FROM public.target_sources source_link
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.property_houses ph ON ph.property_house_id = po.property_house_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    WHERE source_link.target_type = 'listing'
      AND source_link.source_type = 'source_listing'
      AND source_link.link_status <> 'rejected'
      AND (sqlc.arg('source')::text = 'all' OR sl.sale_listing_source_provider = sqlc.arg('source')::text)
      AND (sqlc.arg('kind')::text = 'all' OR sl.sale_listing_source_kind = sqlc.arg('kind')::text)
      AND (sqlc.narg('query_text')::text IS NULL OR lower(concat_ws(' ', po.property_offering_headline, sl.sale_listing_search_text, sl.sale_listing_canonical_id, sl.sale_listing_native_id, hc.housing_company_name, pu.property_unit_address_norm, ph.property_house_address_norm)) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
      AND (sqlc.narg('city')::text IS NULL OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, hc.housing_company_city_norm, ph.property_house_city_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
      AND (sqlc.narg('postal')::text IS NULL OR lower(COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, hc.housing_company_postal_norm, ph.property_house_postal_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
      AND (sqlc.narg('min_price')::bigint IS NULL OR COALESCE(sl.sale_listing_asking_price, po.property_offering_asking_price) >= sqlc.narg('min_price')::bigint)
      AND (sqlc.narg('max_price')::bigint IS NULL OR COALESCE(sl.sale_listing_asking_price, po.property_offering_asking_price) <= sqlc.narg('max_price')::bigint)
      AND (sqlc.narg('min_area')::float8 IS NULL OR COALESCE(sl.sale_listing_area_value, pu.property_unit_area_value, ph.property_house_area_value) >= sqlc.narg('min_area')::float8)
      AND (sqlc.narg('max_area')::float8 IS NULL OR COALESCE(sl.sale_listing_area_value, pu.property_unit_area_value, ph.property_house_area_value) <= sqlc.narg('max_area')::float8)
      AND (sqlc.narg('published_after')::timestamptz IS NULL OR sl.sale_listing_published_at >= sqlc.narg('published_after')::timestamptz)
      AND (sqlc.narg('published_before')::timestamptz IS NULL OR sl.sale_listing_published_at <= sqlc.narg('published_before')::timestamptz)
)
SELECT count(DISTINCT property_offering_id)::bigint
FROM source_rows;

-- name: SearchGroupedOfferings :many
WITH source_rows AS (
    SELECT
        source_link.target_id AS property_offering_id,
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        sl.sale_listing_last_seen_at
    FROM public.target_sources source_link
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.property_houses ph ON ph.property_house_id = po.property_house_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    WHERE source_link.target_type = 'listing'
      AND source_link.source_type = 'source_listing'
      AND source_link.link_status <> 'rejected'
      AND (sqlc.arg('source')::text = 'all' OR sl.sale_listing_source_provider = sqlc.arg('source')::text)
      AND (sqlc.arg('kind')::text = 'all' OR sl.sale_listing_source_kind = sqlc.arg('kind')::text)
      AND (sqlc.narg('query_text')::text IS NULL OR lower(concat_ws(' ', po.property_offering_headline, sl.sale_listing_search_text, sl.sale_listing_canonical_id, sl.sale_listing_native_id, hc.housing_company_name, pu.property_unit_address_norm, ph.property_house_address_norm)) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
      AND (sqlc.narg('city')::text IS NULL OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, hc.housing_company_city_norm, ph.property_house_city_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('city')::text)) || '%'))
      AND (sqlc.narg('postal')::text IS NULL OR lower(COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, hc.housing_company_postal_norm, ph.property_house_postal_norm, '')) LIKE ('%' || lower(trim(sqlc.narg('postal')::text)) || '%'))
      AND (sqlc.narg('min_price')::bigint IS NULL OR COALESCE(sl.sale_listing_asking_price, po.property_offering_asking_price) >= sqlc.narg('min_price')::bigint)
      AND (sqlc.narg('max_price')::bigint IS NULL OR COALESCE(sl.sale_listing_asking_price, po.property_offering_asking_price) <= sqlc.narg('max_price')::bigint)
      AND (sqlc.narg('min_area')::float8 IS NULL OR COALESCE(sl.sale_listing_area_value, pu.property_unit_area_value, ph.property_house_area_value) >= sqlc.narg('min_area')::float8)
      AND (sqlc.narg('max_area')::float8 IS NULL OR COALESCE(sl.sale_listing_area_value, pu.property_unit_area_value, ph.property_house_area_value) <= sqlc.narg('max_area')::float8)
      AND (sqlc.narg('published_after')::timestamptz IS NULL OR sl.sale_listing_published_at >= sqlc.narg('published_after')::timestamptz)
      AND (sqlc.narg('published_before')::timestamptz IS NULL OR sl.sale_listing_published_at <= sqlc.narg('published_before')::timestamptz)
), offering_matches AS (
    SELECT
        property_offering_id,
        count(DISTINCT sale_listing_id)::int4 AS source_count,
        string_agg(DISTINCT sale_listing_source_provider, ',' ORDER BY sale_listing_source_provider) AS sources,
        max(sale_listing_last_seen_at) AS source_last_seen_at
    FROM source_rows
    GROUP BY property_offering_id
), offering_rows AS (
    SELECT
        po.property_offering_id,
        COALESCE(hc.housing_company_id::text, '')::text AS housing_company_id,
        COALESCE(hc.housing_company_name, '') AS housing_company_name,
        COALESCE(po.property_offering_headline, primary_listing.sale_listing_headline, primary_listing.sale_listing_street_address, pu.property_unit_room_layout, pu.property_unit_address_norm, ph.property_house_address_norm, po.property_offering_id::text) AS headline,
        COALESCE(primary_listing.sale_listing_street_address, pu.property_unit_address_norm, hc.housing_company_address_norm, ph.property_house_address_norm, '') AS address,
        COALESCE(primary_listing.sale_listing_city, primary_listing.sale_listing_city_norm, hc.housing_company_city_norm, ph.property_house_city_norm, '') AS city,
        COALESCE(primary_listing.sale_listing_postal, primary_listing.sale_listing_postal_norm, hc.housing_company_postal_norm, ph.property_house_postal_norm, '') AS postal,
        po.property_offering_asking_price AS price,
        COALESCE(pu.property_unit_area_value, ph.property_house_area_value) AS area,
        COALESCE(pu.property_unit_room_layout, primary_listing.sale_listing_room_layout, '') AS room_layout,
        COALESCE(po.property_offering_last_seen_at, offering_matches.source_last_seen_at) AS last_seen_at,
        offering_matches.source_count,
        COALESCE(offering_matches.sources, '')::text AS sources
    FROM offering_matches
    JOIN public.property_offerings po ON po.property_offering_id = offering_matches.property_offering_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.property_houses ph ON ph.property_house_id = po.property_house_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    LEFT JOIN public.property_source_offerings primary_listing ON primary_listing.sale_listing_id = po.primary_sale_listing_id
)
SELECT
    row.property_offering_id::text,
    row.housing_company_id,
    row.housing_company_name,
    row.headline,
    row.address,
    row.city,
    row.postal,
    row.price,
    row.area,
    row.room_layout,
    row.last_seen_at,
    row.source_count,
    row.sources,
    COALESCE(price_match.transaction_id::text, '')::text AS price_match_transaction_id,
    COALESCE(price_match.match_scope, '') AS price_match_scope,
    COALESCE(price_match.match_status, '') AS price_match_status,
    COALESCE(price_match.match_method, '') AS price_match_method,
    COALESCE(price_match.match_score, 0)::int4 AS price_match_score,
    COALESCE(price_match.price_eur, 0)::bigint AS price_match_price_eur,
    COALESCE(insight_stats.insight_count, 0)::int4 AS insight_count,
    COALESCE(insight_stats.top_severity, '') AS insight_top_severity
FROM offering_rows row
LEFT JOIN LATERAL (
    SELECT
        pt.prices_transaction_id AS transaction_id,
        price_link.target_type AS match_scope,
        price_link.link_status AS match_status,
        price_link.link_method AS match_method,
        price_link.link_score::int4 AS match_score,
        pt.prices_transaction_price AS price_eur
    FROM public.price_links price_link
    JOIN origin.prices_transactions pt ON pt.prices_transaction_id = price_link.prices_transaction_id
    WHERE price_link.link_status <> 'rejected'
        AND (
            (price_link.target_type = 'source_listing' AND EXISTS (
                SELECT 1 FROM source_rows sr WHERE sr.property_offering_id = row.property_offering_id AND sr.sale_listing_id = price_link.target_id
            ))
            OR (price_link.target_type = 'listing' AND price_link.target_id = row.property_offering_id)
        )
    ORDER BY CASE WHEN price_link.target_type = 'source_listing' THEN 0 ELSE 1 END, price_link.link_score DESC NULLS LAST, pt.prices_transaction_updated_at DESC
    LIMIT 1
) price_match ON true
LEFT JOIN LATERAL (
    SELECT
        count(*)::int4 AS insight_count,
        max(observation.severity)::text AS top_severity
    FROM public.target_observations observation
    WHERE observation.target_type = 'listing'
        AND observation.target_id = row.property_offering_id
        AND observation.superseded_at IS NULL
) insight_stats ON true
ORDER BY
    CASE WHEN sqlc.arg('sort_mode')::text = 'price_asc' THEN row.price END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode')::text = 'price_desc' THEN row.price END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode')::text = 'area_asc' THEN row.area END ASC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode')::text = 'area_desc' THEN row.area END DESC NULLS LAST,
    CASE WHEN sqlc.arg('sort_mode')::text = 'seen_desc' THEN row.last_seen_at END DESC NULLS LAST,
    row.last_seen_at DESC NULLS LAST,
    row.price ASC NULLS LAST,
    row.property_offering_id
LIMIT sqlc.arg('limit_count')::int
OFFSET sqlc.arg('offset_count')::int;

-- name: LookupAddressListings :many
WITH lookup_input_raw AS (
    SELECT
        NULLIF(trim(regexp_replace(lower(regexp_replace(trim($1::text), '[^[:alnum:]åäöÅÄÖ]+', ' ', 'g')), '\s+', ' ', 'g')), '') AS address_norm,
        NULLIF(regexp_replace(trim(COALESCE($3::text, '')), '[^0-9]+', '', 'g'), '') AS postal_norm
),
lookup_input AS (
    SELECT
        address_norm,
        translate(address_norm, 'åäö', 'aao') AS address_ascii_norm,
        substring(address_norm from '^(.*)\s+[0-9]+(\s*[[:alpha:]])?\s*$') AS street_name_norm,
        substring(translate(address_norm, 'åäö', 'aao') from '^(.*)\s+[0-9]+(\s*[[:alpha:]])?\s*$') AS street_name_ascii_norm,
        substring(address_norm from '\s([0-9]+)(\s*[[:alpha:]])?\s*$') AS street_number_norm,
        substring(translate(address_norm, 'åäö', 'aao') from '\s[0-9]+\s*([[:alpha:]])\s*$') AS building_letter_ascii_norm,
        postal_norm
    FROM lookup_input_raw
),
selected_listing_matches AS (
    SELECT
        sl.sale_listing_id,
        source_link.target_id AS property_offering_id
    FROM public.property_source_offerings sl
    CROSS JOIN lookup_input li
    LEFT JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    WHERE ($4::text = 'all' OR sl.sale_listing_source_provider = $4::text)
        AND sl.sale_listing_source_kind = ANY(ARRAY['ad'::text, 'announcement'::text])
        AND trim($3::text) <> ''
        AND COALESCE(sl.sale_listing_postal_norm, NULLIF(regexp_replace(trim(COALESCE(sl.sale_listing_postal, '')), '[^0-9]+', '', 'g'), '')) = li.postal_norm
        AND (
            sl.sale_listing_address_norm = li.address_norm
            OR translate(sl.sale_listing_address_norm, 'åäö', 'aao') = li.address_ascii_norm
            OR lower(COALESCE(sl.sale_listing_street_address, '')) LIKE ('%' || lower(trim($1::text)) || '%')
            OR translate(lower(COALESCE(sl.sale_listing_street_address, '')), 'åäö', 'aao') LIKE ('%' || li.address_ascii_norm || '%')
            OR (
                sl.sale_listing_street_name_norm IS NOT NULL
                AND sl.sale_listing_street_number_norm IS NOT NULL
                AND (
                    li.building_letter_ascii_norm IS NULL
                    OR translate(COALESCE(sl.sale_listing_building_letter_norm, ''), 'åäö', 'aao') = li.building_letter_ascii_norm
                )
                AND (
                    (' ' || li.address_norm || ' ')
                        LIKE ('% ' || sl.sale_listing_street_name_norm || ' ' || sl.sale_listing_street_number_norm || ' %')
                    OR (' ' || li.address_ascii_norm || ' ')
                        LIKE ('% ' || translate(sl.sale_listing_street_name_norm, 'åäö', 'aao') || ' ' || sl.sale_listing_street_number_norm || ' %')
                )
            )
        )
        AND (trim($2::text) = '' OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')) LIKE ('%' || lower(trim($2::text)) || '%'))
    UNION ALL
    SELECT
        sl.sale_listing_id,
        source_link.target_id AS property_offering_id
    FROM public.property_source_offerings sl
    CROSS JOIN lookup_input li
    LEFT JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    WHERE ($4::text = 'all' OR sl.sale_listing_source_provider = $4::text)
        AND sl.sale_listing_source_kind = ANY(ARRAY['ad'::text, 'announcement'::text])
        AND trim($3::text) = ''
        AND li.street_name_ascii_norm IS NOT NULL
        AND li.street_number_norm IS NOT NULL
        AND translate(sl.sale_listing_street_name_norm, 'åäö', 'aao') = li.street_name_ascii_norm
        AND sl.sale_listing_street_number_norm = li.street_number_norm
        AND (
            li.building_letter_ascii_norm IS NULL
            OR translate(COALESCE(sl.sale_listing_building_letter_norm, ''), 'åäö', 'aao') = li.building_letter_ascii_norm
        )
        AND (trim($2::text) = '' OR lower(COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '')) LIKE ('%' || lower(trim($2::text)) || '%'))
),
selected_listing_ids AS (
    SELECT DISTINCT ON (sale_listing_id)
        sale_listing_id,
        property_offering_id
    FROM selected_listing_matches
    ORDER BY sale_listing_id, property_offering_id NULLS LAST
),
selected_listings AS (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_canonical_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        sl.sale_listing_native_id,
        COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
        COALESCE(sl.sale_listing_street_address, '') AS address,
        COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS city,
        COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS postal,
        sl.sale_listing_latitude,
        sl.sale_listing_longitude,
        sl.sale_listing_asking_price,
        sl.sale_listing_debt_free_price,
        sl.sale_listing_area_value,
        COALESCE(sl.sale_listing_room_layout, '') AS room_layout,
        COALESCE(sl.sale_listing_url, '') AS url,
        CASE
            WHEN sl.sale_listing_source_provider = 'shortcut' AND sl.sale_listing_source_kind = 'ad' THEN sl.shortcut_ad_id IS NOT NULL AND COALESCE(sl.sale_listing_url, '') <> '' AND sl.sale_listing_last_seen_at >= now() - interval '7 days'
            WHEN sl.sale_listing_source_provider = 'frontdoor' AND sl.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
            WHEN sl.sale_listing_source_provider = 'frontdoor' AND sl.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
            ELSE false
        END AS external_url_available,
        sl.sale_listing_first_seen_at,
        sl.sale_listing_last_seen_at,
        sl.sale_listing_published_at,
        sl.sale_listing_created_at,
        sl.sale_listing_updated_at,
        sl.sale_listing_previous_asking_price,
        sl.sale_listing_previous_debt_free_price,
        COALESCE(direct_price.link_status, '') AS prices_match_status,
        COALESCE(sl.sale_listing_source_match_status, '') AS source_match_status,
        sli.property_offering_id,
        pu.housing_company_id,
        COALESCE(hc.housing_company_name, '') AS housing_company_name,
        COALESCE(sl.sale_listing_availability_text, '') AS availability_text,
        COALESCE(sl.sale_listing_renovations_done_text, '') AS renovations_done_text,
        COALESCE(sl.sale_listing_renovations_planned_text, '') AS renovations_planned_text,
        COALESCE(sl.sale_listing_additional_info_text, '') AS additional_info_text,
        COALESCE(sl.sale_listing_charges_text, '') AS charges_text,
        COALESCE(insight_rows.insights_json, '[]'::jsonb) AS insights_json,
        ROW_NUMBER() OVER (
            ORDER BY
                CASE
                    WHEN sl.sale_listing_address_norm = li.address_norm THEN 0
                    WHEN translate(sl.sale_listing_address_norm, 'åäö', 'aao') = li.address_ascii_norm THEN 1
                    WHEN lower(COALESCE(sl.sale_listing_street_address, '')) = lower(trim($1::text)) THEN 1
                    ELSE 3
                END,
                sl.sale_listing_last_seen_at DESC NULLS LAST,
                sl.sale_listing_source_provider,
                sl.sale_listing_source_kind,
                sl.sale_listing_native_id
        ) AS listing_rank,
        count(*) OVER ()::integer AS listing_count
    FROM selected_listing_ids sli
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = sli.sale_listing_id
    LEFT JOIN public.property_offerings po ON po.property_offering_id = sli.property_offering_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
    LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    LEFT JOIN LATERAL (
        SELECT price_link.link_status
        FROM public.price_links price_link
        WHERE price_link.target_type = 'source_listing'
            AND price_link.target_id = sl.sale_listing_id
            AND price_link.link_status <> 'rejected'
        ORDER BY price_link.link_score DESC NULLS LAST, price_link.updated_at DESC
        LIMIT 1
    ) direct_price ON true
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
            AND observation.source_id = sl.sale_listing_id
            AND observation.superseded_at IS NULL
    ) insight_rows ON true
    CROSS JOIN lookup_input li
),
limited_listings AS (
    SELECT
        sale_listing_id,
        sale_listing_canonical_id,
        sale_listing_source_provider,
        sale_listing_source_kind,
        sale_listing_native_id,
        headline,
        address,
        city,
        postal,
        sale_listing_latitude,
        sale_listing_longitude,
        sale_listing_asking_price,
        sale_listing_debt_free_price,
        sale_listing_area_value,
        room_layout,
        url,
        external_url_available,
        sale_listing_first_seen_at,
        sale_listing_last_seen_at,
        sale_listing_published_at,
        sale_listing_created_at,
        sale_listing_updated_at,
        sale_listing_previous_asking_price,
        sale_listing_previous_debt_free_price,
        prices_match_status,
        source_match_status,
        property_offering_id,
        housing_company_id,
        housing_company_name,
        availability_text,
        renovations_done_text,
        renovations_planned_text,
        additional_info_text,
        charges_text,
        insights_json,
        listing_rank,
        listing_count
    FROM selected_listings
    WHERE listing_rank <= $5::int
),
matched_offerings AS (
    SELECT DISTINCT property_offering_id
    FROM limited_listings
    WHERE property_offering_id IS NOT NULL
),
offering_source_records AS (
    SELECT
        source_link.target_id AS property_offering_id,
        sr.sale_listing_id,
        sr.sale_listing_canonical_id,
        sr.sale_listing_source_provider,
        sr.sale_listing_source_kind,
        sr.sale_listing_native_id,
        COALESCE(sr.sale_listing_headline, sr.sale_listing_street_address, sr.sale_listing_native_id) AS headline,
        COALESCE(sr.sale_listing_street_address, '') AS address,
        COALESCE(sr.sale_listing_city, sr.sale_listing_city_norm, '') AS city,
        COALESCE(sr.sale_listing_postal, sr.sale_listing_postal_norm, '') AS postal,
        sr.sale_listing_latitude,
        sr.sale_listing_longitude,
        sr.sale_listing_asking_price,
        sr.sale_listing_debt_free_price,
        sr.sale_listing_area_value,
        COALESCE(sr.sale_listing_room_layout, '') AS room_layout,
        COALESCE(sr.sale_listing_url, '') AS url,
        CASE
            WHEN sr.sale_listing_source_provider = 'shortcut' AND sr.sale_listing_source_kind = 'ad' THEN sr.shortcut_ad_id IS NOT NULL AND COALESCE(sr.sale_listing_url, '') <> '' AND sr.sale_listing_last_seen_at >= now() - interval '7 days'
            WHEN sr.sale_listing_source_provider = 'frontdoor' AND sr.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
            WHEN sr.sale_listing_source_provider = 'frontdoor' AND sr.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
            ELSE false
        END AS external_url_available,
        sr.sale_listing_first_seen_at,
        sr.sale_listing_last_seen_at,
        sr.sale_listing_updated_at,
        sr.sale_listing_previous_asking_price,
        sr.sale_listing_previous_debt_free_price,
        direct_price.prices_transaction_id,
        COALESCE(direct_price.link_status, '') AS prices_match_status,
        COALESCE(source_link.link_status, '') AS source_link_status,
        COALESCE(source_link.link_method, '') AS source_link_method,
        source_link.link_score AS property_offering_source_link_score,
        COALESCE(sr.sale_listing_availability_text, '') AS availability_text,
        COALESCE(sr.sale_listing_renovations_done_text, '') AS renovations_done_text,
        COALESCE(sr.sale_listing_renovations_planned_text, '') AS renovations_planned_text,
        COALESCE(sr.sale_listing_additional_info_text, '') AS additional_info_text,
        COALESCE(sr.sale_listing_charges_text, '') AS charges_text,
        COALESCE(insight_rows.insights_json, '[]'::jsonb) AS insights_json
    FROM matched_offerings mo
    JOIN public.target_sources source_link ON source_link.target_id = mo.property_offering_id
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_source_offerings sr ON sr.sale_listing_id = source_link.source_id
    LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = sr.frontdoor_ad_id
    LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sr.frontdoor_building_announcement_id
    LEFT JOIN LATERAL (
        SELECT price_link.prices_transaction_id, price_link.link_status
        FROM public.price_links price_link
        WHERE price_link.target_type = 'source_listing'
            AND price_link.target_id = sr.sale_listing_id
            AND price_link.link_status <> 'rejected'
        ORDER BY price_link.link_score DESC NULLS LAST, price_link.updated_at DESC
        LIMIT 1
    ) direct_price ON true
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
            AND observation.source_id = sr.sale_listing_id
            AND observation.superseded_at IS NULL
    ) insight_rows ON true
),
latest_candidates AS (
    SELECT DISTINCT ON (c.sale_listing_id, c.prices_transaction_id)
        c.sale_listing_id,
        c.prices_transaction_id,
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_confidence,
        c.sale_listing_prices_transaction_match_status,
        c.sale_listing_prices_transaction_match_reasons,
        c.sale_listing_prices_transaction_match_price_delta_percent
    FROM public.sale_listing_prices_transaction_match_candidates c
    JOIN limited_listings sl ON sl.sale_listing_id = c.sale_listing_id
    ORDER BY c.sale_listing_id, c.prices_transaction_id, c.sale_listing_prices_transaction_match_created_at DESC
),
links AS (
    SELECT
        selected.sale_listing_id,
        price_link.prices_transaction_id,
        'direct'::text AS link_type,
        price_link.link_status,
        price_link.link_method,
        COALESCE(price_link.link_score, lc.sale_listing_prices_transaction_match_score) AS score,
        lc.sale_listing_prices_transaction_match_confidence AS confidence,
        lc.sale_listing_prices_transaction_match_price_delta_percent AS price_delta_percent,
        COALESCE(lc.sale_listing_prices_transaction_match_reasons, price_link.link_reasons) AS reasons,
        1 AS link_rank
    FROM limited_listings selected
    JOIN public.price_links price_link ON price_link.target_type = 'source_listing'
        AND price_link.target_id = selected.sale_listing_id
        AND price_link.link_status <> 'rejected'
    LEFT JOIN latest_candidates lc ON lc.sale_listing_id = selected.sale_listing_id
        AND lc.prices_transaction_id = price_link.prices_transaction_id
    UNION ALL
    SELECT
        selected.sale_listing_id,
        price_link.prices_transaction_id,
        'offering'::text,
        price_link.link_status,
        price_link.link_method,
        price_link.link_score,
        lc.sale_listing_prices_transaction_match_confidence,
        lc.sale_listing_prices_transaction_match_price_delta_percent,
        COALESCE(lc.sale_listing_prices_transaction_match_reasons, price_link.link_reasons),
        2
    FROM limited_listings selected
    JOIN public.price_links price_link ON price_link.target_type = 'listing'
        AND price_link.target_id = selected.property_offering_id
        AND price_link.link_status <> 'rejected'
    LEFT JOIN latest_candidates lc ON lc.sale_listing_id = selected.sale_listing_id
        AND lc.prices_transaction_id = price_link.prices_transaction_id
    UNION ALL
    SELECT
        selected.sale_listing_id,
        price_link.prices_transaction_id,
        'source_record'::text,
        price_link.link_status,
        price_link.link_method,
        price_link.link_score,
        lc.sale_listing_prices_transaction_match_confidence,
        NULL::double precision,
        jsonb_build_object(
            'source_listing_id', osr.sale_listing_id,
            'source_listing_provider', osr.sale_listing_source_provider,
            'source_listing_native_id', osr.sale_listing_native_id
        ),
        3
    FROM limited_listings selected
    JOIN offering_source_records osr ON osr.property_offering_id = selected.property_offering_id
    JOIN public.price_links price_link ON price_link.target_type = 'source_listing'
        AND price_link.target_id = osr.sale_listing_id
        AND price_link.link_status <> 'rejected'
    LEFT JOIN latest_candidates lc ON lc.sale_listing_id = osr.sale_listing_id
        AND lc.prices_transaction_id = price_link.prices_transaction_id
    UNION ALL
    SELECT
        lc.sale_listing_id,
        lc.prices_transaction_id,
        'candidate'::text,
        lc.sale_listing_prices_transaction_match_status,
        'match_candidate'::text,
        lc.sale_listing_prices_transaction_match_score,
        lc.sale_listing_prices_transaction_match_confidence,
        lc.sale_listing_prices_transaction_match_price_delta_percent,
        lc.sale_listing_prices_transaction_match_reasons,
        4
    FROM latest_candidates lc
    WHERE lc.sale_listing_prices_transaction_match_status = ANY(ARRAY['candidate'::text, 'ambiguous'::text])
),
dedup_links AS (
    SELECT DISTINCT ON (sale_listing_id, prices_transaction_id)
        sale_listing_id,
        prices_transaction_id,
        link_type,
        link_status,
        link_method,
        score,
        confidence,
        price_delta_percent,
        reasons,
        link_rank
    FROM links
    ORDER BY sale_listing_id, prices_transaction_id, link_rank, score DESC NULLS LAST
)
SELECT
    sl.sale_listing_id,
    sl.sale_listing_canonical_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.headline,
    sl.address,
    sl.city,
    sl.postal,
    sl.sale_listing_latitude,
    sl.sale_listing_longitude,
    sl.sale_listing_asking_price,
    sl.sale_listing_debt_free_price,
    sl.sale_listing_area_value,
    sl.room_layout,
    sl.url,
    sl.external_url_available,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_published_at,
    sl.sale_listing_created_at,
    sl.sale_listing_updated_at,
    sl.sale_listing_previous_asking_price,
    sl.sale_listing_previous_debt_free_price,
    sl.prices_match_status,
    sl.source_match_status,
    sl.property_offering_id,
    sl.housing_company_id,
    sl.housing_company_name,
    sl.availability_text,
    sl.renovations_done_text,
    sl.renovations_planned_text,
    sl.additional_info_text,
    sl.charges_text,
    sl.insights_json,
    sl.listing_count,
    pt.prices_transaction_id,
    COALESCE(dl.link_type, ''),
    COALESCE(dl.link_status, ''),
    COALESCE(dl.link_method, ''),
    dl.score,
    COALESCE(dl.confidence, ''),
    dl.price_delta_percent,
    COALESCE(dl.reasons, '{}'::jsonb),
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    pt.prices_transaction_area,
    pt.prices_transaction_price::bigint,
    pt.prices_transaction_price_per_square_meter::bigint,
    pt.prices_transaction_build_year,
    COALESCE(pt.prices_transaction_floor, ''),
    pt.prices_transaction_elevator,
    COALESCE(pt.prices_transaction_condition, ''),
    COALESCE(pt.prices_transaction_plot, ''),
    COALESCE(pt.prices_transaction_energy_class, ''),
    COALESCE(pt.prices_transaction_period_identifier, ''),
    COALESCE(pc.prices_city_name, ''),
    COALESCE(pn.prices_neighborhood_name, ''),
    COALESCE(ppc.prices_postal_code_code, postal.postal_postal_code_code, ''),
    pt.prices_transaction_created_at,
    pt.prices_transaction_updated_at,
    osr.sale_listing_id,
    COALESCE(osr.sale_listing_canonical_id, ''),
    COALESCE(osr.sale_listing_source_provider, ''),
    COALESCE(osr.sale_listing_source_kind, ''),
    COALESCE(osr.sale_listing_native_id, ''),
    COALESCE(osr.headline, ''),
    COALESCE(osr.address, ''),
    COALESCE(osr.city, ''),
    COALESCE(osr.postal, ''),
    osr.sale_listing_latitude,
    osr.sale_listing_longitude,
    osr.sale_listing_asking_price,
    osr.sale_listing_debt_free_price,
    osr.sale_listing_area_value,
    COALESCE(osr.room_layout, ''),
    COALESCE(osr.url, ''),
    COALESCE(osr.external_url_available, false),
    osr.sale_listing_first_seen_at,
    osr.sale_listing_last_seen_at,
    osr.sale_listing_updated_at,
    osr.sale_listing_previous_asking_price,
    osr.sale_listing_previous_debt_free_price,
    COALESCE(osr.source_link_status, ''),
    COALESCE(osr.source_link_method, ''),
    osr.property_offering_source_link_score,
    COALESCE(osr.availability_text, ''),
    COALESCE(osr.renovations_done_text, ''),
    COALESCE(osr.renovations_planned_text, ''),
    COALESCE(osr.additional_info_text, ''),
    COALESCE(osr.charges_text, ''),
    COALESCE(osr.insights_json, '[]'::jsonb)
FROM limited_listings sl
LEFT JOIN dedup_links dl ON dl.sale_listing_id = sl.sale_listing_id
LEFT JOIN origin.prices_transactions pt ON pt.prices_transaction_id = dl.prices_transaction_id
LEFT JOIN origin.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN origin.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes postal ON postal.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN offering_source_records osr ON osr.property_offering_id = sl.property_offering_id
ORDER BY sl.listing_rank, dl.link_rank NULLS LAST, dl.score DESC NULLS LAST, pt.prices_transaction_created_at DESC NULLS LAST, osr.sale_listing_source_provider, osr.sale_listing_native_id;

-- name: LookupAddressSourceCandidates :many
WITH selected AS (
    SELECT unnest(@listing_ids::uuid[])::uuid AS sale_listing_id
),
selected_links AS (
    SELECT
        selected.sale_listing_id AS selected_sale_listing_id,
        selected_link.target_id AS selected_property_offering_id
    FROM selected
    JOIN public.target_sources selected_link ON selected_link.target_type = 'listing'
        AND selected_link.source_type = 'source_listing'
        AND selected_link.source_id = selected.sale_listing_id
        AND selected_link.link_status <> 'rejected'
),
candidate_links AS (
    SELECT
        selected_links.selected_sale_listing_id,
        candidate_link.source_id AS candidate_sale_listing_id,
        selected_links.selected_property_offering_id,
        candidate_link.target_id AS candidate_property_offering_id,
        'target_source'::text AS direction,
        candidate_link.link_score AS match_score,
        CASE WHEN candidate_link.link_score >= 95 THEN 'high' WHEN candidate_link.link_score >= 80 THEN 'medium' ELSE 'low' END AS match_confidence,
        candidate_link.link_status AS match_status,
        candidate_link.link_reasons AS match_reasons,
        NULL::double precision AS price_delta_percent,
        candidate_link.updated_at AS match_created_at
    FROM selected_links
    JOIN public.target_sources candidate_link ON candidate_link.target_type = 'listing'
        AND candidate_link.target_id = selected_links.selected_property_offering_id
        AND candidate_link.source_type = 'source_listing'
        AND candidate_link.source_id <> selected_links.selected_sale_listing_id
        AND candidate_link.link_status <> 'rejected'
),
ranked_latest AS (
    SELECT
        candidate_links.selected_sale_listing_id,
        candidate_links.candidate_sale_listing_id,
        candidate_links.selected_property_offering_id,
        candidate_links.candidate_property_offering_id,
        candidate_links.direction,
        candidate_links.match_score,
        candidate_links.match_confidence,
        candidate_links.match_status,
        candidate_links.match_reasons,
        candidate_links.price_delta_percent,
        candidate_links.match_created_at,
        row_number() OVER (
            PARTITION BY selected_sale_listing_id
            ORDER BY match_score DESC, match_created_at DESC
        ) AS candidate_rank
    FROM candidate_links
)
SELECT
    latest.selected_sale_listing_id,
    candidate.sale_listing_id,
    candidate.sale_listing_canonical_id,
    candidate.sale_listing_source_provider,
    candidate.sale_listing_source_kind,
    candidate.sale_listing_native_id,
    COALESCE(candidate.sale_listing_headline, candidate.sale_listing_street_address, candidate.sale_listing_native_id) AS headline,
    COALESCE(candidate.sale_listing_street_address, '') AS address,
    COALESCE(candidate.sale_listing_city, candidate.sale_listing_city_norm, '') AS city,
    COALESCE(candidate.sale_listing_postal, candidate.sale_listing_postal_norm, '') AS postal,
    candidate.sale_listing_asking_price,
    candidate.sale_listing_debt_free_price,
    candidate.sale_listing_area_value,
    COALESCE(candidate.sale_listing_room_layout, '') AS room_layout,
    COALESCE(candidate.sale_listing_url, '') AS url,
    CASE
        WHEN candidate.sale_listing_source_provider = 'shortcut' AND candidate.sale_listing_source_kind = 'ad' THEN candidate.shortcut_ad_id IS NOT NULL AND COALESCE(candidate.sale_listing_url, '') <> '' AND candidate.sale_listing_last_seen_at >= now() - interval '7 days'
        WHEN candidate.sale_listing_source_provider = 'frontdoor' AND candidate.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
        WHEN candidate.sale_listing_source_provider = 'frontdoor' AND candidate.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
        ELSE false
    END AS external_url_available,
    latest.selected_property_offering_id,
    latest.candidate_property_offering_id,
    latest.direction,
    latest.match_status,
    latest.match_score,
    latest.match_confidence,
    latest.price_delta_percent,
    latest.match_reasons,
    latest.match_created_at
FROM ranked_latest latest
JOIN public.property_source_offerings candidate ON candidate.sale_listing_id = latest.candidate_sale_listing_id
LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = candidate.frontdoor_ad_id
LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = candidate.frontdoor_building_announcement_id
WHERE latest.candidate_rank <= 5
ORDER BY latest.selected_sale_listing_id, latest.match_score DESC, latest.match_created_at DESC
LIMIT 250;

-- name: LookupAddressRawTransactions :many
WITH raw_transactions (
    transaction_id,
    description,
    type,
    category,
    area,
    price,
    price_per_square_meter,
    build_year,
    floor,
    elevator,
    condition,
    plot,
    energy_class,
    period_identifier,
    city,
    neighborhood,
    postal,
    created_at,
    updated_at,
    is_matched,
    linked_to_lookup,
    candidate_to_lookup,
    matched_listing_count,
    matched_offering_count,
    matches,
    postal_history_rank
) AS (
SELECT
    pt.prices_transaction_id,
    COALESCE(pt.prices_transaction_description, ''),
    COALESCE(pt.prices_transaction_type, ''),
    COALESCE(pt.prices_transaction_category, ''),
    pt.prices_transaction_area,
    pt.prices_transaction_price::bigint,
    pt.prices_transaction_price_per_square_meter::bigint,
    pt.prices_transaction_build_year,
    COALESCE(pt.prices_transaction_floor, ''),
    pt.prices_transaction_elevator,
    COALESCE(pt.prices_transaction_condition, ''),
    COALESCE(pt.prices_transaction_plot, ''),
    COALESCE(pt.prices_transaction_energy_class, ''),
    COALESCE(pt.prices_transaction_period_identifier, ''),
    COALESCE(pc.prices_city_name, ''),
    COALESCE(pn.prices_neighborhood_name, ''),
    COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code, ''),
    pt.prices_transaction_created_at,
    pt.prices_transaction_updated_at,
    (
        EXISTS (
            SELECT 1
            FROM public.price_links price_link
            WHERE price_link.prices_transaction_id = pt.prices_transaction_id
                AND price_link.target_type = 'source_listing'
                AND price_link.link_status <> 'rejected'
        )
        OR EXISTS (
            SELECT 1
            FROM public.price_links price_link
            WHERE price_link.prices_transaction_id = pt.prices_transaction_id
                AND price_link.target_type = 'listing'
                AND price_link.link_status <> 'rejected'
        )
    ) AS is_matched,
    pt.prices_transaction_id = ANY($3::uuid[]) AS linked_to_lookup,
    pt.prices_transaction_id = ANY($4::uuid[]) AS candidate_to_lookup,
    (
        SELECT count(*)::integer
        FROM public.price_links price_link
        WHERE price_link.prices_transaction_id = pt.prices_transaction_id
            AND price_link.target_type = 'source_listing'
            AND price_link.link_status <> 'rejected'
    ) AS matched_listing_count,
    (
        SELECT count(*)::integer
        FROM public.price_links price_link
        WHERE price_link.prices_transaction_id = pt.prices_transaction_id
            AND price_link.target_type = 'listing'
            AND price_link.link_status <> 'rejected'
    ) AS matched_offering_count,
    COALESCE(
        (
            SELECT jsonb_agg(
                jsonb_build_object(
                    'type', match_type,
                    'id', id,
                    'offering_id', offering_id,
                    'canonical_id', canonical_id,
                    'source', source,
                    'native_id', native_id,
                    'headline', headline,
                    'address', address,
                    'city', city,
                    'postal', postal,
                    'status', status,
                    'method', method,
                    'score', score
                )
                ORDER BY match_type, headline, id
            )
            FROM (
                SELECT
                    'listing'::text AS match_type,
                    price_link.price_link_id::text AS id,
                    ''::text AS offering_id,
                    sl.sale_listing_canonical_id AS canonical_id,
                    sl.sale_listing_source_provider AS source,
                    sl.sale_listing_native_id AS native_id,
                    COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
                    COALESCE(sl.sale_listing_street_address, '') AS address,
                    COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS city,
                    COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS postal,
                    price_link.link_status AS status,
                    price_link.link_method AS method,
                    price_link.link_score AS score
                FROM public.price_links price_link
                JOIN public.property_source_offerings sl ON sl.sale_listing_id = price_link.target_id
                WHERE price_link.prices_transaction_id = pt.prices_transaction_id
                    AND price_link.target_type = 'source_listing'
                    AND price_link.link_status <> 'rejected'
                UNION ALL
                SELECT
                    'offering_source'::text AS match_type,
                    price_link.price_link_id::text || ':' || sl.sale_listing_id::text AS id,
                    price_link.target_id::text AS offering_id,
                    sl.sale_listing_canonical_id AS canonical_id,
                    sl.sale_listing_source_provider AS source,
                    sl.sale_listing_native_id AS native_id,
                    COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
                    COALESCE(sl.sale_listing_street_address, '') AS address,
                    COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS city,
                    COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS postal,
                    price_link.link_status AS status,
                    price_link.link_method AS method,
                    price_link.link_score AS score
                FROM public.price_links price_link
                JOIN public.target_sources source_link ON source_link.target_type = 'listing'
                    AND source_link.target_id = price_link.target_id
                    AND source_link.source_type = 'source_listing'
                    AND source_link.link_status <> 'rejected'
                JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
                WHERE price_link.prices_transaction_id = pt.prices_transaction_id
                    AND price_link.target_type = 'listing'
                    AND price_link.link_status <> 'rejected'
                LIMIT 8
            ) match
        ),
        '[]'::jsonb
    ) AS matches,
    row_number() OVER (ORDER BY pt.prices_transaction_created_at DESC, pt.prices_transaction_price ASC) AS postal_history_rank
FROM origin.prices_transactions pt
JOIN origin.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
JOIN origin.prices_cities pc ON pc.prices_city_id = pn.prices_city_id
LEFT JOIN origin.prices_postal_codes ppc_prices ON ppc_prices.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes ppc_scraped ON ppc_scraped.postal_postal_code_code = ppc_prices.prices_postal_code_code
LEFT JOIN origin.postal_municipalities pm_scraped ON pm_scraped.postal_municipality_id = ppc_scraped.postal_municipality_id
LEFT JOIN origin.postal_postal_codes ppc ON ppc.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
LEFT JOIN origin.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE (trim($1::text) = '' OR lower(COALESCE(pc.prices_city_name, pm_scraped.postal_municipality_name_fi, pm.postal_municipality_name_fi, '')) LIKE ('%' || lower(trim($1::text)) || '%'))
    AND (trim($2::text) = '' OR COALESCE(ppc_scraped.postal_postal_code_code, ppc.postal_postal_code_code, ppc_prices.prices_postal_code_code, '') = NULLIF(regexp_replace(trim(COALESCE($2::text, '')), '[^0-9]+', '', 'g'), ''))
)
SELECT
    transaction_id,
    description,
    type,
    category,
    area,
    price,
    price_per_square_meter,
    build_year,
    floor,
    elevator,
    condition,
    plot,
    energy_class,
    period_identifier,
    city,
    neighborhood,
    postal,
    created_at,
    updated_at,
    is_matched,
    linked_to_lookup,
    candidate_to_lookup,
    matched_listing_count,
    matched_offering_count,
    matches::jsonb AS matches
FROM raw_transactions
WHERE linked_to_lookup OR candidate_to_lookup OR postal_history_rank <= $5::int
ORDER BY linked_to_lookup DESC, candidate_to_lookup DESC, created_at DESC, price ASC;

-- name: LookupPostalCity :one
SELECT COALESCE(pm.postal_municipality_name_fi, '')::text AS city_fi,
    COALESCE(pm.postal_municipality_name_sv, '')::text AS city_sv
FROM origin.postal_postal_codes ppc
JOIN origin.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
WHERE ppc.postal_postal_code_code = NULLIF(regexp_replace(trim(COALESCE(sqlc.arg('postal')::text, '')), '[^0-9]+', '', 'g'), '')
ORDER BY pm.postal_municipality_name_fi
LIMIT 1;
