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
    shortcut_building_id,
    shortcut_ad_last_seen_at
) VALUES (
    $1, $2, $3, $4, $5, now()
)
ON CONFLICT (shortcut_ad_id) DO UPDATE SET
    shortcut_ad_url = EXCLUDED.shortcut_ad_url,
    shortcut_ad_type = EXCLUDED.shortcut_ad_type,
    shortcut_ad_data = EXCLUDED.shortcut_ad_data,
    shortcut_building_id = EXCLUDED.shortcut_building_id,
    shortcut_ad_last_seen_at = now(),
    shortcut_ad_updated_at = CURRENT_TIMESTAMP
RETURNING *;
