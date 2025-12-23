-- name: GetFrontdoorAdByExternalID :one
SELECT * FROM public.frontdoor_ads
WHERE frontdoor_ads_external_id = $1;

-- name: ListFrontdoorAds :many
SELECT * FROM public.frontdoor_ads
ORDER BY frontdoor_ads_last_seen_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedFrontdoorAds :many
SELECT * FROM public.frontdoor_ads
WHERE frontdoor_ads_processed_at IS NULL AND frontdoor_ads_page_not_found = false
ORDER BY frontdoor_ads_first_seen_at ASC
LIMIT $1;

-- name: UpsertFrontdoorAds :exec
INSERT INTO public.frontdoor_ads (frontdoor_ads_external_id)
SELECT unnest($1::text[])
ON CONFLICT (frontdoor_ads_external_id) DO UPDATE SET
    frontdoor_ads_last_seen_at = NOW(),
    frontdoor_ads_updated_at = NOW();

-- name: UpsertFrontdoorAdFromSitemap :one
INSERT INTO public.frontdoor_ads (
    frontdoor_ads_external_id,
    frontdoor_ads_url,
    frontdoor_ads_first_seen_at,
    frontdoor_ads_last_seen_at,
    frontdoor_ads_updated_at
) VALUES ($1, $2, now(), now(), now())
ON CONFLICT (frontdoor_ads_external_id) DO UPDATE
SET frontdoor_ads_last_seen_at = now(),
    frontdoor_ads_updated_at = now(),
    frontdoor_ads_url = COALESCE(EXCLUDED.frontdoor_ads_url, frontdoor_ads.frontdoor_ads_url)
RETURNING *;

-- name: UpdateFrontdoorAdData :exec
UPDATE public.frontdoor_ads
SET frontdoor_ads_data = $2::jsonb,
    frontdoor_ads_processed_at = NOW(),
    frontdoor_ads_updated_at = NOW(),
    frontdoor_ads_page_not_found = false
WHERE frontdoor_ads_external_id = $1;

-- name: MarkFrontdoorAdProcessed :exec
UPDATE public.frontdoor_ads
SET frontdoor_ads_processed_at = now(), frontdoor_ads_updated_at = now()
WHERE frontdoor_ads_id = $1;

-- name: MarkFrontdoorAdNotFoundByExternalID :exec
UPDATE public.frontdoor_ads
SET frontdoor_ads_page_not_found = true,
    frontdoor_ads_processed_at = NOW(),
    frontdoor_ads_updated_at = NOW()
WHERE frontdoor_ads_external_id = $1;

-- name: MarkFrontdoorAdNotFound :exec
UPDATE public.frontdoor_ads
SET frontdoor_ads_page_not_found = true, frontdoor_ads_updated_at = now()
WHERE frontdoor_ads_id = $1;

-- name: GetFrontdoorBuildingByID :one
SELECT * FROM public.frontdoor_buildings
WHERE frontdoor_buildings_id = $1;

-- name: GetFrontdoorBuildingByHousingCompanyID :one
SELECT * FROM public.frontdoor_buildings
WHERE frontdoor_buildings_housing_company_id = $1;

-- name: ListFrontdoorBuildings :many
SELECT * FROM public.frontdoor_buildings
ORDER BY frontdoor_buildings_last_seen_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedFrontdoorBuildings :many
SELECT * FROM public.frontdoor_buildings
WHERE frontdoor_buildings_processed_at IS NULL
ORDER BY frontdoor_buildings_first_seen_at ASC
LIMIT $1;

-- name: UpsertFrontdoorBuildings :exec
INSERT INTO public.frontdoor_buildings (frontdoor_buildings_housing_company_id)
SELECT unnest($1::int8[])
ON CONFLICT (frontdoor_buildings_housing_company_id) DO UPDATE SET
    frontdoor_buildings_last_seen_at = NOW(),
    frontdoor_buildings_updated_at = NOW();

