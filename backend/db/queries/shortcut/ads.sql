-- name: GetShortcutAdByID :one
SELECT * FROM public.shortcut_ads
WHERE shortcut_ad_id = $1;

-- name: ListShortcutAds :many
SELECT * FROM public.shortcut_ads
ORDER BY shortcut_ad_last_seen_at DESC
LIMIT $1 OFFSET $2;

-- name: UpsertShortcutAdFromSitemap :one
INSERT INTO public.shortcut_ads (
    shortcut_ad_id,
    shortcut_ad_url,
    shortcut_ad_type,
    shortcut_ad_last_seen_at
) VALUES (
    $1, $2, $3, now()
)
ON CONFLICT (shortcut_ad_id) DO UPDATE SET
    shortcut_ad_url = EXCLUDED.shortcut_ad_url,
    shortcut_ad_type = EXCLUDED.shortcut_ad_type,
    shortcut_ad_last_seen_at = now()
RETURNING *;

-- name: BatchUpsertShortcutAdsFromSitemap :many
INSERT INTO public.shortcut_ads (
    shortcut_ad_id,
    shortcut_ad_url,
    shortcut_ad_type,
    shortcut_ad_last_seen_at
)
SELECT UNNEST($1::bigint[]), UNNEST($2::text[]), UNNEST($3::text[]), now()
ON CONFLICT (shortcut_ad_id) DO UPDATE SET
    shortcut_ad_url = EXCLUDED.shortcut_ad_url,
    shortcut_ad_type = EXCLUDED.shortcut_ad_type,
    shortcut_ad_last_seen_at = now()
RETURNING *;

-- name: UpsertShortcutAd :one
INSERT INTO public.shortcut_ads (
    shortcut_ad_id,
    shortcut_ad_url,
    shortcut_ad_type,
    shortcut_ad_data,
    shortcut_ad_data_hash,
    shortcut_ad_data_hash_algorithm,
    shortcut_ad_data_changed_at,
    shortcut_ad_data_schema_version,
    shortcut_building_id,
    shortcut_ad_last_seen_at
) VALUES (
    $1, $2, $3, sqlc.arg(shortcut_ad_data)::jsonb, sqlc.arg(shortcut_ad_data_hash), sqlc.arg(shortcut_ad_data_hash_algorithm), now(), sqlc.arg(shortcut_ad_data_schema_version), sqlc.arg(shortcut_building_id), now()
)
ON CONFLICT (shortcut_ad_id) DO UPDATE SET
    shortcut_ad_url = EXCLUDED.shortcut_ad_url,
    shortcut_ad_type = EXCLUDED.shortcut_ad_type,
    shortcut_ad_data = EXCLUDED.shortcut_ad_data,
    shortcut_ad_data_hash = EXCLUDED.shortcut_ad_data_hash,
    shortcut_ad_data_hash_algorithm = EXCLUDED.shortcut_ad_data_hash_algorithm,
    shortcut_ad_data_changed_at = CASE WHEN shortcut_ads.shortcut_ad_data_hash IS DISTINCT FROM EXCLUDED.shortcut_ad_data_hash THEN now() ELSE shortcut_ads.shortcut_ad_data_changed_at END,
    shortcut_ad_data_normalized_at = CASE WHEN shortcut_ads.shortcut_ad_data_hash IS DISTINCT FROM EXCLUDED.shortcut_ad_data_hash THEN NULL ELSE shortcut_ads.shortcut_ad_data_normalized_at END,
    shortcut_ad_data_schema_version = EXCLUDED.shortcut_ad_data_schema_version,
    shortcut_building_id = EXCLUDED.shortcut_building_id,
    shortcut_ad_last_seen_at = now(),
    shortcut_ad_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: ListShortcutAdsMissingDataHash :many
SELECT shortcut_ad_id, shortcut_ad_data
FROM public.shortcut_ads
WHERE shortcut_ad_data IS NOT NULL
  AND shortcut_ad_data_hash IS NULL
ORDER BY shortcut_ad_updated_at ASC NULLS FIRST, shortcut_ad_first_seen_at ASC
LIMIT $1;

-- name: BackfillShortcutAdDataHash :exec
UPDATE public.shortcut_ads
SET shortcut_ad_data = sqlc.arg(shortcut_ad_data)::jsonb,
    shortcut_ad_data_hash = sqlc.arg(shortcut_ad_data_hash),
    shortcut_ad_data_hash_algorithm = sqlc.arg(shortcut_ad_data_hash_algorithm),
    shortcut_ad_data_changed_at = COALESCE(shortcut_ad_data_changed_at, shortcut_ad_updated_at, shortcut_ad_last_seen_at, now())
WHERE shortcut_ad_id = sqlc.arg(shortcut_ad_id);

-- name: MarkShortcutAdDataNormalized :exec
UPDATE public.shortcut_ads
SET shortcut_ad_data_normalized_at = now()
WHERE shortcut_ad_id = sqlc.arg(shortcut_ad_id)
  AND shortcut_ad_data_hash = sqlc.arg(shortcut_ad_data_hash);
