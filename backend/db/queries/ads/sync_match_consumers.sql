-- name: LoadPricesMatchSaleListing :one
SELECT
    sale_listing_id::text AS id,
    sale_listing_last_seen_at,
    COALESCE(prices_transaction_id::text, '')::text AS transaction_id,
    sale_listing_prices_match_status,
    sale_listing_prices_match_attempt_count,
    sale_listing_prices_match_expires_at
FROM public.property_source_offerings
WHERE sale_listing_id = @sale_listing_id::uuid;

-- name: UpdatePricesMatchState :exec
WITH updated_source AS (
    UPDATE public.property_source_offerings
    SET
        sale_listing_prices_match_status = @status::text,
        sale_listing_prices_match_next_attempt_at = sqlc.narg('next_attempt_at')::timestamptz,
        sale_listing_prices_match_last_attempted_at = now(),
        sale_listing_prices_match_attempt_count = sale_listing_prices_match_attempt_count + 1,
        sale_listing_prices_match_run_id = COALESCE(sqlc.narg('run_id')::uuid, sale_listing_prices_match_run_id),
        sale_listing_prices_match_expires_at = COALESCE(sqlc.narg('expires_at')::timestamptz, sale_listing_prices_match_expires_at),
        sale_listing_updated_at = now()
    WHERE sale_listing_id = @sale_listing_id::uuid
    RETURNING sale_listing_id, sale_listing_updated_at
)
UPDATE public.source_listings src
SET normalized_at = updated_source.sale_listing_updated_at,
    updated_at = updated_source.sale_listing_updated_at
FROM updated_source
WHERE src.source_listing_id = updated_source.sale_listing_id;

-- name: BackfillBuildingCoordinates :execrows
UPDATE public.physical_buildings pb
SET physical_building_latitude = coordinates.lat,
    physical_building_longitude = coordinates.lng,
    physical_building_updated_at = now()
FROM (
    SELECT DISTINCT ON (pu.physical_building_id)
        pu.physical_building_id,
        COALESCE(fb.frontdoor_building_latitude, sb.shortcut_building_latitude, sl.sale_listing_latitude, postgis.ST_Y(hc.housing_company_geom)::double precision) AS lat,
        COALESCE(fb.frontdoor_building_longitude, sb.shortcut_building_longitude, sl.sale_listing_longitude, postgis.ST_X(hc.housing_company_geom)::double precision) AS lng
    FROM public.property_units pu
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    LEFT JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = po.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    LEFT JOIN public.property_source_offerings sl ON sl.sale_listing_id = source_link.source_id
    LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE pu.physical_building_id IS NOT NULL
    ORDER BY pu.physical_building_id,
        (fb.frontdoor_building_latitude IS NOT NULL AND fb.frontdoor_building_longitude IS NOT NULL) DESC,
        (sb.shortcut_building_latitude IS NOT NULL AND sb.shortcut_building_longitude IS NOT NULL) DESC,
        (sl.sale_listing_latitude IS NOT NULL AND sl.sale_listing_longitude IS NOT NULL) DESC,
        sl.sale_listing_last_seen_at DESC NULLS LAST
) coordinates
WHERE pb.physical_building_id = coordinates.physical_building_id
  AND coordinates.lat IS NOT NULL
  AND coordinates.lng IS NOT NULL
  AND (pb.physical_building_latitude IS NULL OR pb.physical_building_longitude IS NULL);

-- name: ListDimensionLayerBackfillListingIDs :many
SELECT source_listing_id
FROM public.source_listings
WHERE (sqlc.narg('cursor')::uuid IS NULL OR source_listing_id > sqlc.narg('cursor')::uuid)
ORDER BY source_listing_id
LIMIT @limit_count::int4;

-- name: ListDirtyDimensionTargets :many
SELECT target_type, target_id, dirty_at
FROM public.property_dimension_dirty_targets
WHERE (resolved_at IS NULL OR resolved_at < dirty_at)
    AND (queued_at IS NULL OR queued_at < dirty_at OR queued_at < now() - interval '30 minutes')
