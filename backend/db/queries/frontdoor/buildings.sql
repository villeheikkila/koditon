-- name: GetFrontdoorBuildingByID :one
SELECT * FROM public.frontdoor_buildings
WHERE frontdoor_building_id = $1;

-- name: GetFrontdoorBuildingByHousingCompanyID :one
SELECT * FROM public.frontdoor_buildings
WHERE frontdoor_building_housing_company_id = $1;

-- name: ListFrontdoorBuildings :many
SELECT * FROM public.frontdoor_buildings
ORDER BY frontdoor_building_last_seen_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedFrontdoorBuildings :many
SELECT * FROM public.frontdoor_buildings
WHERE frontdoor_building_processed_at IS NULL
ORDER BY frontdoor_building_first_seen_at ASC
LIMIT $1;

-- name: UpsertFrontdoorBuildings :exec
INSERT INTO public.frontdoor_buildings (frontdoor_building_housing_company_id)
SELECT unnest($1::int8[])
ON CONFLICT (frontdoor_building_housing_company_id) DO UPDATE SET
    frontdoor_building_last_seen_at = NOW(),
    frontdoor_building_updated_at = NOW();

-- name: UpsertFrontdoorBuilding :one
INSERT INTO public.frontdoor_buildings (
    frontdoor_building_url,
    frontdoor_building_first_seen_at,
    frontdoor_building_last_seen_at,
    frontdoor_building_updated_at,
    frontdoor_building_housing_company_id,
    frontdoor_building_housing_company_friendly_id
) VALUES ($1, now(), now(), now(), $2, $3)
ON CONFLICT (frontdoor_building_housing_company_id) DO UPDATE
SET frontdoor_building_last_seen_at = now(),
    frontdoor_building_url = COALESCE(EXCLUDED.frontdoor_building_url, frontdoor_buildings.frontdoor_building_url),
    frontdoor_building_housing_company_friendly_id = COALESCE(EXCLUDED.frontdoor_building_housing_company_friendly_id, frontdoor_buildings.frontdoor_building_housing_company_friendly_id)
RETURNING *;

-- name: BatchUpsertFrontdoorBuildingsFromSitemap :many
INSERT INTO public.frontdoor_buildings (
    frontdoor_building_url,
    frontdoor_building_first_seen_at,
    frontdoor_building_last_seen_at,
    frontdoor_building_updated_at
)
SELECT UNNEST($1::text[]), now(), now(), now()
ON CONFLICT (frontdoor_building_url) DO UPDATE
SET frontdoor_building_last_seen_at = now()
RETURNING *;

-- name: GetFrontdoorBuildingURLByHousingCompanyID :one
SELECT frontdoor_building_url FROM public.frontdoor_buildings
WHERE frontdoor_building_housing_company_id = $1;

-- name: UpdateFrontdoorBuildingDetails :one
UPDATE public.frontdoor_buildings
SET frontdoor_building_company_name = $2,
    frontdoor_building_business_id = $3,

    frontdoor_building_apartment_count = $4,
    frontdoor_building_floor_count = $5,
    frontdoor_building_construction_end_year = $6,
    frontdoor_building_build_year = $7,
    frontdoor_building_has_elevator = $8,
    frontdoor_building_has_sauna = $9,
    frontdoor_building_energy_certificate_code = $10,
    frontdoor_building_plot_holding_type = $11,
    frontdoor_building_outer_roof_material = $12,
    frontdoor_building_outer_roof_type = $13,
    frontdoor_building_heating = $14,
    frontdoor_building_heating_fuel = $15,
    frontdoor_building_street_address = $16,
    frontdoor_building_house_number = $17,
    frontdoor_building_postcode = $18,
    frontdoor_building_post_area = $19,
    frontdoor_building_municipality = $20,
    frontdoor_building_district = $21,
    frontdoor_building_latitude = $22,
    frontdoor_building_longitude = $23,
    frontdoor_building_description = $24,
    frontdoor_building_car_storage_description = $25,
    frontdoor_building_other_info = $26,
    frontdoor_building_data = $27,
    frontdoor_building_processed_at = now(),
    frontdoor_building_updated_at = now()
WHERE frontdoor_building_id = $1
RETURNING *;

-- name: UpdateFrontdoorBuildingDetailsByHousingCompanyID :exec
UPDATE public.frontdoor_buildings
SET frontdoor_building_company_name = $2,
    frontdoor_building_business_id = $3,
    frontdoor_building_apartment_count = $4,
    frontdoor_building_floor_count = $5,
    frontdoor_building_construction_end_year = $6,
    frontdoor_building_build_year = $7,
    frontdoor_building_has_elevator = $8,
    frontdoor_building_has_sauna = $9,
    frontdoor_building_energy_certificate_code = $10,
    frontdoor_building_plot_holding_type = $11,
    frontdoor_building_outer_roof_material = $12,
    frontdoor_building_outer_roof_type = $13,
    frontdoor_building_heating = $14,
    frontdoor_building_heating_fuel = $15,
    frontdoor_building_street_address = $16,
    frontdoor_building_house_number = $17,
    frontdoor_building_postcode = $18,
    frontdoor_building_post_area = $19,
    frontdoor_building_municipality = $20,
    frontdoor_building_district = $21,
    frontdoor_building_latitude = $22,
    frontdoor_building_longitude = $23,
    frontdoor_building_elevator_renovated = $24,
    frontdoor_building_elevator_renovated_year = $25,
    frontdoor_building_facade_renovated = $26,
    frontdoor_building_facade_renovated_year = $27,
    frontdoor_building_window_renovated = $28,
    frontdoor_building_window_renovated_year = $29,
    frontdoor_building_roof_renovated = $30,
    frontdoor_building_roof_renovated_year = $31,
    frontdoor_building_pipe_renovated = $32,
    frontdoor_building_pipe_renovated_year = $33,
    frontdoor_building_balcony_renovated = $34,
    frontdoor_building_balcony_renovated_year = $35,
    frontdoor_building_electricity_renovated = $36,
    frontdoor_building_electricity_renovated_year = $37,
    frontdoor_building_contact_phone = $38,
    frontdoor_building_contact_office_name = $39,
    frontdoor_building_contact_office_id = $40,
    frontdoor_building_description = $41,
    frontdoor_building_car_storage_description = $42,
    frontdoor_building_other_info = $43,
    frontdoor_building_data = $44::jsonb,
    frontdoor_building_processed_at = NOW(),
    frontdoor_building_updated_at = NOW()
WHERE frontdoor_building_housing_company_id = $1;

-- name: MarkFrontdoorBuildingProcessed :exec
UPDATE public.frontdoor_buildings
SET frontdoor_building_processed_at = now(), frontdoor_building_updated_at = now()
WHERE frontdoor_building_id = $1;

-- name: GetFrontdoorBuildingIDByHousingCompanyID :one
SELECT frontdoor_building_id FROM public.frontdoor_buildings
WHERE frontdoor_building_housing_company_id = $1;
