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
UPDATE origin.source_listings src
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
    LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
    LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    LEFT JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
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