ORDER BY dirty_at
LIMIT @limit_count::int4;

-- name: MarkDimensionTargetQueued :one
SELECT public.fnc__mark_dimension_target_queued(@target_type::text, @target_id::uuid);

-- name: ListPricesMatchFanoutListings :many
SELECT sale_listing_id::text AS sale_listing_id, COALESCE(sale_listing_prices_match_attempt_count, 0)::int4 AS attempt_count
FROM public.property_source_offerings
WHERE sale_listing_source_kind = 'ad'
    AND prices_transaction_id IS NULL
    AND sale_listing_last_seen_at IS NOT NULL
    AND sale_listing_last_seen_at <= now() - interval '7 days'
    AND sale_listing_last_seen_at >= now() - interval '4 months'
    AND COALESCE(sale_listing_prices_match_status, 'pending') IN ('pending', 'deferred', 'noop')
    AND COALESCE(sale_listing_prices_match_next_attempt_at, sale_listing_last_seen_at + interval '7 days') <= now()
ORDER BY COALESCE(sale_listing_prices_match_next_attempt_at, sale_listing_last_seen_at + interval '7 days'), sale_listing_last_seen_at
LIMIT @limit_count::int4;

-- name: ListCanonicalizeSourceAdsFanout :many
(SELECT 'frontdoor_ad'::text AS source_table, frontdoor_ad_id::text AS source_id
 FROM public.frontdoor_ads
 WHERE frontdoor_ad_data IS NOT NULL
     AND (frontdoor_ad_data_hash IS NULL
         OR frontdoor_ad_data_normalized_at IS NULL
         OR frontdoor_ad_data_changed_at > frontdoor_ad_data_normalized_at
         OR frontdoor_ad_data_normalized_version < @version::int4)
 ORDER BY frontdoor_ad_updated_at ASC
 LIMIT @limit_count::int4)
UNION ALL
(SELECT 'shortcut_ad'::text AS source_table, shortcut_ad_id::text AS source_id
 FROM public.shortcut_ads
 WHERE shortcut_ad_data IS NOT NULL
     AND (shortcut_ad_data_hash IS NULL
         OR shortcut_ad_data_normalized_at IS NULL
         OR shortcut_ad_data_changed_at > shortcut_ad_data_normalized_at
         OR shortcut_ad_data_normalized_version < @version::int4)
 ORDER BY shortcut_ad_updated_at ASC NULLS FIRST
 LIMIT @limit_count::int4)
UNION ALL
(SELECT 'frontdoor_building_announcement'::text AS source_table, frontdoor_building_announcement_id::text AS source_id
 FROM public.frontdoor_building_announcements
 WHERE frontdoor_building_announcement_rent_period IS NULL
     AND frontdoor_building_announcement_rental_unique_no IS NULL
     AND (frontdoor_building_announcement_data_normalized_at IS NULL
         OR frontdoor_building_announcement_data_normalized_version < @version::int4)
 ORDER BY frontdoor_building_announcement_last_seen_at ASC
 LIMIT @limit_count::int4);

-- name: ListCanonicalMatchFanoutListings :many
SELECT sl.sale_listing_id::text AS sale_listing_id, COALESCE(sl.sale_listing_source_match_attempt_count, 0)::int4 AS attempt_count
FROM public.property_source_offerings sl
JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
WHERE sl.sale_listing_source_kind = 'ad'
    AND source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.link_status <> 'rejected'
    AND source_link.link_method <> 'manual'
    AND COALESCE(sl.sale_listing_source_match_status, 'pending') IN ('pending', 'deferred', 'noop')
    AND COALESCE(sl.sale_listing_source_match_next_attempt_at, sl.sale_listing_updated_at) <= now()
ORDER BY COALESCE(sl.sale_listing_source_match_next_attempt_at, sl.sale_listing_updated_at), sl.sale_listing_updated_at
LIMIT @limit_count::int4;