-- name: UpsertFrontdoorBuilding :one
INSERT INTO public.frontdoor_buildings (
    frontdoor_buildings_url,
    frontdoor_buildings_first_seen_at,
    frontdoor_buildings_last_seen_at,
    frontdoor_buildings_updated_at,
    frontdoor_buildings_housing_company_id,
    frontdoor_buildings_housing_company_friendly_id
) VALUES ($1, now(), now(), now(), $2, $3)
ON CONFLICT (frontdoor_buildings_housing_company_id) DO UPDATE
SET frontdoor_buildings_last_seen_at = now(),
    frontdoor_buildings_updated_at = now(),
    frontdoor_buildings_url = COALESCE(EXCLUDED.frontdoor_buildings_url, frontdoor_buildings.frontdoor_buildings_url),
    frontdoor_buildings_housing_company_friendly_id = COALESCE(EXCLUDED.frontdoor_buildings_housing_company_friendly_id, frontdoor_buildings.frontdoor_buildings_housing_company_friendly_id)
RETURNING *;

-- name: GetFrontdoorBuildingURLByHousingCompanyID :one
SELECT frontdoor_buildings_url FROM public.frontdoor_buildings
WHERE frontdoor_buildings_housing_company_id = $1;

-- name: UpdateFrontdoorBuildingDetails :one
UPDATE public.frontdoor_buildings
SET frontdoor_buildings_company_name = $2,
    frontdoor_buildings_business_id = $3,

    frontdoor_buildings_apartment_count = $4,
    frontdoor_buildings_floor_count = $5,
    frontdoor_buildings_construction_end_year = $6,
    frontdoor_buildings_build_year = $7,
    frontdoor_buildings_has_elevator = $8,
    frontdoor_buildings_has_sauna = $9,
    frontdoor_buildings_energy_certificate_code = $10,
    frontdoor_buildings_plot_holding_type = $11,
    frontdoor_buildings_outer_roof_material = $12,
    frontdoor_buildings_outer_roof_type = $13,
    frontdoor_buildings_heating = $14,
    frontdoor_buildings_heating_fuel = $15,
    frontdoor_buildings_street_address = $16,
    frontdoor_buildings_house_number = $17,
    frontdoor_buildings_postcode = $18,
    frontdoor_buildings_post_area = $19,
    frontdoor_buildings_municipality = $20,
    frontdoor_buildings_district = $21,
    frontdoor_buildings_latitude = $22,
    frontdoor_buildings_longitude = $23,
    frontdoor_buildings_description = $24,
    frontdoor_buildings_car_storage_description = $25,
    frontdoor_buildings_other_info = $26,
    frontdoor_buildings_data = $27,
    frontdoor_buildings_processed_at = now(),
    frontdoor_buildings_updated_at = now()
WHERE frontdoor_buildings_id = $1
RETURNING *;

-- name: UpdateFrontdoorBuildingDetailsByHousingCompanyID :exec
UPDATE public.frontdoor_buildings
SET frontdoor_buildings_company_name = $2,
    frontdoor_buildings_business_id = $3,
    frontdoor_buildings_apartment_count = $4,
    frontdoor_buildings_floor_count = $5,
    frontdoor_buildings_construction_end_year = $6,
    frontdoor_buildings_build_year = $7,
    frontdoor_buildings_has_elevator = $8,
    frontdoor_buildings_has_sauna = $9,
    frontdoor_buildings_energy_certificate_code = $10,
    frontdoor_buildings_plot_holding_type = $11,
    frontdoor_buildings_outer_roof_material = $12,
    frontdoor_buildings_outer_roof_type = $13,
    frontdoor_buildings_heating = $14,
    frontdoor_buildings_heating_fuel = $15,
    frontdoor_buildings_street_address = $16,
    frontdoor_buildings_house_number = $17,
    frontdoor_buildings_postcode = $18,
    frontdoor_buildings_post_area = $19,
    frontdoor_buildings_municipality = $20,
    frontdoor_buildings_district = $21,
    frontdoor_buildings_latitude = $22,
    frontdoor_buildings_longitude = $23,
    frontdoor_buildings_elevator_renovated = $24,
    frontdoor_buildings_elevator_renovated_year = $25,
    frontdoor_buildings_facade_renovated = $26,
    frontdoor_buildings_facade_renovated_year = $27,
    frontdoor_buildings_window_renovated = $28,
    frontdoor_buildings_window_renovated_year = $29,
    frontdoor_buildings_roof_renovated = $30,
    frontdoor_buildings_roof_renovated_year = $31,
    frontdoor_buildings_pipe_renovated = $32,
    frontdoor_buildings_pipe_renovated_year = $33,
    frontdoor_buildings_balcony_renovated = $34,
    frontdoor_buildings_balcony_renovated_year = $35,
    frontdoor_buildings_electricity_renovated = $36,
    frontdoor_buildings_electricity_renovated_year = $37,
    frontdoor_buildings_contact_phone = $38,
    frontdoor_buildings_contact_office_name = $39,
    frontdoor_buildings_contact_office_id = $40,
    frontdoor_buildings_description = $41,
    frontdoor_buildings_car_storage_description = $42,
    frontdoor_buildings_other_info = $43,
    frontdoor_buildings_data = $44::jsonb,
    frontdoor_buildings_processed_at = NOW(),
    frontdoor_buildings_updated_at = NOW()
