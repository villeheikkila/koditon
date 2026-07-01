-- name: GetShortcutBuildingByID :one
SELECT shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom FROM public.shortcut_buildings
WHERE shortcut_building_id = $1;

-- name: GetShortcutBuildingByExternalID :one
SELECT shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom FROM public.shortcut_buildings
WHERE shortcut_building_external_id = $1;

-- name: ListShortcutBuildings :many
SELECT shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom FROM public.shortcut_buildings
ORDER BY shortcut_building_created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListUnprocessedShortcutBuildings :many
SELECT shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom FROM public.shortcut_buildings
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
RETURNING shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom;

-- name: BatchUpsertShortcutBuildingsFromSitemap :many
INSERT INTO public.shortcut_buildings (
    shortcut_building_external_id,
    shortcut_building_url
)
SELECT UNNEST($1::bigint[]), UNNEST($2::text[])
ON CONFLICT (shortcut_building_external_id) DO UPDATE SET
    shortcut_building_url = EXCLUDED.shortcut_building_url
WHERE public.shortcut_buildings.shortcut_building_url IS DISTINCT FROM EXCLUDED.shortcut_building_url
RETURNING shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom;

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
    @shortcut_building_external_id,
    @shortcut_building_building_id,
    @shortcut_building_building_type,
    @shortcut_building_building_subtype,
    @shortcut_building_construction_year,
    @shortcut_building_floor_count,
    @shortcut_building_apartment_count,
    @shortcut_building_heating_system,
    @shortcut_building_building_material,
    @shortcut_building_plot_type,
    @shortcut_building_wall_structure,
    @shortcut_building_heat_source,
    @shortcut_building_has_elevator,
    @shortcut_building_has_sauna,
    @shortcut_building_latitude,
    @shortcut_building_longitude,
    @shortcut_building_additional_addresses,
    @shortcut_building_url,
    @shortcut_building_address,
    @shortcut_building_frame_construction_method,
    @shortcut_building_housing_company
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
RETURNING shortcut_building_id, shortcut_building_external_id, shortcut_building_building_id, shortcut_building_building_type, shortcut_building_building_subtype, shortcut_building_construction_year, shortcut_building_floor_count, shortcut_building_apartment_count, shortcut_building_heating_system, shortcut_building_building_material, shortcut_building_plot_type, shortcut_building_wall_structure, shortcut_building_heat_source, shortcut_building_has_elevator, shortcut_building_has_sauna, shortcut_building_latitude, shortcut_building_longitude, shortcut_building_additional_addresses, shortcut_building_url, shortcut_building_created_at, shortcut_building_updated_at, shortcut_building_address, shortcut_building_processed_at, shortcut_building_page_not_found, shortcut_building_frame_construction_method, shortcut_building_housing_company, shortcut_building_geom;

-- name: MarkShortcutBuildingProcessed :exec
UPDATE public.shortcut_buildings
SET shortcut_building_processed_at = CURRENT_TIMESTAMP, shortcut_building_updated_at = CURRENT_TIMESTAMP
WHERE shortcut_building_id = $1;

-- name: MarkShortcutBuildingPageNotFound :exec
UPDATE public.shortcut_buildings
SET shortcut_building_page_not_found = true, shortcut_building_updated_at = CURRENT_TIMESTAMP
WHERE shortcut_building_id = $1;
