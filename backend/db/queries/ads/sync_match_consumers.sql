-- name: LoadPricesMatchSaleListing :one
SELECT
    doc.primary_source_listing_id::text AS id,
    doc.last_seen_at AS sale_listing_last_seen_at,
    COALESCE(price_link.prices_transaction_id::text, '')::text AS transaction_id,
    state.match_status AS sale_listing_prices_match_status,
    COALESCE(state.attempt_count, 0)::int4 AS sale_listing_prices_match_attempt_count,
    state.expires_at AS sale_listing_prices_match_expires_at
FROM public.listing_search_documents doc
LEFT JOIN public.listing_price_match_states state ON state.source_listing_id = doc.primary_source_listing_id
LEFT JOIN LATERAL (
    SELECT pl.prices_transaction_id
    FROM public.price_links pl
    WHERE pl.link_status <> 'rejected'
        AND (
            (pl.target_type = 'source_listing' AND pl.target_id = doc.primary_source_listing_id)
            OR (pl.target_type = 'listing' AND pl.target_id = doc.property_offering_id)
        )
    ORDER BY pl.link_score DESC, pl.updated_at DESC
    LIMIT 1
) price_link ON true
WHERE doc.primary_source_listing_id = @sale_listing_id::uuid
    AND doc.listing_status = 'active'
    AND doc.kind = 'ad'
ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
LIMIT 1;

-- name: UpdatePricesMatchState :exec
WITH source_doc AS (
    SELECT doc.primary_source_listing_id, doc.listing_id
    FROM public.listing_search_documents doc
    WHERE doc.primary_source_listing_id = @sale_listing_id::uuid
        AND doc.listing_status = 'active'
        AND doc.kind = 'ad'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
)
INSERT INTO public.listing_price_match_states (
    source_listing_id,
    listing_id,
    match_status,
    next_attempt_at,
    last_attempted_at,
    attempt_count,
    run_id,
    expires_at,
    created_at,
    updated_at
)
SELECT
    source_doc.primary_source_listing_id,
    source_doc.listing_id,
    @status::text,
    sqlc.narg('next_attempt_at')::timestamptz,
    now(),
    1,
    sqlc.narg('run_id')::uuid,
    sqlc.narg('expires_at')::timestamptz,
    now(),
    now()
FROM source_doc
ON CONFLICT (source_listing_id) DO UPDATE SET
    listing_id = EXCLUDED.listing_id,
    match_status = EXCLUDED.match_status,
    next_attempt_at = EXCLUDED.next_attempt_at,
    last_attempted_at = EXCLUDED.last_attempted_at,
    attempt_count = public.listing_price_match_states.attempt_count + 1,
    run_id = COALESCE(EXCLUDED.run_id, public.listing_price_match_states.run_id),
    expires_at = COALESCE(EXCLUDED.expires_at, public.listing_price_match_states.expires_at),
    updated_at = now();

-- name: BackfillBuildingCoordinates :execrows
UPDATE public.physical_buildings pb
SET physical_building_latitude = coordinates.lat,
    physical_building_longitude = coordinates.lng,
    physical_building_updated_at = now()
FROM (
    SELECT DISTINCT ON (pu.physical_building_id)
        pu.physical_building_id,
        COALESCE(doc.latitude, postgis.ST_Y(hc.housing_company_geom)::double precision) AS lat,
        COALESCE(doc.longitude, postgis.ST_X(hc.housing_company_geom)::double precision) AS lng,
        doc.last_seen_at
    FROM public.property_units pu
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    LEFT JOIN public.property_offerings po ON po.property_unit_id = pu.property_unit_id
    LEFT JOIN public.listing_search_documents doc ON doc.property_offering_id = po.property_offering_id
        AND doc.listing_status = 'active'
    WHERE pu.physical_building_id IS NOT NULL
    ORDER BY pu.physical_building_id,
        (doc.latitude IS NOT NULL AND doc.longitude IS NOT NULL) DESC,
        doc.last_seen_at DESC NULLS LAST
) coordinates
WHERE pb.physical_building_id = coordinates.physical_building_id
  AND coordinates.lat IS NOT NULL
  AND coordinates.lng IS NOT NULL
  AND (pb.physical_building_latitude IS NULL OR pb.physical_building_longitude IS NULL);