-- name: LoadCanonicalMatchSaleListing :one
SELECT
    sl.sale_listing_id::text AS id,
    source_link.link_method,
    source_link.link_status,
    sl.sale_listing_source_match_status,
    sl.sale_listing_source_match_attempt_count
FROM public.property_source_offerings sl
LEFT JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
    AND source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.link_status <> 'rejected'
WHERE sl.sale_listing_id = @sale_listing_id::uuid;

-- name: RunCanonicalSourceMatchForSaleListing :one
WITH base AS (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_unit_match_key,
        link.target_id
    FROM public.property_source_offerings sl
    JOIN public.target_sources link ON link.target_type = 'listing'
        AND link.source_type = 'source_listing'
        AND link.source_id = sl.sale_listing_id
        AND link.link_status <> 'rejected'
    WHERE sl.sale_listing_id = @sale_listing_id::uuid
        AND sl.sale_listing_source_kind = 'ad'
        AND COALESCE(sl.sale_listing_unit_match_key, '') <> ''
    ORDER BY CASE WHEN link.link_status = 'confirmed' THEN 0 ELSE 1 END, link.link_score DESC
    LIMIT 1
),
candidates AS (
    SELECT
        candidate.sale_listing_id,
        candidate.sale_listing_source_provider,
        candidate.sale_listing_source_kind,
        candidate.sale_listing_native_id,
        candidate.sale_listing_first_seen_at,
        candidate.sale_listing_last_seen_at,
        active_link.target_source_id AS active_target_source_id,
        active_link.target_id AS active_target_id,
        active_link.link_method AS active_link_method
    FROM base
    JOIN public.property_source_offerings candidate ON candidate.sale_listing_unit_match_key = base.sale_listing_unit_match_key
        AND candidate.sale_listing_id <> base.sale_listing_id
        AND candidate.sale_listing_source_kind = 'ad'
    LEFT JOIN public.target_sources active_link ON active_link.target_type = 'listing'
        AND active_link.source_type = 'source_listing'
        AND active_link.source_id = candidate.sale_listing_id
        AND active_link.link_status <> 'rejected'
    WHERE candidate.sale_listing_source_provider <> base.sale_listing_source_provider
),
linkable AS (
    SELECT
        candidates.sale_listing_id,
        candidates.sale_listing_source_provider,
        candidates.sale_listing_source_kind,
        candidates.sale_listing_native_id,
        candidates.sale_listing_first_seen_at,
        candidates.sale_listing_last_seen_at,
        candidates.active_target_source_id,
        candidates.active_target_id,
        candidates.active_link_method
    FROM candidates
    WHERE active_target_source_id IS NULL
        OR active_target_id = (SELECT target_id FROM base)
),
inserted AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        'listing',
        base.target_id,
        'source_listing',
        linkable.sale_listing_id,
        'confirmed',
        'source_match_auto',
        100,
        jsonb_build_object('method', 'unit_match_key_exact', 'matched_source_listing_id', base.sale_listing_id, 'provider', linkable.sale_listing_source_provider, 'native_id', linkable.sale_listing_native_id),
        linkable.sale_listing_first_seen_at,
        linkable.sale_listing_last_seen_at,
        now(),
        now()
    FROM base
    JOIN linkable ON true
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_status ELSE EXCLUDED.link_status END,
        link_method = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_method ELSE EXCLUDED.link_method END,
        link_score = GREATEST(public.target_sources.link_score, EXCLUDED.link_score),
        link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_source_id
)
SELECT
    @run_id::text AS run_id,
    (SELECT count(*)::int4 FROM candidates) AS candidates,
    (SELECT count(*)::int4 FROM inserted) AS auto_linked,
    (SELECT count(*)::int4 FROM candidates WHERE active_target_source_id IS NOT NULL AND active_target_id <> (SELECT target_id FROM base)) AS ambiguous;

