-- name: GetShortcutBuildingByID :one
SELECT * FROM public.shortcut_buildings
WHERE shortcut_buildings_id = $1;

-- name: GetShortcutBuildingByExternalID :one
SELECT * FROM public.shortcut_buildings
WHERE shortcut_buildings_external_id = $1;

-- name: ListShortcutBuildings :many
SELECT * FROM public.shortcut_buildings
ORDER BY shortcut_buildings_created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedShortcutBuildings :many
SELECT * FROM public.shortcut_buildings
WHERE shortcut_buildings_processed_at IS NULL AND shortcut_buildings_page_not_found = false
ORDER BY shortcut_buildings_created_at DESC
LIMIT $1;

-- name: UpsertShortcutBuildingFromSitemap :one
INSERT INTO public.shortcut_buildings (
    shortcut_buildings_external_id,
    shortcut_buildings_url
) VALUES (
    $1, $2
)
ON CONFLICT (shortcut_buildings_external_id) DO UPDATE SET
    shortcut_buildings_url = EXCLUDED.shortcut_buildings_url,
    shortcut_buildings_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: UpsertShortcutBuilding :one
INSERT INTO public.shortcut_buildings (
    shortcut_buildings_external_id,
    shortcut_buildings_building_id,
    shortcut_buildings_building_type,
    shortcut_buildings_building_subtype,
    shortcut_buildings_construction_year,
    shortcut_buildings_floor_count,
    shortcut_buildings_apartment_count,
    shortcut_buildings_heating_system,
    shortcut_buildings_building_material,
    shortcut_buildings_plot_type,
    shortcut_buildings_wall_structure,
    shortcut_buildings_heat_source,
    shortcut_buildings_has_elevator,
    shortcut_buildings_has_sauna,
    shortcut_buildings_latitude,
    shortcut_buildings_longitude,
    shortcut_buildings_additional_addresses,
    shortcut_buildings_url,
    shortcut_buildings_address,
    shortcut_buildings_frame_construction_method,
    shortcut_buildings_housing_company
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21
)
ON CONFLICT (shortcut_buildings_external_id) DO UPDATE SET
    shortcut_buildings_building_id = EXCLUDED.shortcut_buildings_building_id,
    shortcut_buildings_building_type = EXCLUDED.shortcut_buildings_building_type,
    shortcut_buildings_building_subtype = EXCLUDED.shortcut_buildings_building_subtype,
    shortcut_buildings_construction_year = EXCLUDED.shortcut_buildings_construction_year,
    shortcut_buildings_floor_count = EXCLUDED.shortcut_buildings_floor_count,
    shortcut_buildings_apartment_count = EXCLUDED.shortcut_buildings_apartment_count,
    shortcut_buildings_heating_system = EXCLUDED.shortcut_buildings_heating_system,
    shortcut_buildings_building_material = EXCLUDED.shortcut_buildings_building_material,
    shortcut_buildings_plot_type = EXCLUDED.shortcut_buildings_plot_type,
    shortcut_buildings_wall_structure = EXCLUDED.shortcut_buildings_wall_structure,
    shortcut_buildings_heat_source = EXCLUDED.shortcut_buildings_heat_source,
    shortcut_buildings_has_elevator = EXCLUDED.shortcut_buildings_has_elevator,
    shortcut_buildings_has_sauna = EXCLUDED.shortcut_buildings_has_sauna,
    shortcut_buildings_latitude = EXCLUDED.shortcut_buildings_latitude,
    shortcut_buildings_longitude = EXCLUDED.shortcut_buildings_longitude,
    shortcut_buildings_additional_addresses = EXCLUDED.shortcut_buildings_additional_addresses,
    shortcut_buildings_url = EXCLUDED.shortcut_buildings_url,
    shortcut_buildings_address = EXCLUDED.shortcut_buildings_address,
    shortcut_buildings_frame_construction_method = EXCLUDED.shortcut_buildings_frame_construction_method,
    shortcut_buildings_housing_company = EXCLUDED.shortcut_buildings_housing_company,
    shortcut_buildings_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: MarkShortcutBuildingProcessed :exec
UPDATE public.shortcut_buildings
SET shortcut_buildings_processed_at = CURRENT_TIMESTAMP, shortcut_buildings_updated_at = CURRENT_TIMESTAMP
WHERE shortcut_buildings_id = $1;