-- name: ListDimensionLayerBackfillListingIDs :many
SELECT source_listing_id
FROM origin.source_listings
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
WITH queued AS (
    UPDATE public.property_dimension_dirty_targets
    SET queued_at = now()
    WHERE target_type = @target_type::text
        AND target_id = @target_id::uuid
        AND (resolved_at IS NULL OR resolved_at < dirty_at)
    RETURNING 1
)
SELECT count(*)::integer AS count FROM queued;

-- name: ListPricesMatchFanoutListings :many
SELECT doc.primary_source_listing_id::text AS sale_listing_id, COALESCE(state.attempt_count, 0)::int4 AS attempt_count
FROM public.listing_search_documents doc
LEFT JOIN public.listing_price_match_states state ON state.source_listing_id = doc.primary_source_listing_id
WHERE doc.kind = 'ad'
    AND doc.listing_status = 'active'
    AND doc.primary_source_listing_id IS NOT NULL
    AND doc.last_seen_at IS NOT NULL
    AND doc.last_seen_at <= now() - interval '7 days'
    AND doc.last_seen_at >= now() - interval '4 months'
    AND COALESCE(state.match_status, 'pending') IN ('pending', 'deferred', 'noop')
    AND COALESCE(state.next_attempt_at, doc.last_seen_at + interval '7 days') <= now()
    AND NOT EXISTS (
        SELECT 1
        FROM public.price_links pl
        WHERE pl.link_status <> 'rejected'
            AND (
                (pl.target_type = 'source_listing' AND pl.target_id = doc.primary_source_listing_id)
                OR (pl.target_type = 'listing' AND pl.target_id = doc.property_offering_id)
            )
    )
ORDER BY COALESCE(state.next_attempt_at, doc.last_seen_at + interval '7 days'), doc.last_seen_at
LIMIT @limit_count::int4;

-- name: ListCanonicalizeSourceAdsFanout :many
(SELECT 'frontdoor_ad'::text AS source_table, frontdoor_ad_id::text AS source_id
 FROM origin.frontdoor_ads
 WHERE frontdoor_ad_data IS NOT NULL
     AND (frontdoor_ad_data_hash IS NULL
         OR frontdoor_ad_data_normalized_at IS NULL
         OR frontdoor_ad_data_changed_at > frontdoor_ad_data_normalized_at
         OR frontdoor_ad_data_normalized_version < @version::int4)
 ORDER BY frontdoor_ad_updated_at ASC
 LIMIT @limit_count::int4)
UNION ALL
(SELECT 'shortcut_ad'::text AS source_table, shortcut_ad_id::text AS source_id
 FROM origin.shortcut_ads
 WHERE shortcut_ad_data IS NOT NULL
     AND (shortcut_ad_data_hash IS NULL
         OR shortcut_ad_data_normalized_at IS NULL
         OR shortcut_ad_data_changed_at > shortcut_ad_data_normalized_at
         OR shortcut_ad_data_normalized_version < @version::int4)
 ORDER BY shortcut_ad_updated_at ASC NULLS FIRST
 LIMIT @limit_count::int4)
UNION ALL
(SELECT 'frontdoor_building_announcement'::text AS source_table, frontdoor_building_announcement_id::text AS source_id
 FROM origin.frontdoor_building_announcements
 WHERE frontdoor_building_announcement_rent_period IS NULL
     AND frontdoor_building_announcement_rental_unique_no IS NULL
     AND (frontdoor_building_announcement_data_normalized_at IS NULL
         OR frontdoor_building_announcement_data_normalized_version < @version::int4)
 ORDER BY frontdoor_building_announcement_last_seen_at ASC
 LIMIT @limit_count::int4);