WHERE frontdoor_buildings_housing_company_id = $1;

-- name: MarkFrontdoorBuildingProcessed :exec
UPDATE public.frontdoor_buildings
SET frontdoor_buildings_processed_at = now(), frontdoor_buildings_updated_at = now()
WHERE frontdoor_buildings_id = $1;

-- name: GetFrontdoorBuildingAnnouncementByID :one
SELECT * FROM public.frontdoor_building_announcements
WHERE frontdoor_building_announcements_id = $1;

-- name: ListFrontdoorBuildingAnnouncements :many
SELECT * FROM public.frontdoor_building_announcements
WHERE frontdoor_building_announcements_building_id = $1
ORDER BY frontdoor_building_announcements_last_seen_at DESC;

-- name: GetFrontdoorBuildingIDByHousingCompanyID :one
SELECT frontdoor_buildings_id FROM public.frontdoor_buildings
WHERE frontdoor_buildings_housing_company_id = $1;

-- name: UpsertFrontdoorBuildingAnnouncement :one
INSERT INTO public.frontdoor_building_announcements (
    frontdoor_building_announcements_external_id,
    frontdoor_building_announcements_friendly_id,
    frontdoor_building_announcements_unpublishing_time,
    frontdoor_building_announcements_address_line1,
    frontdoor_building_announcements_address_line2,
    frontdoor_building_announcements_location,
    frontdoor_building_announcements_search_price,
    frontdoor_building_announcements_notify_price_changed,
    frontdoor_building_announcements_property_type,
    frontdoor_building_announcements_property_subtype,
    frontdoor_building_announcements_construction_finished_year,
    frontdoor_building_announcements_main_image_uri,
    frontdoor_building_announcements_has_open_bidding,
    frontdoor_building_announcements_room_structure,
    frontdoor_building_announcements_area,
    frontdoor_building_announcements_total_area,
    frontdoor_building_announcements_price_per_square,
    frontdoor_building_announcements_days_on_market,
    frontdoor_building_announcements_new_building,
    frontdoor_building_announcements_main_image_hidden,
    frontdoor_building_announcements_is_company_announcement,
    frontdoor_building_announcements_show_bidding_indicators,
    frontdoor_building_announcements_published,
    frontdoor_building_announcements_rent_period,
    frontdoor_building_announcements_rental_unique_no,
    frontdoor_building_announcements_building_id,
    frontdoor_building_announcements_first_seen_at,
    frontdoor_building_announcements_last_seen_at,
    frontdoor_building_announcements_unpublishing_time_date
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16, $17, $18, $19, $20,
    $21, $22, $23, $24, $25, $26, now(), now(), $27
)
ON CONFLICT (frontdoor_building_announcements_external_id, frontdoor_building_announcements_unpublishing_time, frontdoor_building_announcements_search_price) DO UPDATE
SET frontdoor_building_announcements_last_seen_at = now(),
    frontdoor_building_announcements_friendly_id = COALESCE(EXCLUDED.frontdoor_building_announcements_friendly_id, frontdoor_building_announcements.frontdoor_building_announcements_friendly_id),
    frontdoor_building_announcements_address_line1 = COALESCE(EXCLUDED.frontdoor_building_announcements_address_line1, frontdoor_building_announcements.frontdoor_building_announcements_address_line1),
    frontdoor_building_announcements_address_line2 = COALESCE(EXCLUDED.frontdoor_building_announcements_address_line2, frontdoor_building_announcements.frontdoor_building_announcements_address_line2),
    frontdoor_building_announcements_location = COALESCE(EXCLUDED.frontdoor_building_announcements_location, frontdoor_building_announcements.frontdoor_building_announcements_location),
    frontdoor_building_announcements_notify_price_changed = COALESCE(EXCLUDED.frontdoor_building_announcements_notify_price_changed, frontdoor_building_announcements.frontdoor_building_announcements_notify_price_changed),
    frontdoor_building_announcements_property_type = COALESCE(EXCLUDED.frontdoor_building_announcements_property_type, frontdoor_building_announcements.frontdoor_building_announcements_property_type),
    frontdoor_building_announcements_property_subtype = COALESCE(EXCLUDED.frontdoor_building_announcements_property_subtype, frontdoor_building_announcements.frontdoor_building_announcements_property_subtype),
    frontdoor_building_announcements_construction_finished_year = COALESCE(EXCLUDED.frontdoor_building_announcements_construction_finished_year, frontdoor_building_announcements.frontdoor_building_announcements_construction_finished_year),
    frontdoor_building_announcements_main_image_uri = COALESCE(EXCLUDED.frontdoor_building_announcements_main_image_uri, frontdoor_building_announcements.frontdoor_building_announcements_main_image_uri),
    frontdoor_building_announcements_has_open_bidding = COALESCE(EXCLUDED.frontdoor_building_announcements_has_open_bidding, frontdoor_building_announcements.frontdoor_building_announcements_has_open_bidding),
    frontdoor_building_announcements_room_structure = COALESCE(EXCLUDED.frontdoor_building_announcements_room_structure, frontdoor_building_announcements.frontdoor_building_announcements_room_structure),
    frontdoor_building_announcements_area = COALESCE(EXCLUDED.frontdoor_building_announcements_area, frontdoor_building_announcements.frontdoor_building_announcements_area),
    frontdoor_building_announcements_total_area = COALESCE(EXCLUDED.frontdoor_building_announcements_total_area, frontdoor_building_announcements.frontdoor_building_announcements_total_area),
    frontdoor_building_announcements_price_per_square = COALESCE(EXCLUDED.frontdoor_building_announcements_price_per_square, frontdoor_building_announcements.frontdoor_building_announcements_price_per_square),
    frontdoor_building_announcements_days_on_market = COALESCE(EXCLUDED.frontdoor_building_announcements_days_on_market, frontdoor_building_announcements.frontdoor_building_announcements_days_on_market),
    frontdoor_building_announcements_new_building = COALESCE(EXCLUDED.frontdoor_building_announcements_new_building, frontdoor_building_announcements.frontdoor_building_announcements_new_building),
    frontdoor_building_announcements_main_image_hidden = COALESCE(EXCLUDED.frontdoor_building_announcements_main_image_hidden, frontdoor_building_announcements.frontdoor_building_announcements_main_image_hidden),
    frontdoor_building_announcements_is_company_announcement = COALESCE(EXCLUDED.frontdoor_building_announcements_is_company_announcement, frontdoor_building_announcements.frontdoor_building_announcements_is_company_announcement),
    frontdoor_building_announcements_show_bidding_indicators = COALESCE(EXCLUDED.frontdoor_building_announcements_show_bidding_indicators, frontdoor_building_announcements.frontdoor_building_announcements_show_bidding_indicators),
    frontdoor_building_announcements_published = COALESCE(EXCLUDED.frontdoor_building_announcements_published, frontdoor_building_announcements.frontdoor_building_announcements_published),
    frontdoor_building_announcements_rent_period = COALESCE(EXCLUDED.frontdoor_building_announcements_rent_period, frontdoor_building_announcements.frontdoor_building_announcements_rent_period),
    frontdoor_building_announcements_rental_unique_no = COALESCE(EXCLUDED.frontdoor_building_announcements_rental_unique_no, frontdoor_building_announcements.frontdoor_building_announcements_rental_unique_no),
    frontdoor_building_announcements_unpublishing_time_date = COALESCE(EXCLUDED.frontdoor_building_announcements_unpublishing_time_date, frontdoor_building_announcements.frontdoor_building_announcements_unpublishing_time_date)
RETURNING *;
