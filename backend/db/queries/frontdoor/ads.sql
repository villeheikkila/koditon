-- name: GetFrontdoorAdByExternalID :one
SELECT * FROM public.frontdoor_ads
WHERE frontdoor_ad_external_id = $1;

-- name: GetFrontdoorAdByID :one
SELECT * FROM public.frontdoor_ads
WHERE frontdoor_ad_id = $1;

-- name: ListFrontdoorAds :many
SELECT * FROM public.frontdoor_ads
ORDER BY frontdoor_ad_last_seen_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedFrontdoorAds :many
SELECT * FROM public.frontdoor_ads
WHERE frontdoor_ad_processed_at IS NULL AND frontdoor_ad_page_not_found = false
ORDER BY frontdoor_ad_first_seen_at ASC
LIMIT $1;

-- name: UpsertFrontdoorAds :exec
INSERT INTO public.frontdoor_ads (frontdoor_ad_external_id)
SELECT unnest($1::text[])
ON CONFLICT (frontdoor_ad_external_id) DO UPDATE SET
    frontdoor_ad_last_seen_at = NOW(),
    frontdoor_ad_updated_at = NOW();

-- name: UpsertFrontdoorAdFromSitemap :one
INSERT INTO public.frontdoor_ads (
    frontdoor_ad_external_id,
    frontdoor_ad_url,
    frontdoor_ad_first_seen_at,
    frontdoor_ad_last_seen_at,
    frontdoor_ad_updated_at
) VALUES ($1, $2, now(), now(), now())
ON CONFLICT (frontdoor_ad_external_id) DO UPDATE
SET frontdoor_ad_last_seen_at = now(),
    frontdoor_ad_url = COALESCE(EXCLUDED.frontdoor_ad_url, frontdoor_ads.frontdoor_ad_url)
RETURNING *;

-- name: BatchUpsertFrontdoorAdsFromSitemap :many
INSERT INTO public.frontdoor_ads (
    frontdoor_ad_external_id,
    frontdoor_ad_url,
    frontdoor_ad_first_seen_at,
    frontdoor_ad_last_seen_at,
    frontdoor_ad_updated_at
)
SELECT UNNEST($1::text[]), UNNEST($2::text[]), now(), now(), now()
ON CONFLICT (frontdoor_ad_external_id) DO UPDATE
SET frontdoor_ad_last_seen_at = now(),
    frontdoor_ad_url = COALESCE(EXCLUDED.frontdoor_ad_url, frontdoor_ads.frontdoor_ad_url)
RETURNING *;

-- name: UpdateFrontdoorAdData :exec
UPDATE public.frontdoor_ads
SET frontdoor_ad_data = sqlc.arg(frontdoor_ad_data)::jsonb,
    frontdoor_ad_data_hash = sqlc.arg(frontdoor_ad_data_hash),
    frontdoor_ad_data_hash_algorithm = sqlc.arg(frontdoor_ad_data_hash_algorithm),
    frontdoor_ad_data_changed_at = CASE WHEN frontdoor_ad_data_hash IS DISTINCT FROM sqlc.arg(frontdoor_ad_data_hash) THEN now() ELSE frontdoor_ad_data_changed_at END,
    frontdoor_ad_data_normalized_at = CASE WHEN frontdoor_ad_data_hash IS DISTINCT FROM sqlc.arg(frontdoor_ad_data_hash) THEN NULL ELSE frontdoor_ad_data_normalized_at END,
    frontdoor_ad_data_normalized_version = CASE WHEN frontdoor_ad_data_hash IS DISTINCT FROM sqlc.arg(frontdoor_ad_data_hash) THEN 0 ELSE frontdoor_ad_data_normalized_version END,
    frontdoor_ad_processed_at = NOW(),
    frontdoor_ad_updated_at = NOW(),
    frontdoor_ad_page_not_found = false
WHERE frontdoor_ad_external_id = sqlc.arg(frontdoor_ad_external_id);

-- name: ListFrontdoorAdsMissingDataHash :many
SELECT frontdoor_ad_external_id, frontdoor_ad_data
FROM public.frontdoor_ads
WHERE frontdoor_ad_data IS NOT NULL
  AND frontdoor_ad_data_hash IS NULL
ORDER BY frontdoor_ad_updated_at ASC NULLS FIRST, frontdoor_ad_first_seen_at ASC
LIMIT $1;

-- name: BackfillFrontdoorAdDataHash :exec
UPDATE public.frontdoor_ads
SET frontdoor_ad_data = sqlc.arg(frontdoor_ad_data)::jsonb,
    frontdoor_ad_data_hash = sqlc.arg(frontdoor_ad_data_hash),
    frontdoor_ad_data_hash_algorithm = sqlc.arg(frontdoor_ad_data_hash_algorithm),
    frontdoor_ad_data_changed_at = COALESCE(frontdoor_ad_data_changed_at, frontdoor_ad_updated_at, frontdoor_ad_processed_at, now())
WHERE frontdoor_ad_external_id = sqlc.arg(frontdoor_ad_external_id);

-- name: MarkFrontdoorAdDataNormalized :exec
UPDATE public.frontdoor_ads
SET frontdoor_ad_data_normalized_at = now(),
    frontdoor_ad_data_normalized_version = sqlc.arg(frontdoor_ad_data_normalized_version)
WHERE frontdoor_ad_external_id = sqlc.arg(frontdoor_ad_external_id)
  AND frontdoor_ad_data_hash = sqlc.arg(frontdoor_ad_data_hash);

-- name: MarkFrontdoorAdProcessed :exec
UPDATE public.frontdoor_ads
SET frontdoor_ad_processed_at = now(), frontdoor_ad_updated_at = now()
WHERE frontdoor_ad_id = $1;

-- name: MarkFrontdoorAdNotFoundByExternalID :exec
UPDATE public.frontdoor_ads
SET frontdoor_ad_page_not_found = true,
    frontdoor_ad_processed_at = NOW(),
    frontdoor_ad_updated_at = NOW()
WHERE frontdoor_ad_external_id = $1;

-- name: MarkFrontdoorAdNotFound :exec
UPDATE public.frontdoor_ads
SET frontdoor_ad_page_not_found = true, frontdoor_ad_updated_at = now()
WHERE frontdoor_ad_id = $1;