-- name: MarkShortcutBuildingPageNotFound :exec
UPDATE public.shortcut_buildings
SET shortcut_buildings_page_not_found = true, shortcut_buildings_updated_at = CURRENT_TIMESTAMP
WHERE shortcut_buildings_id = $1;

-- name: GetShortcutAdByID :one
SELECT * FROM public.shortcut_ads
WHERE shortcut_ads_id = $1;

-- name: ListShortcutAds :many
SELECT * FROM public.shortcut_ads
ORDER BY shortcut_ads_last_seen_at DESC
LIMIT $1 OFFSET $2;

-- name: UpsertShortcutAd :one
INSERT INTO public.shortcut_ads (
    shortcut_ads_id,
    shortcut_ads_url,
    shortcut_ads_type,
    shortcut_ads_data,
    shortcut_ads_building_id
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (shortcut_ads_id) DO UPDATE SET
    shortcut_ads_url = EXCLUDED.shortcut_ads_url,
    shortcut_ads_type = EXCLUDED.shortcut_ads_type,
    shortcut_ads_data = EXCLUDED.shortcut_ads_data,
    shortcut_ads_building_id = EXCLUDED.shortcut_ads_building_id,
    shortcut_ads_last_seen_at = now(),
    shortcut_ads_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetShortcutBuildingListingsByBuildingID :many
SELECT * FROM public.shortcut_building_listings
WHERE shortcut_building_listings_building_id = $1
ORDER BY shortcut_building_listings_created_at DESC;

-- name: UpsertShortcutBuildingListing :one
INSERT INTO public.shortcut_building_listings (
    shortcut_building_listings_building_id,
    shortcut_building_listings_layout,
    shortcut_building_listings_size,
    shortcut_building_listings_price,
    shortcut_building_listings_price_per_sqm,
    shortcut_building_listings_marketing_time,
    shortcut_building_listings_idx
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
ON CONFLICT (shortcut_building_listings_building_id, shortcut_building_listings_layout, shortcut_building_listings_size, shortcut_building_listings_price, shortcut_building_listings_price_per_sqm, shortcut_building_listings_deleted_at, shortcut_building_listings_marketing_time, shortcut_building_listings_idx) DO UPDATE SET
    shortcut_building_listings_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetShortcutBuildingRentalsByBuildingID :many
SELECT * FROM public.shortcut_building_rentals
WHERE shortcut_building_rentals_building_id = $1
ORDER BY shortcut_building_rentals_created_at DESC;

-- name: UpsertShortcutBuildingRental :one
INSERT INTO public.shortcut_building_rentals (
    shortcut_building_rentals_building_id,
    shortcut_building_rentals_layout,
    shortcut_building_rentals_size,
    shortcut_building_rentals_price,
    shortcut_building_rentals_marketing_time,
    shortcut_building_rentals_idx
) VALUES (
    $1, $2, $3, $4, $5, $6
)
ON CONFLICT (shortcut_building_rentals_building_id, shortcut_building_rentals_layout, shortcut_building_rentals_size, shortcut_building_rentals_price, shortcut_building_rentals_deleted_at, shortcut_building_rentals_marketing_time, shortcut_building_rentals_idx) DO UPDATE SET
    shortcut_building_rentals_updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: GetValidShortcutToken :one
SELECT * FROM public.shortcut_tokens
ORDER BY shortcut_tokens_created_at DESC
LIMIT 1;

-- name: GetAllValidShortcutTokens :many
SELECT * FROM public.shortcut_tokens
ORDER BY shortcut_tokens_created_at DESC;

-- name: InsertShortcutToken :one
INSERT INTO public.shortcut_tokens (
    shortcut_tokens_cuid,
    shortcut_tokens_token,
    shortcut_tokens_loaded,
    shortcut_tokens_expires_at
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (shortcut_tokens_cuid) DO UPDATE SET
    shortcut_tokens_token = EXCLUDED.shortcut_tokens_token,
    shortcut_tokens_loaded = EXCLUDED.shortcut_tokens_loaded,
    shortcut_tokens_expires_at = EXCLUDED.shortcut_tokens_expires_at,
    shortcut_tokens_updated_at = NOW()
RETURNING *;

-- name: DeleteShortcutToken :exec
DELETE FROM public.shortcut_tokens
WHERE shortcut_tokens_cuid = $1;
