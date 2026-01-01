-- name: GetShortcutBuildingByID :one
SELECT * FROM public.shortcut_buildings
WHERE shortcut_building_id = $1;

-- name: GetShortcutBuildingByExternalID :one
SELECT * FROM public.shortcut_buildings
WHERE shortcut_building_external_id = $1;

-- name: ListShortcutBuildings :many
SELECT * FROM public.shortcut_buildings
ORDER BY shortcut_building_created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedShortcutBuildings :many
SELECT * FROM public.shortcut_buildings
WHERE shortcut_building_processed_at IS NULL AND shortcut_building_page_not_found = false
ORDER BY shortcut_building_created_at DESC
LIMIT $1;

-- name: UpsertShortcutBuildingFromSitemap :one
INSERT INTO public.shortcut_buildings (
    shortcut_building_external_id,
    shortcut_building_url
) VALUES (
    $1, $2
)
ON CONFLICT (shortcut_building_external_id) DO UPDATE SET
    shortcut_building_url = EXCLUDED.shortcut_building_url
WHERE public.shortcut_buildings.shortcut_building_url IS DISTINCT FROM EXCLUDED.shortcut_building_url
RETURNING *;

-- name: BatchUpsertShortcutBuildingsFromSitemap :many
INSERT INTO public.shortcut_buildings (
    shortcut_building_external_id,
    shortcut_building_url
)
SELECT UNNEST($1::bigint[]), UNNEST($2::text[])
ON CONFLICT (shortcut_building_external_id) DO UPDATE SET
    shortcut_building_url = EXCLUDED.shortcut_building_url
WHERE public.shortcut_buildings.shortcut_building_url IS DISTINCT FROM EXCLUDED.shortcut_building_url
RETURNING *;

-- name: UpsertShortcutBuilding :one
INSERT INTO public.shortcut_buildings (
    shortcut_building_external_id,
    shortcut_building_building_id,
    shortcut_building_building_type,
    shortcut_building_building_subtype,
    shortcut_building_construction_year,
    shortcut_building_floor_count,
    shortcut_building_apartment_count,
    shortcut_building_heating_system,
    shortcut_building_building_material,
    shortcut_building_plot_type,
    shortcut_building_wall_structure,
    shortcut_building_heat_source,
    shortcut_building_has_elevator,
    shortcut_building_has_sauna,
    shortcut_building_latitude,
    shortcut_building_longitude,
    shortcut_building_additional_addresses,
    shortcut_building_url,
    shortcut_building_address,
    shortcut_building_frame_construction_method,
    shortcut_building_housing_company
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (shortcut_building_external_id) DO UPDATE SET
    shortcut_building_building_id = EXCLUDED.shortcut_building_building_id,
    shortcut_building_building_type = EXCLUDED.shortcut_building_building_type,
    shortcut_building_building_subtype = EXCLUDED.shortcut_building_building_subtype,
    shortcut_building_construction_year = EXCLUDED.shortcut_building_construction_year,
    shortcut_building_floor_count = EXCLUDED.shortcut_building_floor_count,
    shortcut_building_apartment_count = EXCLUDED.shortcut_building_apartment_count,
    shortcut_building_heating_system = EXCLUDED.shortcut_building_heating_system,
    shortcut_building_building_material = EXCLUDED.shortcut_building_building_material,
    shortcut_building_plot_type = EXCLUDED.shortcut_building_plot_type,
    shortcut_building_wall_structure = EXCLUDED.shortcut_building_wall_structure,
    shortcut_building_heat_source = EXCLUDED.shortcut_building_heat_source,
    shortcut_building_has_elevator = EXCLUDED.shortcut_building_has_elevator,
    shortcut_building_has_sauna = EXCLUDED.shortcut_building_has_sauna,
    shortcut_building_latitude = EXCLUDED.shortcut_building_latitude,
    shortcut_building_longitude = EXCLUDED.shortcut_building_longitude,
    shortcut_building_additional_addresses = EXCLUDED.shortcut_building_additional_addresses,
    shortcut_building_url = EXCLUDED.shortcut_building_url,
    shortcut_building_address = EXCLUDED.shortcut_building_address,
    shortcut_building_frame_construction_method = EXCLUDED.shortcut_building_frame_construction_method,
    shortcut_building_housing_company = EXCLUDED.shortcut_building_housing_company,
    shortcut_building_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: MarkShortcutBuildingProcessed :exec
UPDATE public.shortcut_buildings
SET shortcut_building_processed_at = CURRENT_TIMESTAMP, shortcut_building_updated_at = CURRENT_TIMESTAMP
WHERE shortcut_building_id = $1;

-- name: MarkShortcutBuildingPageNotFound :exec
UPDATE public.shortcut_buildings
SET shortcut_building_page_not_found = true, shortcut_building_updated_at = CURRENT_TIMESTAMP
WHERE shortcut_building_id = $1;

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

-- name: GetShortcutBuildingListingsByBuildingID :many
SELECT * FROM public.shortcut_building_listings
WHERE shortcut_building_id = $1
ORDER BY shortcut_building_listing_created_at DESC;

-- name: UpsertShortcutBuildingListing :one
INSERT INTO public.shortcut_building_listings (
    shortcut_building_id,
    shortcut_building_listing_layout,
    shortcut_building_listing_size,
    shortcut_building_listing_price,
    shortcut_building_listing_price_per_sqm,
    shortcut_building_listing_marketing_time,
    shortcut_building_listing_idx
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (shortcut_building_id, shortcut_building_listing_layout, shortcut_building_listing_size, shortcut_building_listing_price, shortcut_building_listing_price_per_sqm, shortcut_building_listing_deleted_at, shortcut_building_listing_marketing_time, shortcut_building_listing_idx) DO UPDATE SET
    shortcut_building_listing_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetShortcutBuildingRentalsByBuildingID :many
SELECT * FROM public.shortcut_building_rentals
WHERE shortcut_building_id = $1
ORDER BY shortcut_building_rental_created_at DESC;

-- name: UpsertShortcutBuildingRental :one
INSERT INTO public.shortcut_building_rentals (
    shortcut_building_id,
    shortcut_building_rental_layout,
    shortcut_building_rental_size,
    shortcut_building_rental_price,
    shortcut_building_rental_marketing_time,
    shortcut_building_rental_idx
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (shortcut_building_id, shortcut_building_rental_layout, shortcut_building_rental_size, shortcut_building_rental_price, shortcut_building_rental_deleted_at, shortcut_building_rental_marketing_time, shortcut_building_rental_idx) DO UPDATE SET
    shortcut_building_rental_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetValidShortcutToken :one
SELECT * FROM public.shortcut_tokens
ORDER BY shortcut_token_created_at DESC
LIMIT 1;

-- name: GetAllValidShortcutTokens :many
SELECT * FROM public.shortcut_tokens
ORDER BY shortcut_token_created_at DESC;

-- name: InsertShortcutToken :one
INSERT INTO public.shortcut_tokens (
    shortcut_token_cuid,
    shortcut_token_token,
    shortcut_token_loaded,
    shortcut_token_expires_at
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (shortcut_token_cuid) DO UPDATE SET
    shortcut_token_token = EXCLUDED.shortcut_token_token,
    shortcut_token_loaded = EXCLUDED.shortcut_token_loaded,
    shortcut_token_expires_at = EXCLUDED.shortcut_token_expires_at,
    shortcut_token_updated_at = NOW()
RETURNING *;

-- name: DeleteShortcutToken :exec
DELETE FROM public.shortcut_tokens
WHERE shortcut_token_cuid = $1;
