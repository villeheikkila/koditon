-- name: GetShortcutBuildingListingsByBuildingID :many
SELECT shortcut_building_listing_id, shortcut_building_id, shortcut_building_listing_layout, shortcut_building_listing_size, shortcut_building_listing_price, shortcut_building_listing_price_per_sqm, shortcut_building_listing_deleted_at, shortcut_building_listing_created_at, shortcut_building_listing_updated_at, shortcut_building_listing_marketing_time, shortcut_building_listing_idx FROM origin.shortcut_building_listings
WHERE shortcut_building_id = $1
ORDER BY shortcut_building_listing_created_at DESC;

-- name: UpsertShortcutBuildingListing :one
INSERT INTO origin.shortcut_building_listings (
    shortcut_building_id,
    shortcut_building_listing_layout,
    shortcut_building_listing_size,
    shortcut_building_listing_price,
    shortcut_building_listing_price_per_sqm,
    shortcut_building_listing_marketing_time,
    shortcut_building_listing_idx
) VALUES (
    @shortcut_building_id,
    @shortcut_building_listing_layout,
    @shortcut_building_listing_size,
    @shortcut_building_listing_price,
    @shortcut_building_listing_price_per_sqm,
    @shortcut_building_listing_marketing_time,
    @shortcut_building_listing_idx
)
ON CONFLICT (shortcut_building_id, shortcut_building_listing_layout, shortcut_building_listing_size, shortcut_building_listing_price, shortcut_building_listing_price_per_sqm, shortcut_building_listing_deleted_at, shortcut_building_listing_marketing_time, shortcut_building_listing_idx) DO UPDATE SET
    shortcut_building_listing_updated_at = CURRENT_TIMESTAMP
RETURNING shortcut_building_listing_id, shortcut_building_id, shortcut_building_listing_layout, shortcut_building_listing_size, shortcut_building_listing_price, shortcut_building_listing_price_per_sqm, shortcut_building_listing_deleted_at, shortcut_building_listing_created_at, shortcut_building_listing_updated_at, shortcut_building_listing_marketing_time, shortcut_building_listing_idx;

-- name: GetShortcutBuildingRentalsByBuildingID :many
SELECT shortcut_building_rental_id, shortcut_building_id, shortcut_building_rental_layout, shortcut_building_rental_size, shortcut_building_rental_price, shortcut_building_rental_deleted_at, shortcut_building_rental_created_at, shortcut_building_rental_updated_at, shortcut_building_rental_marketing_time, shortcut_building_rental_idx FROM origin.shortcut_building_rentals
WHERE shortcut_building_id = $1
ORDER BY shortcut_building_rental_created_at DESC;

-- name: UpsertShortcutBuildingRental :one
INSERT INTO origin.shortcut_building_rentals (
    shortcut_building_id,
    shortcut_building_rental_layout,
    shortcut_building_rental_size,
    shortcut_building_rental_price,
    shortcut_building_rental_marketing_time,
    shortcut_building_rental_idx
) VALUES (
    @shortcut_building_id,
    @shortcut_building_rental_layout,
    @shortcut_building_rental_size,
    @shortcut_building_rental_price,
    @shortcut_building_rental_marketing_time,
    @shortcut_building_rental_idx
)
ON CONFLICT (shortcut_building_id, shortcut_building_rental_layout, shortcut_building_rental_size, shortcut_building_rental_price, shortcut_building_rental_deleted_at, shortcut_building_rental_marketing_time, shortcut_building_rental_idx) DO UPDATE SET
    shortcut_building_rental_updated_at = CURRENT_TIMESTAMP
RETURNING shortcut_building_rental_id, shortcut_building_id, shortcut_building_rental_layout, shortcut_building_rental_size, shortcut_building_rental_price, shortcut_building_rental_deleted_at, shortcut_building_rental_created_at, shortcut_building_rental_updated_at, shortcut_building_rental_marketing_time, shortcut_building_rental_idx;