-- name: RunCanonicalSourceMatchBackfill :one
WITH base AS (
    SELECT
        link.target_id,
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_unit_match_key
    FROM public.target_sources link
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = link.source_id
    WHERE link.target_type = 'listing'
        AND link.source_type = 'source_listing'
        AND link.link_status <> 'rejected'
        AND link.link_method <> 'manual'
        AND sl.sale_listing_source_kind = 'ad'
        AND COALESCE(sl.sale_listing_unit_match_key, '') <> ''
),
candidate_pairs AS (
    SELECT
        base.target_id,
        base.sale_listing_id AS matched_sale_listing_id,
        candidate.sale_listing_id,
        candidate.sale_listing_source_provider,
        candidate.sale_listing_source_kind,
        candidate.sale_listing_native_id,
        candidate.sale_listing_first_seen_at,
        candidate.sale_listing_last_seen_at,
        active_link.target_source_id AS active_target_source_id,
        active_link.target_id AS active_target_id,
        row_number() OVER (
            PARTITION BY candidate.sale_listing_id
            ORDER BY base.target_id, base.sale_listing_id
        ) AS candidate_rank
    FROM base
    JOIN public.property_source_offerings candidate ON candidate.sale_listing_unit_match_key = base.sale_listing_unit_match_key
        AND candidate.sale_listing_id <> base.sale_listing_id
        AND candidate.sale_listing_source_kind = 'ad'
        AND candidate.sale_listing_source_provider <> base.sale_listing_source_provider
    LEFT JOIN public.target_sources active_link ON active_link.target_type = 'listing'
        AND active_link.source_type = 'source_listing'
        AND active_link.source_id = candidate.sale_listing_id
        AND active_link.link_status <> 'rejected'
),
linkable AS (
    SELECT
        target_id,
        matched_sale_listing_id,
        sale_listing_id,
        sale_listing_source_provider,
        sale_listing_source_kind,
        sale_listing_native_id,
        sale_listing_first_seen_at,
        sale_listing_last_seen_at,
        active_target_source_id,
        active_target_id,
        candidate_rank
    FROM candidate_pairs
    WHERE candidate_rank = 1
        AND (
            active_target_source_id IS NULL
            OR active_target_id = target_id
        )
),
inserted AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        'listing',
        target_id,
        'source_listing',
        sale_listing_id,
        'confirmed',
        'source_match_auto',
        100,
        jsonb_build_object('method', 'unit_match_key_exact_backfill', 'matched_source_listing_id', matched_sale_listing_id, 'provider', sale_listing_source_provider, 'native_id', sale_listing_native_id),
        sale_listing_first_seen_at,
        sale_listing_last_seen_at,
        now(),
        now()
    FROM linkable
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_status ELSE EXCLUDED.link_status END,
        link_method = CASE WHEN public.target_sources.link_method = 'manual' THEN public.target_sources.link_method ELSE EXCLUDED.link_method END,
        link_score = GREATEST(public.target_sources.link_score, EXCLUDED.link_score),
        link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_source_id
)
SELECT
    @run_id::text AS run_id,
    (SELECT count(*)::int4 FROM candidate_pairs) AS candidates,
    (SELECT count(*)::int4 FROM inserted) AS auto_linked,
    (SELECT count(*)::int4 FROM candidate_pairs WHERE active_target_source_id IS NOT NULL AND active_target_id <> target_id) AS ambiguous;

-- name: UpdateCanonicalSourceMatchState :exec
WITH updated_source AS (
    UPDATE public.property_source_offerings
    SET
        sale_listing_source_match_status = @status::text,
        sale_listing_source_match_next_attempt_at = sqlc.narg('next_attempt_at')::timestamptz,
        sale_listing_source_match_last_attempted_at = now(),
        sale_listing_source_match_attempt_count = sale_listing_source_match_attempt_count + 1,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = @sale_listing_id::uuid
    RETURNING sale_listing_id, sale_listing_updated_at
)
UPDATE public.source_listings src
SET normalized_at = updated_source.sale_listing_updated_at,
    updated_at = updated_source.sale_listing_updated_at
FROM updated_source
WHERE src.source_listing_id = updated_source.sale_listing_id;
