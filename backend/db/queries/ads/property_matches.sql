-- name: ListTransactionMatchPostals :many
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
        AND NOT EXISTS (
            SELECT 1
            FROM public.price_links source_link
            WHERE source_link.target_type = 'source_listing'
                AND source_link.target_id = sl.sale_listing_id
                AND source_link.link_status <> 'rejected'
        )
        AND sl.sale_listing_postal_norm IS NOT NULL
        AND NOT EXISTS (
            SELECT 1
            FROM public.price_links linked
            WHERE linked.prices_transaction_id = latest.prices_transaction_id
                AND linked.link_status <> 'rejected'
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
    COALESCE(max(sale_listing_prices_transaction_match_created_at)::text, '')::text AS latest_at
FROM potential
LEFT JOIN public.postal_postal_codes ppc ON ppc.postal_postal_code_code = potential.postal
LEFT JOIN public.postal_municipalities pm ON pm.postal_municipality_id = ppc.postal_municipality_id
GROUP BY postal, ppc.postal_postal_code_name_fi, pm.postal_municipality_name_fi
ORDER BY candidate_count DESC, postal
LIMIT sqlc.arg(limit_count)::int;

-- name: ListTransactionMatchCandidates :many
WITH latest_candidates AS (
    SELECT DISTINCT ON (c.sale_listing_id, c.prices_transaction_id)
        c.*
    FROM public.sale_listing_prices_transaction_match_candidates c
    ORDER BY c.sale_listing_id, c.prices_transaction_id, c.sale_listing_prices_transaction_match_created_at DESC
),
review_rows AS (
    SELECT
        c.sale_listing_prices_transaction_match_candidate_id::text AS id,
        c.sale_listing_prices_transaction_match_status AS status,
        'candidate'::text AS link_type,
        'match_candidate'::text AS link_method,
        c.sale_listing_prices_transaction_match_score AS score,
        c.sale_listing_prices_transaction_match_confidence AS confidence,
        c.sale_listing_prices_transaction_match_price_delta_percent AS price_delta_percent,
        c.sale_listing_prices_transaction_match_reasons AS reasons,
        c.sale_listing_prices_transaction_match_created_at AS created_at,
        c.sale_listing_id,
        c.prices_transaction_id
    FROM latest_candidates c
    UNION ALL
    SELECT
        pl.price_link_id::text || ':' || sl.sale_listing_id::text,
        pl.link_status,
        'listing'::text,
        pl.link_method,
        pl.link_score,
        ''::text,
        NULL::double precision,
        pl.link_reasons,
        pl.created_at,
        sl.sale_listing_id,
        pl.prices_transaction_id
    FROM public.price_links pl
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = pl.target_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    WHERE sqlc.narg(transaction_id)::uuid IS NOT NULL
        AND pl.target_type = 'listing'
        AND pl.link_status <> 'rejected'
        AND pl.prices_transaction_id = sqlc.narg(transaction_id)::uuid
        AND NOT EXISTS (
            SELECT 1
            FROM latest_candidates c
            WHERE c.sale_listing_id = sl.sale_listing_id
                AND c.prices_transaction_id = pl.prices_transaction_id
        )
    UNION ALL
    SELECT
        pl.price_link_id::text,
        pl.link_status,
        'source_listing'::text,
        pl.link_method,
        pl.link_score,
        ''::text,
        NULL::double precision,
        pl.link_reasons,
        pl.created_at,
        pl.target_id,
        pl.prices_transaction_id
    FROM public.price_links pl
    WHERE sqlc.narg(transaction_id)::uuid IS NOT NULL
        AND pl.target_type = 'source_listing'
        AND pl.prices_transaction_id = sqlc.narg(transaction_id)::uuid
        AND pl.link_status <> 'rejected'
        AND NOT EXISTS (
            SELECT 1
            FROM latest_candidates c
            WHERE c.sale_listing_id = pl.target_id
                AND c.prices_transaction_id = pl.prices_transaction_id
        )
        AND NOT EXISTS (
            SELECT 1
            FROM public.price_links listing_link
            JOIN public.target_sources source_link ON source_link.target_type = 'listing'
                AND source_link.target_id = listing_link.target_id
                AND source_link.source_type = 'source_listing'
                AND source_link.link_status <> 'rejected'
            WHERE listing_link.target_type = 'listing'
                AND listing_link.prices_transaction_id = pl.prices_transaction_id
                AND listing_link.link_status <> 'rejected'
                AND source_link.source_id = pl.target_id
        )
)
SELECT
    latest.id,
    latest.status,
    latest.link_type,
    latest.link_method,
    latest.score::int4,
    latest.confidence,
    latest.price_delta_percent,
    latest.reasons,
    COALESCE(latest.created_at::text, '')::text AS created_at,
    sl.sale_listing_id::text AS listing_id,
    COALESCE(source_link.target_id::text, '')::text AS offering_id,
    sl.sale_listing_canonical_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_native_id,
    COALESCE(sl.sale_listing_url, '') AS listing_url,
    CASE
        WHEN sl.sale_listing_source_provider = 'shortcut' AND sl.sale_listing_source_kind = 'ad' THEN sl.shortcut_ad_id IS NOT NULL AND COALESCE(sl.sale_listing_url, '') <> '' AND sl.sale_listing_last_seen_at >= now() - interval '7 days'
        WHEN sl.sale_listing_source_provider = 'frontdoor' AND sl.sale_listing_source_kind = 'ad' THEN fa.frontdoor_ad_id IS NOT NULL AND fa.frontdoor_ad_page_not_found = false
        WHEN sl.sale_listing_source_provider = 'frontdoor' AND sl.sale_listing_source_kind = 'announcement' THEN COALESCE(fba.frontdoor_building_announcement_published, false)
        ELSE false
    END AS external_url_available,
    COALESCE(sl.sale_listing_headline, '') AS listing_headline,
    COALESCE(sl.sale_listing_street_address, '') AS listing_street_address,
    COALESCE(sl.sale_listing_city, sl.sale_listing_city_norm, '') AS listing_city,
    COALESCE(sl.sale_listing_postal, sl.sale_listing_postal_norm, '') AS listing_postal,
    COALESCE(sl.sale_listing_room_layout, '') AS listing_room_layout,
    COALESCE(sl.sale_listing_condition, '') AS listing_condition,
    COALESCE(public.fnc__condition_match_code(sl.sale_listing_condition), '')::text AS listing_condition_match_code,
    sl.sale_listing_area_value,
    sl.sale_listing_asking_price,
    sl.sale_listing_price_per_m2,
    sl.sale_listing_build_year,
    sl.sale_listing_floor_level,
    sl.sale_listing_total_floors,
    sl.sale_listing_elevator,
    COALESCE(sl.sale_listing_energy_efficiency_match_code, '') AS listing_energy_match_code,
    COALESCE(sl.sale_listing_energy_efficiency_label, '') AS listing_energy_label,
    COALESCE(sl.sale_listing_plot_type_raw, '') AS listing_plot_ownership_raw,
    sl.sale_listing_plot_owned,
    COALESCE(sl.sale_listing_first_seen_at::text, '')::text AS listing_first_seen_at,
    COALESCE(sl.sale_listing_last_seen_at::text, '')::text AS listing_last_seen_at,
    pt.prices_transaction_id::text AS transaction_id_text,
    COALESCE(pt.prices_transaction_description, '') AS transaction_description,
    COALESCE(pt.prices_transaction_type, '') AS transaction_type,
    COALESCE(pt.prices_transaction_category, '') AS transaction_category,
    pt.prices_transaction_area,
    pt.prices_transaction_price,
    pt.prices_transaction_price_per_square_meter,
    pt.prices_transaction_build_year,
    COALESCE(pt.prices_transaction_floor, '') AS transaction_floor,
    pt.prices_transaction_elevator,
    COALESCE(pt.prices_transaction_condition, '') AS transaction_condition,
    COALESCE(public.fnc__condition_match_code(pt.prices_transaction_condition), '')::text AS transaction_condition_match_code,
    COALESCE(pt.prices_transaction_plot, '') AS transaction_plot,
    pt.prices_transaction_plot_owned,
    COALESCE(pt.prices_transaction_energy_class, '') AS transaction_energy_class,
    COALESCE(public.fnc__prices_transaction_energy_match_code(pt.prices_transaction_energy_class), '')::text AS transaction_energy_match_code,
    COALESCE(pt.prices_transaction_period_identifier, '') AS transaction_period_identifier,
    COALESCE(pt.prices_transaction_created_at::text, '')::text AS transaction_created_at
FROM review_rows latest
JOIN public.property_source_offerings sl ON sl.sale_listing_id = latest.sale_listing_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
LEFT JOIN public.target_sources source_link ON source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.source_id = sl.sale_listing_id
    AND source_link.link_status <> 'rejected'
JOIN public.prices_transactions pt ON pt.prices_transaction_id = latest.prices_transaction_id
WHERE (sqlc.narg(transaction_id)::uuid IS NOT NULL OR latest.status = ANY(ARRAY['candidate'::text, 'ambiguous'::text]))
    AND (
        sqlc.narg(transaction_id)::uuid IS NOT NULL
        OR NOT EXISTS (
            SELECT 1
            FROM public.price_links source_link
            WHERE source_link.target_type = 'source_listing'
                AND source_link.target_id = sl.sale_listing_id
                AND source_link.link_status <> 'rejected'
        )
    )
    AND (sqlc.narg(postal)::text IS NULL OR sl.sale_listing_postal_norm = public.fnc__normalize_postal(sqlc.narg(postal)::text))
    AND (sqlc.narg(status)::text IS NULL OR latest.status = sqlc.narg(status)::text)
    AND (sqlc.narg(transaction_id)::uuid IS NULL OR pt.prices_transaction_id = sqlc.narg(transaction_id)::uuid)
    AND (sqlc.narg(transaction_id)::uuid IS NOT NULL OR NOT EXISTS (
        SELECT 1
        FROM public.price_links linked
        WHERE linked.prices_transaction_id = latest.prices_transaction_id
            AND linked.link_status <> 'rejected'
    ))
ORDER BY
    latest.score DESC,
    latest.price_delta_percent ASC NULLS LAST,
    sl.sale_listing_postal_norm,
    sl.sale_listing_street_address
LIMIT sqlc.arg(limit_count)::int;
