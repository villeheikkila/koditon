-- name: DeleteSaleListingForShortcutAd :exec
WITH deleted AS (
    DELETE FROM public.property_source_offerings
    WHERE shortcut_ad_id = sqlc.arg(shortcut_ad_id)
    RETURNING sale_listing_id
)
DELETE FROM public.source_listings sl
USING deleted
WHERE sl.source_listing_id = deleted.sale_listing_id;

-- name: CanonicalizeShortcutAdSaleListing :one
INSERT INTO public.property_source_offerings (
    shortcut_ad_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    sale_listing_url,
    sale_listing_headline,
    sale_listing_street_address,
    sale_listing_street_name,
    sale_listing_street_number,
    sale_listing_building_letter,
    sale_listing_city,
    sale_listing_postal,
    sale_listing_asking_price,
    sale_listing_area_value,
    sale_listing_room_layout,
    sale_listing_last_seen_at,
    sale_listing_published_at,
    sale_listing_search_text,
    sale_listing_price_per_m2,
    sale_listing_debt_free_price,
    sale_listing_debt_share_amount,
    sale_listing_rooms_count,
    sale_listing_floor_level,
    sale_listing_total_floors,
    sale_listing_build_year,
    sale_listing_condition,
    sale_listing_energy_class,
    sale_listing_description_text,
    sale_listing_availability_text,
    sale_listing_renovations_done_text,
    sale_listing_renovations_planned_text,
    sale_listing_additional_info_text,
    sale_listing_charges_text,
    sale_listing_maintenance_charge_monthly,
    sale_listing_total_charge_monthly,
    sale_listing_water_charge,
    sale_listing_living_area_value,
    sale_listing_total_area_value,
    sale_listing_other_area_value,
    sale_listing_bedrooms_count,
    sale_listing_sauna,
    sale_listing_balcony,
    sale_listing_parking_text,
    sale_listing_kitchen_description_text,
    sale_listing_bathroom_description_text,
    sale_listing_storage_description_text,
    sale_listing_floor_materials_description_text,
    sale_listing_wall_materials_description_text,
    sale_listing_balcony_description_text,
    sale_listing_sauna_description_text,
    sale_listing_appliances,
    sale_listing_features,
    sale_listing_plot_area_value,
    sale_listing_services_text,
    sale_listing_transport_text,
    sale_listing_new_development,
    sale_listing_updated_at
)
SELECT
    sa.shortcut_ad_id,
    'shortcut',
    'ad',
    sa.shortcut_ad_id::text,
    'shortcut:ad:' || sa.shortcut_ad_id::text,
    sa.shortcut_ad_url,
    COALESCE(raw.street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text),
    COALESCE(raw.street_address, sb.shortcut_building_address),
    NULLIF(trim(sa.shortcut_ad_data #>> '{address,street,name}'), ''),
    NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''),
    NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), ''),
    raw.city,
    raw.postal,
    raw.price,
    raw.area,
    sa.shortcut_ad_data #>> '{adData,roomConfiguration}',
    sa.shortcut_ad_last_seen_at,
    (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz,
    trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)),
    public.fnc__derived_price_per_m2(raw.price, raw.area, raw.price_per_m2),
    raw.debt_free_price,
    raw.debt_share_amount,
    raw.rooms_count,
    raw.floor_level,
    raw.total_floors,
    COALESCE(raw.build_year, sb.shortcut_building_construction_year),
    raw.condition,
    raw.energy_class,
    raw.description_text,
    raw.availability_text,
    raw.renovations_done_text,
    raw.renovations_planned_text,
    raw.additional_info_text,
    raw.charges_text,
    raw.maintenance_charge_monthly,
    raw.total_charge_monthly,
    raw.water_charge,
    raw.living_area,
    raw.total_area,
    raw.other_area,
    raw.bedrooms_count,
    raw.sauna,
    raw.balcony,
    raw.parking_text,
    raw.kitchen_description_text,
    raw.bathroom_description_text,
    raw.storage_description_text,
    raw.floor_materials_description_text,
    raw.wall_materials_description_text,
    raw.balcony_description_text,
    raw.sauna_description_text,
    raw.appliances,
    raw.features,
    raw.plot_area,
    raw.services_text,
    raw.transport_text,
    raw.new_development,
    now()
FROM public.shortcut_ads sa
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
CROSS JOIN LATERAL (
    SELECT
        public.fnc__shortcut_ad_street_address(sa.shortcut_ad_data) AS street_address,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
        COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerDay}')) AS price,
        COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')) AS area,
        COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceDebtFree}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}')) AS debt_free_price,
        public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,debtShare}') AS debt_share_amount,
        COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,pricePerSqm}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}')) AS price_per_m2,
        COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,rooms}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{rooms}')) AS rooms_count,
        COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,floor}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{floor}')) AS floor_level,
        COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,totalFloors}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{buildingData,floors}')) AS total_floors,
        COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{buildingData,year}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,constructionYear}')) AS build_year,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,condition}', sa.shortcut_ad_data #>> '{property,condition}')), '') AS condition,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}')), '') AS energy_class,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,description}', sa.shortcut_ad_data #>> '{description}', sa.shortcut_ad_data #>> '{text}')), '') AS description_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,availabilityDescription}', sa.shortcut_ad_data #>> '{availabilityDescription}', sa.shortcut_ad_data #>> '{adData,availableFrom}')), '') AS availability_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{property,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{adData,renovationInfo}', sa.shortcut_ad_data #>> '{buildingData,renovationInfo}')), '') AS renovations_done_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{property,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{adData,renovationFutureInfo}', sa.shortcut_ad_data #>> '{buildingData,renovationFutureInfo}')), '') AS renovations_planned_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,additionalInfo}', sa.shortcut_ad_data #>> '{moreInformationAvailableFrom}', sa.shortcut_ad_data #>> '{property,otherInfo}')), '') AS additional_info_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{priceData,chargesText}', sa.shortcut_ad_data #>> '{priceData,additionalInfo}', sa.shortcut_ad_data #>> '{property,periodicChargesAdditionalInfo}', sa.shortcut_ad_data #>> '{property,managementChargesAdditionalInfo}')), '') AS charges_text,
        COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,maintenanceCharge}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,monthlyFee}')) AS maintenance_charge_monthly,
        COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,totalCharge}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,monthlyFee}')) AS total_charge_monthly,
        public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,waterFee}') AS water_charge,
        public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}') AS living_area,
        public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}') AS total_area,
        public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeOther}') AS other_area,
        public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,bedrooms}') AS bedrooms_count,
        COALESCE(public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,sauna}'), public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,hasSauna}')) AS sauna,
        public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,balcony}') AS balcony,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,parkingSpaceInfo}', sa.shortcut_ad_data #>> '{adData,carStorageInfo}')), '') AS parking_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,kitchenApplianceInfo}'), '') AS kitchen_description_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,bathroomApplianceInfo}'), '') AS bathroom_description_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,storageInfo}', sa.shortcut_ad_data #>> '{adData,commonAreaInfo}')), '') AS storage_description_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,floorMaterialInfo}'), '') AS floor_materials_description_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,wallMaterialInfo}'), '') AS wall_materials_description_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,balconyInfo}'), '') AS balcony_description_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,saunaInfo}'), '') AS sauna_description_text,
        CASE WHEN jsonb_typeof(sa.shortcut_ad_data #> '{adData,equipment}') = 'array' THEN ARRAY(SELECT jsonb_array_elements_text(sa.shortcut_ad_data #> '{adData,equipment}')) ELSE NULL::text[] END AS appliances,
        CASE WHEN jsonb_typeof(sa.shortcut_ad_data #> '{adData,features}') = 'array' THEN ARRAY(SELECT jsonb_array_elements_text(sa.shortcut_ad_data #> '{adData,features}')) ELSE NULL::text[] END AS features,
        COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,plotArea}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{buildingData,plotArea}')) AS plot_area,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,servicesInfo}'), '') AS services_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,connectionsInfo}'), '') AS transport_text,
        public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,newDevelopment}') AS new_development
) raw
WHERE sa.shortcut_ad_id = sqlc.arg(shortcut_ad_id)
    AND sa.shortcut_ad_type = 'listing'
    AND sa.shortcut_ad_data IS NOT NULL
ON CONFLICT (sale_listing_canonical_id) DO UPDATE SET
    shortcut_ad_id = EXCLUDED.shortcut_ad_id,
    sale_listing_source_provider = EXCLUDED.sale_listing_source_provider,
    sale_listing_source_kind = EXCLUDED.sale_listing_source_kind,
    sale_listing_native_id = EXCLUDED.sale_listing_native_id,
    sale_listing_url = EXCLUDED.sale_listing_url,
    sale_listing_headline = EXCLUDED.sale_listing_headline,
    sale_listing_street_address = EXCLUDED.sale_listing_street_address,
    sale_listing_street_name = EXCLUDED.sale_listing_street_name,
    sale_listing_street_number = EXCLUDED.sale_listing_street_number,
    sale_listing_building_letter = EXCLUDED.sale_listing_building_letter,
    sale_listing_city = EXCLUDED.sale_listing_city,
    sale_listing_postal = EXCLUDED.sale_listing_postal,
    sale_listing_asking_price = EXCLUDED.sale_listing_asking_price,
    sale_listing_area_value = EXCLUDED.sale_listing_area_value,
    sale_listing_room_layout = EXCLUDED.sale_listing_room_layout,
    sale_listing_last_seen_at = EXCLUDED.sale_listing_last_seen_at,
    sale_listing_published_at = EXCLUDED.sale_listing_published_at,
    sale_listing_search_text = EXCLUDED.sale_listing_search_text,
    sale_listing_price_per_m2 = EXCLUDED.sale_listing_price_per_m2,
    sale_listing_debt_free_price = EXCLUDED.sale_listing_debt_free_price,
    sale_listing_debt_share_amount = EXCLUDED.sale_listing_debt_share_amount,
    sale_listing_rooms_count = EXCLUDED.sale_listing_rooms_count,
    sale_listing_floor_level = EXCLUDED.sale_listing_floor_level,
    sale_listing_total_floors = EXCLUDED.sale_listing_total_floors,
    sale_listing_build_year = EXCLUDED.sale_listing_build_year,
    sale_listing_condition = EXCLUDED.sale_listing_condition,
    sale_listing_energy_class = EXCLUDED.sale_listing_energy_class,
    sale_listing_description_text = EXCLUDED.sale_listing_description_text,
    sale_listing_availability_text = EXCLUDED.sale_listing_availability_text,
    sale_listing_renovations_done_text = EXCLUDED.sale_listing_renovations_done_text,
    sale_listing_renovations_planned_text = EXCLUDED.sale_listing_renovations_planned_text,
    sale_listing_additional_info_text = EXCLUDED.sale_listing_additional_info_text,
    sale_listing_charges_text = EXCLUDED.sale_listing_charges_text,
    sale_listing_maintenance_charge_monthly = EXCLUDED.sale_listing_maintenance_charge_monthly,
    sale_listing_total_charge_monthly = EXCLUDED.sale_listing_total_charge_monthly,
    sale_listing_water_charge = EXCLUDED.sale_listing_water_charge,
    sale_listing_living_area_value = EXCLUDED.sale_listing_living_area_value,
    sale_listing_total_area_value = EXCLUDED.sale_listing_total_area_value,
    sale_listing_other_area_value = EXCLUDED.sale_listing_other_area_value,
    sale_listing_bedrooms_count = EXCLUDED.sale_listing_bedrooms_count,
    sale_listing_sauna = EXCLUDED.sale_listing_sauna,
    sale_listing_balcony = EXCLUDED.sale_listing_balcony,
    sale_listing_parking_text = EXCLUDED.sale_listing_parking_text,
    sale_listing_kitchen_description_text = EXCLUDED.sale_listing_kitchen_description_text,
    sale_listing_bathroom_description_text = EXCLUDED.sale_listing_bathroom_description_text,
    sale_listing_storage_description_text = EXCLUDED.sale_listing_storage_description_text,
    sale_listing_floor_materials_description_text = EXCLUDED.sale_listing_floor_materials_description_text,
    sale_listing_wall_materials_description_text = EXCLUDED.sale_listing_wall_materials_description_text,
    sale_listing_balcony_description_text = EXCLUDED.sale_listing_balcony_description_text,
    sale_listing_sauna_description_text = EXCLUDED.sale_listing_sauna_description_text,
    sale_listing_appliances = EXCLUDED.sale_listing_appliances,
    sale_listing_features = EXCLUDED.sale_listing_features,
    sale_listing_plot_area_value = EXCLUDED.sale_listing_plot_area_value,
    sale_listing_services_text = EXCLUDED.sale_listing_services_text,
    sale_listing_transport_text = EXCLUDED.sale_listing_transport_text,
    sale_listing_new_development = EXCLUDED.sale_listing_new_development,
    sale_listing_updated_at = now()
RETURNING sale_listing_id;

-- name: SyncSourceListingFromPropertySourceOffering :exec
INSERT INTO public.source_listings (
    source_listing_id,
    provider,
    source_kind,
    native_id,
    canonical_source_id,
    raw_table,
    raw_id,
    url,
    payload_hash,
    normalized_version,
    normalized_at,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    sl.sale_listing_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.sale_listing_canonical_id,
    CASE
        WHEN sl.shortcut_ad_id IS NOT NULL THEN 'shortcut_ads'
        WHEN sl.frontdoor_ad_id IS NOT NULL THEN 'frontdoor_ads'
        WHEN sl.frontdoor_building_announcement_id IS NOT NULL THEN 'frontdoor_building_announcements'
        WHEN sl.prices_transaction_id IS NOT NULL THEN 'prices_transactions'
        ELSE 'property_source_offerings'
    END,
    COALESCE(sl.shortcut_ad_id::text, sl.frontdoor_ad_id::text, sl.frontdoor_building_announcement_id::text, sl.prices_transaction_id::text, sl.sale_listing_id::text),
    sl.sale_listing_url,
    COALESCE(sa.shortcut_ad_data_hash, fa.frontdoor_ad_data_hash),
    GREATEST(COALESCE(sa.shortcut_ad_data_normalized_version, 0), COALESCE(fa.frontdoor_ad_data_normalized_version, 0)),
    sl.sale_listing_updated_at,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_created_at,
    sl.sale_listing_updated_at
FROM public.property_source_offerings sl
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
ON CONFLICT (source_listing_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    source_kind = EXCLUDED.source_kind,
    native_id = EXCLUDED.native_id,
    canonical_source_id = EXCLUDED.canonical_source_id,
    raw_table = EXCLUDED.raw_table,
    raw_id = EXCLUDED.raw_id,
    url = EXCLUDED.url,
    payload_hash = EXCLUDED.payload_hash,
    normalized_version = EXCLUDED.normalized_version,
    normalized_at = EXCLUDED.normalized_at,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = EXCLUDED.updated_at;

-- name: CanonicalizeFrontdoorAdSaleListing :one
INSERT INTO public.property_source_offerings (
    frontdoor_ad_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    sale_listing_url,
    sale_listing_headline,
    sale_listing_street_address,
    sale_listing_city,
    sale_listing_postal,
    sale_listing_asking_price,
    sale_listing_area_value,
    sale_listing_room_layout,
    sale_listing_last_seen_at,
    sale_listing_published_at,
    sale_listing_search_text,
    sale_listing_price_per_m2,
    sale_listing_debt_free_price,
    sale_listing_debt_share_amount,
    sale_listing_rooms_count,
    sale_listing_floor_level,
    sale_listing_total_floors,
    sale_listing_build_year,
    sale_listing_condition,
    sale_listing_energy_class,
    sale_listing_description_text,
    sale_listing_availability_text,
    sale_listing_renovations_done_text,
    sale_listing_renovations_planned_text,
    sale_listing_additional_info_text,
    sale_listing_charges_text,
    sale_listing_maintenance_charge_monthly,
    sale_listing_total_charge_monthly,
    sale_listing_water_charge,
    sale_listing_living_area_value,
    sale_listing_total_area_value,
    sale_listing_other_area_value,
    sale_listing_bedrooms_count,
    sale_listing_sauna,
    sale_listing_balcony,
    sale_listing_parking_text,
    sale_listing_kitchen_description_text,
    sale_listing_bathroom_description_text,
    sale_listing_storage_description_text,
    sale_listing_floor_materials_description_text,
    sale_listing_wall_materials_description_text,
    sale_listing_balcony_description_text,
    sale_listing_sauna_description_text,
    sale_listing_views_description_text,
    sale_listing_features,
    sale_listing_plot_area_value,
    sale_listing_services_text,
    sale_listing_transport_text,
    sale_listing_previous_asking_price,
    sale_listing_previous_debt_free_price,
    sale_listing_new_development,
    sale_listing_updated_at
)
SELECT
    fa.frontdoor_ad_id,
    'frontdoor',
    'ad',
    fa.frontdoor_ad_external_id,
    'frontdoor:ad:' || fa.frontdoor_ad_external_id,
    fa.frontdoor_ad_url,
    COALESCE(raw.street_address, fa.frontdoor_ad_external_id),
    raw.street_address,
    raw.city,
    raw.postal,
    raw.price,
    raw.area,
    fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}',
    fa.frontdoor_ad_last_seen_at,
    public.fnc__frontdoor_published_at(fa.frontdoor_ad_data),
    trim(concat_ws(' ', fa.frontdoor_ad_external_id, fa.frontdoor_ad_url, raw.street_address, raw.city, raw.postal, fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}')),
    public.fnc__derived_price_per_m2(raw.price, raw.area, raw.price_per_m2),
    raw.debt_free_price,
    raw.debt_share_amount,
    raw.rooms_count,
    raw.floor_level,
    raw.total_floors,
    raw.build_year,
    raw.condition,
    raw.energy_class,
    raw.description_text,
    raw.availability_text,
    raw.renovations_done_text,
    raw.renovations_planned_text,
    raw.additional_info_text,
    raw.charges_text,
    raw.maintenance_charge_monthly,
    raw.total_charge_monthly,
    raw.water_charge,
    raw.living_area,
    raw.total_area,
    raw.other_area,
    raw.bedrooms_count,
    raw.sauna,
    raw.balcony,
    raw.parking_text,
    raw.kitchen_description_text,
    raw.bathroom_description_text,
    raw.storage_description_text,
    raw.floor_materials_description_text,
    raw.wall_materials_description_text,
    raw.balcony_description_text,
    raw.sauna_description_text,
    raw.views_description_text,
    raw.features,
    raw.plot_area,
    raw.services_text,
    raw.transport_text,
    raw.previous_asking_price,
    raw.previous_debt_free_price,
    raw.new_development,
    now()
FROM public.frontdoor_ads fa
CROSS JOIN LATERAL (
    SELECT
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}', fa.frontdoor_ad_data #>> '{property,streetNameFreeForm}')), '') AS street_address,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,municipalityNameFreeForm}', fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}', fa.frontdoor_ad_data #>> '{property,postCode,postArea}')), '') AS city,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}', fa.frontdoor_ad_data #>> '{property,postCode,postCode}')), '') AS postal,
        COALESCE(public.fnc__try_parse_bigint(fa.frontdoor_ad_data #>> '{debfFreePrice}'), public.fnc__try_parse_bigint(fa.frontdoor_ad_data #>> '{preparsed,price}')) AS price,
        COALESCE(public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{preparsed,area}'), public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{property,livingArea}')) AS area,
        public.fnc__try_parse_bigint(fa.frontdoor_ad_data #>> '{debfFreePrice}') AS debt_free_price,
        public.fnc__try_parse_bigint(fa.frontdoor_ad_data #>> '{debtShareAmount}') AS debt_share_amount,
        COALESCE(public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{pricePerSquareMeter}'), public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{preparsed,pricePerSquareMeter}')) AS price_per_m2,
        public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,totalRoomCount}') AS rooms_count,
        COALESCE(public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,housingCompanyApartmentInformationDTO,floorLevel}'), public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{property,floorLevel}')) AS floor_level,
        COALESCE(public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{property,housingCompany,floorCount}'), public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,floorCount}')) AS total_floors,
        COALESCE(public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,constructionFinishedYear}'), public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{property,housingCompany,usageStartYear}')) AS build_year,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,inspection,overallCondition}', fa.frontdoor_ad_data #>> '{property,condition}')), '') AS condition,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}')), '') AS energy_class,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{text}', fa.frontdoor_ad_data #>> '{property,description}')), '') AS description_text,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{availabilityDescription}'), '') AS availability_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDone}')), '') AS renovations_done_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlanned}')), '') AS renovations_planned_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{moreInformationAvailableFrom}', fa.frontdoor_ad_data #>> '{property,housingCompany,otherInfo}', fa.frontdoor_ad_data #>> '{additionalItemsIncludedInSale}')), '') AS additional_info_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,periodicChargesAdditionalInfo}', fa.frontdoor_ad_data #>> '{property,managementChargesAdditionalInfo}')), '') AS charges_text,
        public.fnc__jsonb_periodic_charge_price(fa.frontdoor_ad_data, 'HOUSING_COMPANY_MAINTENANCE_CHARGE') AS maintenance_charge_monthly,
        public.fnc__jsonb_periodic_charge_price(fa.frontdoor_ad_data, 'HOUSING_COMPANY_TOTAL_CHARGE') AS total_charge_monthly,
        public.fnc__jsonb_periodic_charge_price(fa.frontdoor_ad_data, 'WATER') AS water_charge,
        public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,livingArea}') AS living_area,
        public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,totalArea}') AS total_area,
        public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,otherArea}') AS other_area,
        public.fnc__try_parse_int4(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,bedroomCount}') AS bedrooms_count,
        CASE
            WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
            WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
            ELSE public.fnc__try_parse_bool(fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}')
        END AS sauna,
        COALESCE(public.fnc__try_parse_bool(fa.frontdoor_ad_data #>> '{property,hasBalcony}'), CASE WHEN NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,balconyDescription}', fa.frontdoor_ad_data #>> '{property,balconyDescription}')), '') IS NOT NULL THEN true ELSE NULL::boolean END) AS balcony,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{property,carParkingInformation}'), '') AS parking_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,kitchenDescription}', fa.frontdoor_ad_data #>> '{property,kitchenDescription}')), '') AS kitchen_description_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,bathroomDescription}', fa.frontdoor_ad_data #>> '{property,bathroomDescription}')), '') AS bathroom_description_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,storageSpacesDescription}', fa.frontdoor_ad_data #>> '{residenceDetailsDTO,storageSpacesDescription}')), '') AS storage_description_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,floorMaterialDescription}', fa.frontdoor_ad_data #>> '{property,floorMaterialDescription}')), '') AS floor_materials_description_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,wallMaterialDescription}', fa.frontdoor_ad_data #>> '{property,wallMaterialDescription}')), '') AS wall_materials_description_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,balconyDescription}', fa.frontdoor_ad_data #>> '{property,balconyDescription}')), '') AS balcony_description_text,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,saunaDescription}'), '') AS sauna_description_text,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,viewsDescription}'), '') AS views_description_text,
        CASE WHEN jsonb_typeof(fa.frontdoor_ad_data #> '{residenceDetailsDTO,generalDwellingFeatures}') = 'array' THEN ARRAY(SELECT jsonb_array_elements_text(fa.frontdoor_ad_data #> '{residenceDetailsDTO,generalDwellingFeatures}')) ELSE NULL::text[] END AS features,
        public.fnc__try_parse_float8(fa.frontdoor_ad_data #>> '{property,plot,area}') AS plot_area,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{property,nearbyAmenitiesDescription}'), '') AS services_text,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{property,transportationServicesDescription}'), '') AS transport_text,
        public.fnc__try_parse_bigint(fa.frontdoor_ad_data #>> '{previousPrice}') AS previous_asking_price,
        public.fnc__try_parse_bigint(fa.frontdoor_ad_data #>> '{previousDebtFreePrice}') AS previous_debt_free_price,
        public.fnc__try_parse_bool(fa.frontdoor_ad_data #>> '{newProperty}') AS new_development
) raw
WHERE fa.frontdoor_ad_id = sqlc.arg(frontdoor_ad_id)
    AND fa.frontdoor_ad_data IS NOT NULL
ON CONFLICT (sale_listing_canonical_id) DO UPDATE SET
    frontdoor_ad_id = EXCLUDED.frontdoor_ad_id,
    sale_listing_source_provider = EXCLUDED.sale_listing_source_provider,
    sale_listing_source_kind = EXCLUDED.sale_listing_source_kind,
    sale_listing_native_id = EXCLUDED.sale_listing_native_id,
    sale_listing_url = EXCLUDED.sale_listing_url,
    sale_listing_headline = EXCLUDED.sale_listing_headline,
    sale_listing_street_address = EXCLUDED.sale_listing_street_address,
    sale_listing_city = EXCLUDED.sale_listing_city,
    sale_listing_postal = EXCLUDED.sale_listing_postal,
    sale_listing_asking_price = EXCLUDED.sale_listing_asking_price,
    sale_listing_area_value = EXCLUDED.sale_listing_area_value,
    sale_listing_room_layout = EXCLUDED.sale_listing_room_layout,
    sale_listing_last_seen_at = EXCLUDED.sale_listing_last_seen_at,
    sale_listing_published_at = EXCLUDED.sale_listing_published_at,
    sale_listing_search_text = EXCLUDED.sale_listing_search_text,
    sale_listing_price_per_m2 = EXCLUDED.sale_listing_price_per_m2,
    sale_listing_debt_free_price = EXCLUDED.sale_listing_debt_free_price,
    sale_listing_debt_share_amount = EXCLUDED.sale_listing_debt_share_amount,
    sale_listing_rooms_count = EXCLUDED.sale_listing_rooms_count,
    sale_listing_floor_level = EXCLUDED.sale_listing_floor_level,
    sale_listing_total_floors = EXCLUDED.sale_listing_total_floors,
    sale_listing_build_year = EXCLUDED.sale_listing_build_year,
    sale_listing_condition = EXCLUDED.sale_listing_condition,
    sale_listing_energy_class = EXCLUDED.sale_listing_energy_class,
    sale_listing_description_text = EXCLUDED.sale_listing_description_text,
    sale_listing_availability_text = EXCLUDED.sale_listing_availability_text,
    sale_listing_renovations_done_text = EXCLUDED.sale_listing_renovations_done_text,
    sale_listing_renovations_planned_text = EXCLUDED.sale_listing_renovations_planned_text,
    sale_listing_additional_info_text = EXCLUDED.sale_listing_additional_info_text,
    sale_listing_charges_text = EXCLUDED.sale_listing_charges_text,
    sale_listing_maintenance_charge_monthly = EXCLUDED.sale_listing_maintenance_charge_monthly,
    sale_listing_total_charge_monthly = EXCLUDED.sale_listing_total_charge_monthly,
    sale_listing_water_charge = EXCLUDED.sale_listing_water_charge,
    sale_listing_living_area_value = EXCLUDED.sale_listing_living_area_value,
    sale_listing_total_area_value = EXCLUDED.sale_listing_total_area_value,
    sale_listing_other_area_value = EXCLUDED.sale_listing_other_area_value,
    sale_listing_bedrooms_count = EXCLUDED.sale_listing_bedrooms_count,
    sale_listing_sauna = EXCLUDED.sale_listing_sauna,
    sale_listing_balcony = EXCLUDED.sale_listing_balcony,
    sale_listing_parking_text = EXCLUDED.sale_listing_parking_text,
    sale_listing_kitchen_description_text = EXCLUDED.sale_listing_kitchen_description_text,
    sale_listing_bathroom_description_text = EXCLUDED.sale_listing_bathroom_description_text,
    sale_listing_storage_description_text = EXCLUDED.sale_listing_storage_description_text,
    sale_listing_floor_materials_description_text = EXCLUDED.sale_listing_floor_materials_description_text,
    sale_listing_wall_materials_description_text = EXCLUDED.sale_listing_wall_materials_description_text,
    sale_listing_balcony_description_text = EXCLUDED.sale_listing_balcony_description_text,
    sale_listing_sauna_description_text = EXCLUDED.sale_listing_sauna_description_text,
    sale_listing_views_description_text = EXCLUDED.sale_listing_views_description_text,
    sale_listing_features = EXCLUDED.sale_listing_features,
    sale_listing_plot_area_value = EXCLUDED.sale_listing_plot_area_value,
    sale_listing_services_text = EXCLUDED.sale_listing_services_text,
    sale_listing_transport_text = EXCLUDED.sale_listing_transport_text,
    sale_listing_previous_asking_price = EXCLUDED.sale_listing_previous_asking_price,
    sale_listing_previous_debt_free_price = EXCLUDED.sale_listing_previous_debt_free_price,
    sale_listing_new_development = EXCLUDED.sale_listing_new_development,
    sale_listing_updated_at = now()
RETURNING sale_listing_id;

-- name: DeletePropertySourceOfferingForFrontdoorBuildingAnnouncement :exec
WITH deleted AS (
    DELETE FROM public.property_source_offerings
    WHERE frontdoor_building_announcement_id = sqlc.arg(frontdoor_building_announcement_id)
    RETURNING sale_listing_id
)
DELETE FROM public.source_listings sl
USING deleted
WHERE sl.source_listing_id = deleted.sale_listing_id;

-- name: CanonicalizeFrontdoorBuildingAnnouncementSourceOffering :one
INSERT INTO public.property_source_offerings (
    frontdoor_building_announcement_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    sale_listing_url,
    sale_listing_headline,
    sale_listing_street_address,
    sale_listing_city,
    sale_listing_postal,
    sale_listing_asking_price,
    sale_listing_area_value,
    sale_listing_room_layout,
    sale_listing_last_seen_at,
    sale_listing_published_at,
    sale_listing_search_text,
    sale_listing_price_per_m2,
    sale_listing_rooms_count,
    sale_listing_total_floors,
    sale_listing_build_year,
    sale_listing_property_type_raw,
    sale_listing_elevator,
    sale_listing_energy_class,
    sale_listing_energy_efficiency_label,
    sale_listing_housing_company_name,
    sale_listing_housing_company_business_id,
    sale_listing_building_material,
    sale_listing_heating_system,
    sale_listing_roof_type,
    sale_listing_roof_material,
    sale_listing_apartment_count,
    sale_listing_car_storage_text,
    sale_listing_building_description_text,
    sale_listing_building_other_info_text,
    sale_listing_latitude,
    sale_listing_longitude,
    sale_listing_new_development,
    sale_listing_first_seen_at,
    sale_listing_updated_at
)
SELECT
    fba.frontdoor_building_announcement_id,
    'frontdoor',
    'announcement',
    fba.frontdoor_building_announcement_id::text,
    'frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text,
    fb.frontdoor_building_url,
    COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text),
    concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2),
    COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area),
    fb.frontdoor_building_postcode,
    CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END,
    fba.frontdoor_building_announcement_area,
    fba.frontdoor_building_announcement_room_structure,
    fba.frontdoor_building_announcement_last_seen_at,
    NULL::timestamptz,
    concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure, fb.frontdoor_building_company_name, fb.frontdoor_building_business_id),
    fba.frontdoor_building_announcement_price_per_square,
    NULL::integer,
    fb.frontdoor_building_floor_count,
    COALESCE(fba.frontdoor_building_announcement_construction_finished_year, fb.frontdoor_building_build_year, fb.frontdoor_building_construction_end_year),
    NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
    fb.frontdoor_building_has_elevator,
    fb.frontdoor_building_energy_certificate_code,
    fb.frontdoor_building_energy_certificate_code,
    fb.frontdoor_building_company_name,
    fb.frontdoor_building_business_id,
    NULL::text,
    concat_ws(', ', fb.frontdoor_building_heating, array_to_string(fb.frontdoor_building_heating_fuel, ', ')),
    fb.frontdoor_building_outer_roof_type,
    fb.frontdoor_building_outer_roof_material,
    fb.frontdoor_building_apartment_count,
    fb.frontdoor_building_car_storage_description,
    fb.frontdoor_building_description,
    fb.frontdoor_building_other_info,
    fb.frontdoor_building_latitude,
    fb.frontdoor_building_longitude,
    fba.frontdoor_building_announcement_new_building,
    fba.frontdoor_building_announcement_first_seen_at,
    now()
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE fba.frontdoor_building_announcement_id = sqlc.arg(frontdoor_building_announcement_id)
    AND fba.frontdoor_building_announcement_rent_period IS NULL
    AND fba.frontdoor_building_announcement_rental_unique_no IS NULL
ON CONFLICT (sale_listing_canonical_id) DO UPDATE SET
    frontdoor_building_announcement_id = EXCLUDED.frontdoor_building_announcement_id,
    sale_listing_source_provider = EXCLUDED.sale_listing_source_provider,
    sale_listing_source_kind = EXCLUDED.sale_listing_source_kind,
    sale_listing_native_id = EXCLUDED.sale_listing_native_id,
    sale_listing_url = EXCLUDED.sale_listing_url,
    sale_listing_headline = EXCLUDED.sale_listing_headline,
    sale_listing_street_address = EXCLUDED.sale_listing_street_address,
    sale_listing_city = EXCLUDED.sale_listing_city,
    sale_listing_postal = EXCLUDED.sale_listing_postal,
    sale_listing_asking_price = EXCLUDED.sale_listing_asking_price,
    sale_listing_area_value = EXCLUDED.sale_listing_area_value,
    sale_listing_room_layout = EXCLUDED.sale_listing_room_layout,
    sale_listing_last_seen_at = EXCLUDED.sale_listing_last_seen_at,
    sale_listing_search_text = EXCLUDED.sale_listing_search_text,
    sale_listing_price_per_m2 = EXCLUDED.sale_listing_price_per_m2,
    sale_listing_total_floors = EXCLUDED.sale_listing_total_floors,
    sale_listing_build_year = EXCLUDED.sale_listing_build_year,
    sale_listing_property_type_raw = EXCLUDED.sale_listing_property_type_raw,
    sale_listing_elevator = EXCLUDED.sale_listing_elevator,
    sale_listing_energy_class = EXCLUDED.sale_listing_energy_class,
    sale_listing_energy_efficiency_label = EXCLUDED.sale_listing_energy_efficiency_label,
    sale_listing_housing_company_name = EXCLUDED.sale_listing_housing_company_name,
    sale_listing_housing_company_business_id = EXCLUDED.sale_listing_housing_company_business_id,
    sale_listing_building_material = EXCLUDED.sale_listing_building_material,
    sale_listing_heating_system = EXCLUDED.sale_listing_heating_system,
    sale_listing_roof_type = EXCLUDED.sale_listing_roof_type,
    sale_listing_roof_material = EXCLUDED.sale_listing_roof_material,
    sale_listing_apartment_count = EXCLUDED.sale_listing_apartment_count,
    sale_listing_car_storage_text = EXCLUDED.sale_listing_car_storage_text,
    sale_listing_building_description_text = EXCLUDED.sale_listing_building_description_text,
    sale_listing_building_other_info_text = EXCLUDED.sale_listing_building_other_info_text,
    sale_listing_latitude = EXCLUDED.sale_listing_latitude,
    sale_listing_longitude = EXCLUDED.sale_listing_longitude,
    sale_listing_new_development = EXCLUDED.sale_listing_new_development,
    sale_listing_first_seen_at = EXCLUDED.sale_listing_first_seen_at,
    sale_listing_updated_at = now()
RETURNING sale_listing_id;

-- name: RefreshPropertySourceOfferingRenovationsFromFrontdoorBuilding :exec
WITH listing AS (
    SELECT sl.sale_listing_id, fb.*
    FROM public.property_source_offerings sl
    JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
),
deleted AS (
    DELETE FROM public.property_source_offering_renovations
    WHERE sale_listing_id = sqlc.arg(sale_listing_id)
)
INSERT INTO public.property_source_offering_renovations (
    sale_listing_id,
    property_source_offering_renovation_source_field,
    property_source_offering_renovation_category,
    property_source_offering_renovation_status,
    property_source_offering_renovation_year,
    property_source_offering_renovation_component,
    property_source_offering_renovation_scope,
    property_source_offering_renovation_stage,
    property_source_offering_renovation_responsibility,
    property_source_offering_renovation_cost_estimate_eur,
    property_source_offering_renovation_text,
    property_source_offering_renovation_confidence
)
SELECT
    listing.sale_listing_id,
    renovation.source_field,
    renovation.category,
    'done',
    renovation.year,
    NULL,
    'unknown',
    'completed',
    'housing_company',
    NULL,
    renovation.text,
    100
FROM listing
CROSS JOIN LATERAL (
    VALUES
        ('frontdoor_building_elevator_renovated', 'elevator', listing.frontdoor_building_elevator_renovated, listing.frontdoor_building_elevator_renovated_year, NULL::text),
        ('frontdoor_building_facade_renovated', 'facade', listing.frontdoor_building_facade_renovated, listing.frontdoor_building_facade_renovated_year, NULL::text),
        ('frontdoor_building_window_renovated', 'window', listing.frontdoor_building_window_renovated, listing.frontdoor_building_window_renovated_year, NULL::text),
        ('frontdoor_building_roof_renovated', 'roof', listing.frontdoor_building_roof_renovated, listing.frontdoor_building_roof_renovated_year, NULL::text),
        ('frontdoor_building_pipe_renovated', 'pipe', listing.frontdoor_building_pipe_renovated, listing.frontdoor_building_pipe_renovated_year, NULL::text),
        ('frontdoor_building_balcony_renovated', 'balcony', listing.frontdoor_building_balcony_renovated, listing.frontdoor_building_balcony_renovated_year, NULL::text),
        ('frontdoor_building_electricity_renovated', 'electricity', listing.frontdoor_building_electricity_renovated, listing.frontdoor_building_electricity_renovated_year, NULL::text)
) AS renovation(source_field, category, done, year, text)
WHERE renovation.done IS TRUE;

-- name: RebuildListingDimensionLayer :one
SELECT public.fnc__rebuild_listing_dimension_layer(sqlc.arg(sale_listing_id)::uuid)::jsonb AS payload;

-- name: RebuildListingDimensionLayerAt :one
SELECT public.fnc__rebuild_listing_dimension_layer(sqlc.arg(sale_listing_id)::uuid, sqlc.narg(expected_dirty_at)::timestamptz)::jsonb AS payload;

-- name: ProjectListingProviderDimensionClaims :one
SELECT public.fnc__project_listing_provider_dimension_claims(sqlc.arg(sale_listing_id)::uuid)::integer;

-- name: ResolveDimensionValuesForTarget :one
SELECT public.fnc__resolve_dimension_values_for_target(sqlc.arg(target_type)::text, sqlc.arg(target_id)::uuid)::integer AS count;

-- name: ProjectDimensionProfileForTarget :one
SELECT public.fnc__project_dimension_profile_for_target(sqlc.arg(target_type)::text, sqlc.arg(target_id)::uuid)::integer AS count;

-- name: ResolveDimensionTarget :one
SELECT public.fnc__resolve_dimension_target(sqlc.arg(target_type)::text, sqlc.arg(target_id)::uuid, sqlc.narg(expected_dirty_at)::timestamptz)::jsonb AS payload;

-- name: MarkListingDimensionTargetsDirty :one
SELECT public.fnc__mark_listing_dimension_targets_dirty(sqlc.arg(sale_listing_id)::uuid, sqlc.arg(reason)::text)::integer;

-- name: EnsurePhysicalBuildingForSaleListing :exec
WITH linked AS (
    SELECT
        source_link.source_id AS sale_listing_id,
        pu.property_unit_id,
        pu.housing_company_id,
        hc.housing_company_identity_key
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = sqlc.arg(sale_listing_id)::uuid
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
),
listing AS (
    SELECT
        sl.*,
        linked.housing_company_id,
        linked.property_unit_id,
        linked.housing_company_identity_key
    FROM public.property_source_offerings sl
    JOIN linked ON linked.sale_listing_id = sl.sale_listing_id
    WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)::uuid
),
inserted AS (
    INSERT INTO public.physical_buildings (
        housing_company_id,
        physical_building_identity_key,
        physical_building_address_norm,
        physical_building_postal_norm,
        physical_building_city_norm,
        physical_building_build_year,
        physical_building_floor_count,
        physical_building_apartment_count,
        physical_building_elevator,
        physical_building_latitude,
        physical_building_longitude,
        physical_building_updated_at
    )
    SELECT
        housing_company_id,
        housing_company_identity_key || ':building:' || COALESCE(public.fnc__canonical_identity_part(sale_listing_address_norm), 'main'),
        sale_listing_address_norm,
        sale_listing_postal_norm,
        sale_listing_city_norm,
        sale_listing_build_year,
        sale_listing_total_floors,
        sale_listing_apartment_count,
        sale_listing_elevator,
        sale_listing_latitude,
        sale_listing_longitude,
        now()
    FROM listing
    ON CONFLICT (physical_building_identity_key) DO UPDATE SET
        housing_company_id = COALESCE(public.physical_buildings.housing_company_id, EXCLUDED.housing_company_id),
        physical_building_address_norm = COALESCE(public.physical_buildings.physical_building_address_norm, EXCLUDED.physical_building_address_norm),
        physical_building_postal_norm = COALESCE(public.physical_buildings.physical_building_postal_norm, EXCLUDED.physical_building_postal_norm),
        physical_building_city_norm = COALESCE(public.physical_buildings.physical_building_city_norm, EXCLUDED.physical_building_city_norm),
        physical_building_build_year = COALESCE(public.physical_buildings.physical_building_build_year, EXCLUDED.physical_building_build_year),
        physical_building_floor_count = COALESCE(public.physical_buildings.physical_building_floor_count, EXCLUDED.physical_building_floor_count),
        physical_building_apartment_count = COALESCE(public.physical_buildings.physical_building_apartment_count, EXCLUDED.physical_building_apartment_count),
        physical_building_elevator = COALESCE(public.physical_buildings.physical_building_elevator, EXCLUDED.physical_building_elevator),
        physical_building_latitude = COALESCE(public.physical_buildings.physical_building_latitude, EXCLUDED.physical_building_latitude),
        physical_building_longitude = COALESCE(public.physical_buildings.physical_building_longitude, EXCLUDED.physical_building_longitude),
        physical_building_updated_at = now()
    RETURNING physical_building_id
),
updated AS (
    UPDATE public.property_units pu
    SET physical_building_id = inserted.physical_building_id,
        property_unit_updated_at = now()
    FROM listing, inserted
    WHERE pu.property_unit_id = listing.property_unit_id
    RETURNING pu.property_unit_id
)
INSERT INTO public.units (
    unit_id,
    housing_company_id,
    physical_building_id,
    identity_key,
    address_norm,
    apartment,
    floor_level,
    area_m2,
    room_layout,
    created_at,
    updated_at
)
SELECT
    pu.property_unit_id,
    pu.housing_company_id,
    pu.physical_building_id,
    pu.property_unit_identity_key,
    pu.property_unit_address_norm,
    NULL::text,
    pu.property_unit_floor_level,
    pu.property_unit_area_value,
    pu.property_unit_room_layout,
    pu.property_unit_created_at,
    pu.property_unit_updated_at
FROM updated
JOIN public.property_units pu ON pu.property_unit_id = updated.property_unit_id
ON CONFLICT (unit_id) DO UPDATE SET
    housing_company_id = EXCLUDED.housing_company_id,
    physical_building_id = EXCLUDED.physical_building_id,
    identity_key = EXCLUDED.identity_key,
    address_norm = EXCLUDED.address_norm,
    apartment = EXCLUDED.apartment,
    floor_level = EXCLUDED.floor_level,
    area_m2 = EXCLUDED.area_m2,
    room_layout = EXCLUDED.room_layout,
    updated_at = EXCLUDED.updated_at;

-- name: SyncPropertyHouseForSaleListing :one
WITH synced AS (
    SELECT COALESCE(public.fnc__sync_property_house_for_sale_listing(sqlc.arg(sale_listing_id)::uuid, sqlc.arg(link_method)::text), '00000000-0000-0000-0000-000000000000'::uuid) AS property_house_id
),
synced_houses AS (
    INSERT INTO public.houses (
        house_id,
        identity_key,
        address_norm,
        postal_norm,
        city_norm,
        latitude,
        longitude,
        created_at,
        updated_at
    )
    SELECT
        ph.property_house_id,
        ph.property_house_identity_key,
        ph.property_house_address_norm,
        ph.property_house_postal_norm,
        ph.property_house_city_norm,
        ph.property_house_latitude,
        ph.property_house_longitude,
        ph.property_house_created_at,
        ph.property_house_updated_at
    FROM synced
    JOIN public.property_houses ph ON ph.property_house_id = synced.property_house_id
    WHERE synced.property_house_id <> '00000000-0000-0000-0000-000000000000'::uuid
    ON CONFLICT (house_id) DO UPDATE SET
        identity_key = EXCLUDED.identity_key,
        address_norm = EXCLUDED.address_norm,
        postal_norm = EXCLUDED.postal_norm,
        city_norm = EXCLUDED.city_norm,
        latitude = EXCLUDED.latitude,
        longitude = EXCLUDED.longitude,
        updated_at = EXCLUDED.updated_at
    RETURNING house_id
)
SELECT property_house_id::uuid FROM synced;

-- name: BackfillDetachedPropertyHouses :one
WITH candidates AS (
    SELECT sl.sale_listing_id
    FROM public.property_source_offerings sl
    JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    WHERE sl.sale_listing_property_type_code = 'detached_house'
        AND po.property_house_id IS NULL
    ORDER BY sl.sale_listing_updated_at DESC NULLS LAST, sl.sale_listing_id
    LIMIT sqlc.arg(batch_size)::int
),
synced AS (
    SELECT sale_listing_id, public.fnc__sync_property_house_for_sale_listing(sale_listing_id, 'regroup_v2_backfill') AS property_house_id
    FROM candidates
),
synced_houses AS (
    INSERT INTO public.houses (
        house_id,
        identity_key,
        address_norm,
        postal_norm,
        city_norm,
        latitude,
        longitude,
        created_at,
        updated_at
    )
    SELECT
        ph.property_house_id,
        ph.property_house_identity_key,
        ph.property_house_address_norm,
        ph.property_house_postal_norm,
        ph.property_house_city_norm,
        ph.property_house_latitude,
        ph.property_house_longitude,
        ph.property_house_created_at,
        ph.property_house_updated_at
    FROM synced
    JOIN public.property_houses ph ON ph.property_house_id = synced.property_house_id
    WHERE synced.property_house_id IS NOT NULL
    ON CONFLICT (house_id) DO UPDATE SET
        identity_key = EXCLUDED.identity_key,
        address_norm = EXCLUDED.address_norm,
        postal_norm = EXCLUDED.postal_norm,
        city_norm = EXCLUDED.city_norm,
        latitude = EXCLUDED.latitude,
        longitude = EXCLUDED.longitude,
        updated_at = EXCLUDED.updated_at
    RETURNING house_id
),
synced_listings AS (
    INSERT INTO public.listings (
        listing_id,
        listing_type,
        listing_status,
        primary_source_listing_id,
        unit_id,
        house_id,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        po.property_offering_id,
        po.property_offering_type,
        po.property_offering_status,
        po.primary_sale_listing_id,
        po.property_unit_id,
        po.property_house_id,
        po.property_offering_first_seen_at,
        po.property_offering_last_seen_at,
        po.property_offering_created_at,
        po.property_offering_updated_at
    FROM synced
    JOIN public.target_sources source_link ON source_link.source_id = synced.sale_listing_id
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    WHERE synced.property_house_id IS NOT NULL
    ON CONFLICT (listing_id) DO UPDATE SET
        listing_type = EXCLUDED.listing_type,
        listing_status = EXCLUDED.listing_status,
        primary_source_listing_id = EXCLUDED.primary_source_listing_id,
        unit_id = EXCLUDED.unit_id,
        house_id = EXCLUDED.house_id,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        updated_at = EXCLUDED.updated_at
    RETURNING listing_id
)
SELECT count(*)::integer
FROM synced
WHERE property_house_id IS NOT NULL;

-- name: GetPropertySourceOfferingDescriptionTexts :one
SELECT
    COALESCE(sale_listing_description_text, '') AS description_text,
    COALESCE(sale_listing_building_description_text, sale_listing_building_other_info_text, '') AS building_text,
    COALESCE(sale_listing_additional_info_text, '') AS additional_info_text
FROM public.property_source_offerings
WHERE sale_listing_id = sqlc.arg(sale_listing_id)
LIMIT 1;

-- name: GetPropertySourceOfferingValuationExtractionTexts :one
SELECT
    COALESCE(sale_listing_room_layout, '') AS room_layout,
    sale_listing_rooms_count,
    sale_listing_bedrooms_count,
    sale_listing_area_value,
    sale_listing_living_area_value,
    sale_listing_total_area_value,
    sale_listing_other_area_value,
    sale_listing_floor_level,
    sale_listing_total_floors,
    COALESCE(sale_listing_floor_text, '') AS floor_text,
    COALESCE(sale_listing_condition, '') AS condition,
    sale_listing_sauna,
    sale_listing_balcony,
    COALESCE(sale_listing_parking_text, '') AS parking_text,
    COALESCE(sale_listing_description_text, '') AS description_text,
    COALESCE(sale_listing_additional_info_text, '') AS additional_info_text,
    COALESCE(sale_listing_kitchen_description_text, '') AS kitchen_description_text,
    COALESCE(sale_listing_bathroom_description_text, '') AS bathroom_description_text,
    COALESCE(sale_listing_storage_description_text, '') AS storage_description_text,
    COALESCE(sale_listing_floor_materials_description_text, '') AS floor_materials_description_text,
    COALESCE(sale_listing_wall_materials_description_text, '') AS wall_materials_description_text,
    COALESCE(sale_listing_balcony_description_text, '') AS balcony_description_text,
    COALESCE(sale_listing_sauna_description_text, '') AS sauna_description_text,
    COALESCE(sale_listing_views_description_text, '') AS views_description_text,
    COALESCE(sale_listing_building_material, '') AS building_material,
    COALESCE(sale_listing_heating_system, '') AS heating_system,
    COALESCE(sale_listing_roof_type, '') AS roof_type,
    COALESCE(sale_listing_roof_material, '') AS roof_material,
    COALESCE(sale_listing_car_storage_text, '') AS car_storage_text,
    COALESCE(sale_listing_building_description_text, '') AS building_description_text,
    COALESCE(sale_listing_building_other_info_text, '') AS building_other_info_text,
    COALESCE(sale_listing_charges_text, '') AS charges_text
FROM public.property_source_offerings
WHERE sale_listing_id = sqlc.arg(sale_listing_id)
LIMIT 1;

-- name: ListPropertySourceOfferingInsights :many
SELECT
    observation.observation_key AS property_source_offering_insight_key,
    COALESCE(observation.value #>> '{}', '')::text AS property_source_offering_insight_value,
    observation.direction AS property_source_offering_insight_direction,
    observation.severity AS property_source_offering_insight_severity,
    round(observation.confidence * 100)::integer AS property_source_offering_insight_confidence,
    COALESCE(observation.evidence ->> 'source_field', '')::text AS property_source_offering_insight_source_field,
    COALESCE(observation.text, '') AS property_source_offering_insight_text
FROM public.target_observations observation
WHERE observation.source_type = 'source_listing'
    AND observation.source_id = sqlc.arg(sale_listing_id)
    AND observation.superseded_at IS NULL
ORDER BY observation.severity DESC, observation.observation_key;

-- name: ListPropertyClaimsForEntity :many
SELECT
    COALESCE(source_field, CASE WHEN extraction_model IS NOT NULL THEN 'llm' ELSE 'provider_field' END)::text AS property_claim_source_field,
    split_part(dimension_key, '.', 1)::text AS property_claim_namespace,
    substring(dimension_key from position('.' in dimension_key) + 1)::text AS property_claim_key,
    CASE value_kind
        WHEN 'string' THEN 'text'
        WHEN 'boolean' THEN 'bool'
        WHEN 'object' THEN 'json'
        WHEN 'array' THEN 'json'
        ELSE value_kind
    END::text AS property_claim_value_kind,
    COALESCE(CASE WHEN value_kind = 'string' THEN value #>> '{}' ELSE NULL END, '')::text AS property_claim_value_text,
    COALESCE((CASE WHEN value_kind = 'number' THEN (value #>> '{}')::double precision ELSE NULL END)::double precision, 0)::double precision AS property_claim_value_number,
    COALESCE((CASE WHEN value_kind = 'boolean' THEN (value #>> '{}')::boolean ELSE NULL END)::boolean, false)::boolean AS property_claim_value_bool,
    round(confidence * 100)::integer AS property_claim_confidence,
    COALESCE(evidence #>> '{text}', '')::text AS property_claim_evidence_text,
    COALESCE(extraction_model, '')::text AS property_claim_model,
    COALESCE(extraction_prompt_version, '')::text AS property_claim_prompt_version
FROM public.dimension_claims
WHERE target_type = public.fnc__legacy_property_dimension_target_type(sqlc.arg(entity_type))
    AND target_id = sqlc.arg(entity_id)
ORDER BY property_claim_namespace, property_claim_key;

-- name: CreatePropertyDocumentForOffering :one
WITH linked AS (
    SELECT
        po.property_offering_id,
        po.property_unit_id,
        pu.physical_building_id,
        pu.housing_company_id
    FROM public.property_offerings po
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE po.property_offering_id = sqlc.arg(property_offering_id)
)
INSERT INTO public.property_documents (
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_bytes
)
SELECT
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    sqlc.arg(document_type),
    sqlc.arg(filename),
    sqlc.arg(mime_type),
    sqlc.arg(size_bytes),
    sqlc.arg(sha256),
    sqlc.arg(document_bytes)
FROM linked
ON CONFLICT (property_offering_id, property_document_type, property_document_sha256) DO UPDATE SET
    property_document_filename = EXCLUDED.property_document_filename,
    property_document_mime_type = EXCLUDED.property_document_mime_type,
    property_document_size_bytes = EXCLUDED.property_document_size_bytes,
    property_document_bytes = EXCLUDED.property_document_bytes,
    property_document_extraction_status = 'uploaded',
    property_document_extraction_error = NULL,
    property_document_updated_at = now()
RETURNING
    property_document_id,
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_extraction_status,
    property_document_extraction_error,
    property_document_uploaded_at,
    property_document_extracted_at;

-- name: CreateDetachedPropertyDocument :one
INSERT INTO public.property_documents (
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_bytes
) VALUES (
    sqlc.arg(document_type),
    sqlc.arg(filename),
    sqlc.arg(mime_type),
    sqlc.arg(size_bytes),
    sqlc.arg(sha256),
    sqlc.arg(document_bytes)
)
ON CONFLICT (property_document_type, property_document_sha256) WHERE property_offering_id IS NULL DO UPDATE SET
    property_document_filename = EXCLUDED.property_document_filename,
    property_document_mime_type = EXCLUDED.property_document_mime_type,
    property_document_size_bytes = EXCLUDED.property_document_size_bytes,
    property_document_bytes = EXCLUDED.property_document_bytes,
    property_document_extraction_status = 'uploaded',
    property_document_extraction_error = NULL,
    property_document_updated_at = now()
RETURNING
    property_document_id,
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_extraction_status,
    property_document_extraction_error,
    property_document_uploaded_at,
    property_document_extracted_at;

-- name: AttachPropertyDocumentToOffering :one
WITH relinked AS (
    SELECT public.fnc__relink_property_document_offering(
        sqlc.arg(property_document_id)::uuid,
        sqlc.arg(property_offering_id)::uuid,
        sqlc.arg(reason)::text
    ) AS result
)
SELECT
    property_document_id,
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_extraction_status,
    property_document_extraction_error,
    property_document_uploaded_at,
    property_document_extracted_at
FROM public.property_documents
JOIN relinked ON true
WHERE property_document_id = sqlc.arg(property_document_id);

-- name: EnsureManagerCertificateHousingCompany :one
INSERT INTO public.housing_companies (
    housing_company_identity_key,
    housing_company_name,
    housing_company_business_id,
    housing_company_build_year,
    housing_company_apartment_count,
    housing_company_energy_efficiency_label,
    housing_company_match_reasons
) VALUES (
    sqlc.arg(identity_key),
    sqlc.narg(name),
    sqlc.narg(business_id),
    sqlc.narg(build_year),
    sqlc.narg(apartment_count),
    sqlc.narg(energy_class),
    jsonb_build_object('source', 'manager_certificate', 'property_document_id', sqlc.arg(property_document_id)::text)
)
ON CONFLICT (housing_company_identity_key) DO UPDATE SET
    housing_company_name = COALESCE(EXCLUDED.housing_company_name, public.housing_companies.housing_company_name),
    housing_company_business_id = COALESCE(EXCLUDED.housing_company_business_id, public.housing_companies.housing_company_business_id),
    housing_company_build_year = COALESCE(EXCLUDED.housing_company_build_year, public.housing_companies.housing_company_build_year),
    housing_company_apartment_count = COALESCE(EXCLUDED.housing_company_apartment_count, public.housing_companies.housing_company_apartment_count),
    housing_company_energy_efficiency_label = COALESCE(EXCLUDED.housing_company_energy_efficiency_label, public.housing_companies.housing_company_energy_efficiency_label),
    housing_company_updated_at = now()
RETURNING housing_company_id;

-- name: EnsureManagerCertificatePhysicalBuilding :one
INSERT INTO public.physical_buildings (
    housing_company_id,
    physical_building_identity_key,
    physical_building_build_year,
    physical_building_floor_count,
    physical_building_apartment_count,
    physical_building_elevator
) VALUES (
    sqlc.arg(housing_company_id),
    sqlc.arg(identity_key),
    sqlc.narg(build_year),
    sqlc.narg(floor_count),
    sqlc.narg(apartment_count),
    sqlc.narg(elevator)
)
ON CONFLICT (physical_building_identity_key) DO UPDATE SET
    housing_company_id = COALESCE(EXCLUDED.housing_company_id, public.physical_buildings.housing_company_id),
    physical_building_build_year = COALESCE(EXCLUDED.physical_building_build_year, public.physical_buildings.physical_building_build_year),
    physical_building_floor_count = COALESCE(EXCLUDED.physical_building_floor_count, public.physical_buildings.physical_building_floor_count),
    physical_building_apartment_count = COALESCE(EXCLUDED.physical_building_apartment_count, public.physical_buildings.physical_building_apartment_count),
    physical_building_elevator = COALESCE(EXCLUDED.physical_building_elevator, public.physical_buildings.physical_building_elevator),
    physical_building_updated_at = now()
RETURNING physical_building_id;

-- name: EnsureManagerCertificatePropertyUnit :one
INSERT INTO public.property_units (
    housing_company_id,
    physical_building_id,
    property_unit_identity_key,
    property_unit_floor_level,
    property_unit_area_value,
    property_unit_rooms_count,
    property_unit_room_layout,
    property_unit_layout_match_key,
    property_unit_match_reasons
) VALUES (
    sqlc.arg(housing_company_id),
    sqlc.arg(physical_building_id),
    sqlc.arg(identity_key),
    sqlc.narg(floor_level),
    sqlc.narg(area_m2),
    sqlc.narg(rooms_count),
    sqlc.narg(room_layout),
    sqlc.narg(layout_match_key),
    jsonb_build_object('source', 'manager_certificate', 'property_document_id', sqlc.arg(property_document_id)::text)
)
ON CONFLICT (property_unit_identity_key) DO UPDATE SET
    housing_company_id = EXCLUDED.housing_company_id,
    physical_building_id = EXCLUDED.physical_building_id,
    property_unit_floor_level = COALESCE(EXCLUDED.property_unit_floor_level, public.property_units.property_unit_floor_level),
    property_unit_area_value = COALESCE(EXCLUDED.property_unit_area_value, public.property_units.property_unit_area_value),
    property_unit_rooms_count = COALESCE(EXCLUDED.property_unit_rooms_count, public.property_units.property_unit_rooms_count),
    property_unit_room_layout = COALESCE(EXCLUDED.property_unit_room_layout, public.property_units.property_unit_room_layout),
    property_unit_layout_match_key = COALESCE(EXCLUDED.property_unit_layout_match_key, public.property_units.property_unit_layout_match_key),
    property_unit_updated_at = now()
RETURNING property_unit_id;

-- name: SyncUnitFromPropertyUnit :exec
INSERT INTO public.units (
    unit_id,
    housing_company_id,
    physical_building_id,
    identity_key,
    address_norm,
    apartment,
    floor_level,
    area_m2,
    room_layout,
    created_at,
    updated_at
)
SELECT
    property_unit_id,
    housing_company_id,
    physical_building_id,
    property_unit_identity_key,
    property_unit_address_norm,
    NULL::text,
    property_unit_floor_level,
    property_unit_area_value,
    property_unit_room_layout,
    property_unit_created_at,
    property_unit_updated_at
FROM public.property_units
WHERE property_unit_id = sqlc.arg(property_unit_id)
ON CONFLICT (unit_id) DO UPDATE SET
    housing_company_id = EXCLUDED.housing_company_id,
    physical_building_id = EXCLUDED.physical_building_id,
    identity_key = EXCLUDED.identity_key,
    address_norm = EXCLUDED.address_norm,
    apartment = EXCLUDED.apartment,
    floor_level = EXCLUDED.floor_level,
    area_m2 = EXCLUDED.area_m2,
    room_layout = EXCLUDED.room_layout,
    updated_at = EXCLUDED.updated_at;

-- name: EnsureManagerCertificatePropertyOffering :one
INSERT INTO public.property_offerings (
    property_unit_id,
    property_offering_identity_key,
    property_offering_type,
    property_offering_headline,
    property_offering_match_reasons
) VALUES (
    sqlc.arg(property_unit_id),
    sqlc.arg(identity_key),
    'sale',
    sqlc.arg(headline),
    jsonb_build_object('source', 'manager_certificate', 'property_document_id', sqlc.arg(property_document_id)::text)
)
ON CONFLICT (property_offering_identity_key) DO UPDATE SET
    property_unit_id = EXCLUDED.property_unit_id,
    property_offering_headline = EXCLUDED.property_offering_headline,
    property_offering_updated_at = now()
RETURNING property_offering_id;

-- name: SyncListingFromPropertyOffering :exec
INSERT INTO public.listings (
    listing_id,
    listing_type,
    listing_status,
    primary_source_listing_id,
    unit_id,
    house_id,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    property_offering_id,
    property_offering_type,
    property_offering_status,
    primary_sale_listing_id,
    property_unit_id,
    property_house_id,
    property_offering_first_seen_at,
    property_offering_last_seen_at,
    property_offering_created_at,
    property_offering_updated_at
FROM public.property_offerings
WHERE property_offering_id = sqlc.arg(property_offering_id)
ON CONFLICT (listing_id) DO UPDATE SET
    listing_type = EXCLUDED.listing_type,
    listing_status = EXCLUDED.listing_status,
    primary_source_listing_id = EXCLUDED.primary_source_listing_id,
    unit_id = EXCLUDED.unit_id,
    house_id = EXCLUDED.house_id,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = EXCLUDED.updated_at;

-- name: SyncListingFromSourceListing :exec
INSERT INTO public.listings (
    listing_id,
    listing_type,
    listing_status,
    primary_source_listing_id,
    unit_id,
    house_id,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    po.property_offering_id,
    po.property_offering_type,
    po.property_offering_status,
    po.primary_sale_listing_id,
    po.property_unit_id,
    po.property_house_id,
    po.property_offering_first_seen_at,
    po.property_offering_last_seen_at,
    po.property_offering_created_at,
    po.property_offering_updated_at
FROM public.target_sources source_link
JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
WHERE source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.source_id = sqlc.arg(source_listing_id)
    AND source_link.link_status <> 'rejected'
ON CONFLICT (listing_id) DO UPDATE SET
    listing_type = EXCLUDED.listing_type,
    listing_status = EXCLUDED.listing_status,
    primary_source_listing_id = EXCLUDED.primary_source_listing_id,
    unit_id = EXCLUDED.unit_id,
    house_id = EXCLUDED.house_id,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = EXCLUDED.updated_at;

-- name: SyncHouseFromPropertyHouse :exec
INSERT INTO public.houses (
    house_id,
    identity_key,
    address_norm,
    postal_norm,
    city_norm,
    latitude,
    longitude,
    created_at,
    updated_at
)
SELECT
    property_house_id,
    property_house_identity_key,
    property_house_address_norm,
    property_house_postal_norm,
    property_house_city_norm,
    property_house_latitude,
    property_house_longitude,
    property_house_created_at,
    property_house_updated_at
FROM public.property_houses
WHERE property_house_id = sqlc.arg(property_house_id)
ON CONFLICT (house_id) DO UPDATE SET
    identity_key = EXCLUDED.identity_key,
    address_norm = EXCLUDED.address_norm,
    postal_norm = EXCLUDED.postal_norm,
    city_norm = EXCLUDED.city_norm,
    latitude = EXCLUDED.latitude,
    longitude = EXCLUDED.longitude,
    updated_at = EXCLUDED.updated_at;

-- name: GetPropertyDocumentDownload :one
SELECT
    property_document_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_bytes
FROM public.property_documents
WHERE property_document_id = sqlc.arg(property_document_id)
LIMIT 1;

-- name: GetPropertyDocumentForExtraction :one
SELECT
    property_document_id,
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_bytes
FROM public.property_documents
WHERE property_document_id = sqlc.arg(property_document_id)
LIMIT 1;

-- name: GetPropertyDocumentSummary :one
SELECT
    property_document_id,
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_extraction_status,
    property_document_extraction_error,
    property_document_uploaded_at,
    property_document_extracted_at
FROM public.property_documents
WHERE property_document_id = sqlc.arg(property_document_id)
LIMIT 1;

-- name: ListPropertyDocumentsForOffering :many
SELECT
    property_document_id,
    property_offering_id,
    property_unit_id,
    physical_building_id,
    housing_company_id,
    property_document_type,
    property_document_filename,
    property_document_mime_type,
    property_document_size_bytes,
    property_document_sha256,
    property_document_extraction_status,
    property_document_extraction_error,
    property_document_uploaded_at,
    property_document_extracted_at
FROM public.property_documents
WHERE property_offering_id = sqlc.arg(property_offering_id)
ORDER BY property_document_uploaded_at DESC;

-- name: CreatePropertyDocumentExtractionRun :one
INSERT INTO public.property_document_extraction_runs (
    property_document_id,
    property_document_extraction_run_model,
    property_document_extraction_run_prompt_version,
    property_document_extraction_run_status
) VALUES (
    sqlc.arg(property_document_id),
    sqlc.arg(model),
    sqlc.arg(prompt_version),
    'running'
)
RETURNING property_document_extraction_run_id;

-- name: FinishPropertyDocumentExtractionRun :exec
UPDATE public.property_document_extraction_runs
SET property_document_extraction_run_status = sqlc.arg(status),
    property_document_extraction_run_raw_json = sqlc.narg(raw_json),
    property_document_extraction_run_error = NULLIF(sqlc.arg(error_text), ''),
    property_document_extraction_run_finished_at = now()
WHERE property_document_extraction_run_id = sqlc.arg(property_document_extraction_run_id);

-- name: UpsertPropertyDocumentExtraction :one
WITH superseded AS (
    UPDATE public.property_document_extractions
    SET property_document_extraction_status = 'superseded',
        property_document_extraction_superseded_at = now()
    WHERE property_document_id = sqlc.arg(property_document_id)
        AND property_document_extraction_kind = sqlc.arg(kind)
        AND property_document_extraction_superseded_at IS NULL
    RETURNING property_document_extraction_id
)
INSERT INTO public.property_document_extractions (
    property_document_id,
    property_document_extraction_kind,
    property_document_extraction_schema_version,
    property_document_extraction_model,
    property_document_extraction_prompt_version,
    property_document_extraction_source_json,
    property_document_extraction_status,
    property_document_extraction_error
) VALUES (
    sqlc.arg(property_document_id),
    sqlc.arg(kind),
    sqlc.arg(schema_version),
    sqlc.arg(model),
    sqlc.arg(prompt_version),
    sqlc.arg(source_json),
    'succeeded',
    NULL
)
RETURNING property_document_extraction_id;

-- name: GetLatestPropertyDocumentExtraction :one
SELECT
    property_document_extraction_id,
    property_document_id,
    property_document_extraction_kind,
    property_document_extraction_schema_version,
    property_document_extraction_model,
    property_document_extraction_prompt_version,
    property_document_extraction_source_json,
    property_document_extraction_status,
    property_document_extraction_error,
    property_document_extraction_created_at,
    property_document_extraction_extracted_at,
    property_document_extraction_superseded_at
FROM public.property_document_extractions
WHERE property_document_id = sqlc.arg(property_document_id)
    AND property_document_extraction_kind = sqlc.arg(kind)
    AND property_document_extraction_superseded_at IS NULL
LIMIT 1;

-- name: UpdatePropertyDocumentExtractionStatus :exec
UPDATE public.property_documents
SET property_document_extraction_status = sqlc.arg(status),
    property_document_extraction_error = NULLIF(sqlc.arg(error_text), ''),
    property_document_extracted_at = CASE WHEN sqlc.arg(status) = 'extracted' THEN now() ELSE property_document_extracted_at END,
    property_document_updated_at = now()
WHERE property_document_id = sqlc.arg(property_document_id);

-- name: DeleteLLMPropertyClaimsForDocument :exec
DELETE FROM public.dimension_claims claims
WHERE claims.source_table = 'property_documents'
    AND claims.source_id = sqlc.arg(property_document_id)
    AND claims.extraction_model IS NOT NULL;

-- name: InsertDocumentPropertyClaim :exec
WITH run AS (
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    VALUES (
        'source_claims',
        COALESCE(NULLIF(sqlc.arg(prompt_version), ''), 'document-llm-v1'),
        'property_documents',
        sqlc.arg(property_document_id),
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id
),
payload AS (
    SELECT
        public.fnc__legacy_property_dimension_target_type(sqlc.arg(entity_type)::text) AS target_type,
        public.fnc__legacy_property_dimension_claim_scope(sqlc.arg(entity_type)::text) AS claim_scope,
        public.fnc__legacy_property_dimension_key(sqlc.arg(section)::text, sqlc.arg(key)::text) AS dimension_key,
        public.fnc__legacy_property_dimension_value_kind(sqlc.arg(value_kind)::text, sqlc.narg(value_json)::jsonb) AS value_kind,
        public.fnc__legacy_property_dimension_value(sqlc.arg(value_kind)::text, NULLIF(sqlc.arg(value_text), ''), sqlc.narg(value_number)::double precision, sqlc.narg(value_bool)::boolean, sqlc.narg(value_json)::jsonb) AS value
)
INSERT INTO public.dimension_claims (
    property_dimension_projection_run_id,
    projection_version,
    claim_scope,
    target_type,
    target_id,
    dimension_key,
    value,
    value_kind,
    unit,
    source_table,
    source_id,
    source_field,
    source_observed_at,
    confidence,
    source_reliability,
    evidence,
    extraction_model,
    extraction_prompt_version
)
SELECT
    run.property_dimension_projection_run_id,
    COALESCE(NULLIF(sqlc.arg(prompt_version), ''), 'document-llm-v1'),
    payload.claim_scope,
    payload.target_type,
    sqlc.arg(entity_id),
    payload.dimension_key,
    payload.value,
    payload.value_kind,
    catalog.unit,
    'property_documents',
    sqlc.arg(property_document_id),
    NULLIF(sqlc.arg(source_field), ''),
    COALESCE(sqlc.narg(source_observed_at)::timestamptz, now()),
    GREATEST(0, LEAST(1, sqlc.arg(confidence)::double precision / 100)),
    0.9,
    jsonb_build_object('text', NULLIF(sqlc.arg(evidence_text), '')),
    NULLIF(sqlc.arg(model), ''),
    NULLIF(sqlc.arg(prompt_version), '')
FROM run
CROSS JOIN payload
LEFT JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = payload.dimension_key
WHERE payload.value IS NOT NULL
ON CONFLICT (
    claim_scope,
    target_type,
    target_id,
    dimension_key,
    source_table,
    source_id,
    COALESCE(source_field, ''),
    projection_version
) DO UPDATE SET
    value = EXCLUDED.value,
    value_kind = EXCLUDED.value_kind,
    unit = EXCLUDED.unit,
    source_observed_at = EXCLUDED.source_observed_at,
    confidence = EXCLUDED.confidence,
    source_reliability = EXCLUDED.source_reliability,
    evidence = EXCLUDED.evidence,
    extraction_model = EXCLUDED.extraction_model,
    extraction_prompt_version = EXCLUDED.extraction_prompt_version,
    updated_at = now();

-- name: DeleteLLMPropertySourceOfferingInsights :exec
UPDATE public.target_observations
SET superseded_at = now()
WHERE source_type = 'source_listing'
    AND source_id = sqlc.arg(sale_listing_id)
    AND superseded_at IS NULL
    AND evidence ->> 'source_field' LIKE 'llm_%';

-- name: DeleteLLMPropertyClaimsForEntity :exec
DELETE FROM public.dimension_claims claims
WHERE claims.target_type = public.fnc__legacy_property_dimension_target_type(sqlc.arg(entity_type)::text)
    AND claims.target_id = sqlc.arg(entity_id)
    AND claims.extraction_model IS NOT NULL;

-- name: InsertPropertySourceOfferingInsight :exec
INSERT INTO public.target_observations (
    target_type,
    target_id,
    observation_key,
    observation_kind,
    severity,
    direction,
    value,
    text,
    confidence,
    source_type,
    source_id,
    evidence
)
SELECT
    source_link.target_type,
    source_link.target_id,
    sqlc.arg(key),
    CASE
        WHEN sqlc.arg(direction)::text = 'negative' THEN 'risk'
        WHEN sqlc.arg(direction)::text = 'positive' THEN 'opportunity'
        ELSE 'summary'
    END,
    sqlc.arg(severity),
    sqlc.arg(direction),
    to_jsonb(sqlc.arg(value)::text),
    NULLIF(sqlc.arg(text), ''),
    sqlc.arg(confidence)::integer::double precision / 100,
    'source_listing',
    source_link.source_id,
    jsonb_build_object('source_field', sqlc.arg(source_field)::text)
FROM public.target_sources source_link
WHERE source_link.target_type = 'listing'
    AND source_link.source_type = 'source_listing'
    AND source_link.source_id = sqlc.arg(sale_listing_id)::uuid
    AND source_link.link_status <> 'rejected'
ON CONFLICT (target_type, target_id, observation_key, source_type, source_id) WHERE superseded_at IS NULL DO UPDATE SET
    observation_kind = EXCLUDED.observation_kind,
    severity = EXCLUDED.severity,
    direction = EXCLUDED.direction,
    value = EXCLUDED.value,
    text = EXCLUDED.text,
    confidence = EXCLUDED.confidence,
    evidence = EXCLUDED.evidence;

-- name: InsertPropertyClaim :exec
WITH run AS (
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    VALUES (
        'source_claims',
        COALESCE(NULLIF(sqlc.arg(prompt_version), ''), 'listing-llm-v1'),
        'property_source_offerings',
        sqlc.arg(entity_id),
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id
),
payload AS (
    SELECT
        public.fnc__legacy_property_dimension_target_type(sqlc.arg(entity_type)::text) AS target_type,
        public.fnc__legacy_property_dimension_claim_scope(sqlc.arg(entity_type)::text) AS claim_scope,
        public.fnc__legacy_property_dimension_key(sqlc.arg(section)::text, sqlc.arg(key)::text) AS dimension_key,
        public.fnc__legacy_property_dimension_value_kind(sqlc.arg(value_kind)::text, NULL::jsonb) AS value_kind,
        public.fnc__legacy_property_dimension_value(sqlc.arg(value_kind)::text, NULLIF(sqlc.arg(value_text), ''), sqlc.narg(value_number)::double precision, sqlc.narg(value_bool)::boolean, NULL::jsonb) AS value
)
INSERT INTO public.dimension_claims (
    property_dimension_projection_run_id,
    projection_version,
    claim_scope,
    target_type,
    target_id,
    dimension_key,
    value,
    value_kind,
    unit,
    source_table,
    source_id,
    source_field,
    source_observed_at,
    confidence,
    source_reliability,
    evidence,
    extraction_model,
    extraction_prompt_version
)
SELECT
    run.property_dimension_projection_run_id,
    COALESCE(NULLIF(sqlc.arg(prompt_version), ''), 'listing-llm-v1'),
    payload.claim_scope,
    payload.target_type,
    sqlc.arg(entity_id),
    payload.dimension_key,
    payload.value,
    payload.value_kind,
    catalog.unit,
    'property_source_offerings',
    sqlc.arg(entity_id),
    NULLIF(sqlc.arg(source_field), ''),
    now(),
    GREATEST(0, LEAST(1, sqlc.arg(confidence)::double precision / 100)),
    0.65,
    jsonb_build_object('text', NULLIF(sqlc.arg(evidence_text), '')),
    NULLIF(sqlc.arg(model), ''),
    NULLIF(sqlc.arg(prompt_version), '')
FROM run
CROSS JOIN payload
LEFT JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = payload.dimension_key
WHERE payload.value IS NOT NULL
ON CONFLICT (
    claim_scope,
    target_type,
    target_id,
    dimension_key,
    source_table,
    source_id,
    COALESCE(source_field, ''),
    projection_version
) DO UPDATE SET
    value = EXCLUDED.value,
    value_kind = EXCLUDED.value_kind,
    unit = EXCLUDED.unit,
    source_observed_at = EXCLUDED.source_observed_at,
    confidence = EXCLUDED.confidence,
    source_reliability = EXCLUDED.source_reliability,
    evidence = EXCLUDED.evidence,
    extraction_model = EXCLUDED.extraction_model,
    extraction_prompt_version = EXCLUDED.extraction_prompt_version,
    updated_at = now();

-- name: SearchUnifiedEntities :many
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        sa.shortcut_ad_id::text AS native_id,
        ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id,
        COALESCE(linked.sale_listing_id::text, '') AS listing_id,
        COALESCE(linked.property_offering_id::text, '') AS offering_id,
        linked.latitude,
        linked.longitude,
        COALESCE(linked.link_status, '') AS link_status,
        COALESCE(linked.link_method, '') AS link_method,
        linked.link_score::int4 AS link_score,
        (sa.shortcut_ad_url IS NOT NULL AND sa.shortcut_ad_url <> '' AND sa.shortcut_ad_last_seen_at >= now() - interval '7 days') AS external_url_available,
        COALESCE(raw.street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text)::text AS headline,
        COALESCE(raw.street_address, sb.shortcut_building_address)::text AS address,
        raw.city::text AS city,
        raw.postal::text AS postal,
        raw.price::bigint AS price,
        COALESCE(raw.area, 0::float8)::float8 AS area,
        sa.shortcut_ad_data #>> '{adData,roomConfiguration}' AS room_layout,
        sa.shortcut_ad_url AS url,
        sa.shortcut_ad_last_seen_at AS last_seen_at,
        trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable,
        sa.shortcut_ad_type AS listing_type,
        (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN LATERAL (
        SELECT
            sl.sale_listing_id,
            source_link.target_id AS property_offering_id,
            sl.sale_listing_latitude AS latitude,
            sl.sale_listing_longitude AS longitude,
            source_link.link_status,
            source_link.link_method,
            source_link.link_score
        FROM public.property_source_offerings sl
        LEFT JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
            AND source_link.target_type = 'listing'
            AND source_link.source_type = 'source_listing'
            AND source_link.link_status <> 'rejected'
        WHERE sl.shortcut_ad_id = sa.shortcut_ad_id
        ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC NULLS LAST
        LIMIT 1
    ) linked ON true
    CROSS JOIN LATERAL (
        SELECT
            public.fnc__shortcut_ad_street_address(sa.shortcut_ad_data) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerDay}')) AS price,
            COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')) AS area
    ) raw
    UNION ALL
    SELECT
        'shortcut'::text AS source,
        'building'::text AS kind,
        sb.shortcut_building_id::text AS native_id,
        ('shortcut:building:' || sb.shortcut_building_id::text) AS canonical_id,
        ''::text AS listing_id,
        ''::text AS offering_id,
        sb.shortcut_building_latitude AS latitude,
        sb.shortcut_building_longitude AS longitude,
        ''::text AS link_status,
        ''::text AS link_method,
        NULL::int4 AS link_score,
        (sb.shortcut_building_url IS NOT NULL AND sb.shortcut_building_url <> '') AS external_url_available,
        COALESCE(sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_external_id::text) AS headline,
        sb.shortcut_building_address AS address,
        NULL::text AS city,
        NULL::text AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        NULL::text AS room_layout,
        sb.shortcut_building_url AS url,
        COALESCE(sb.shortcut_building_updated_at, sb.shortcut_building_processed_at, now()) AS last_seen_at,
        concat_ws(' ', sb.shortcut_building_id::text, sb.shortcut_building_external_id::text, sb.shortcut_building_url, sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_building_type, sb.shortcut_building_building_subtype) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        fa.frontdoor_ad_external_id AS native_id,
        sl.sale_listing_id::text AS canonical_id,
        sl.sale_listing_id::text AS listing_id,
        COALESCE(source_link.target_id::text, '') AS offering_id,
        sl.sale_listing_latitude AS latitude,
        sl.sale_listing_longitude AS longitude,
        COALESCE(source_link.link_status, '') AS link_status,
        COALESCE(source_link.link_method, '') AS link_method,
        source_link.link_score::int4 AS link_score,
        (fa.frontdoor_ad_page_not_found = false) AS external_url_available,
        COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, fa.frontdoor_ad_external_id) AS headline,
        sl.sale_listing_street_address AS address,
        sl.sale_listing_city AS city,
        sl.sale_listing_postal AS postal,
        sl.sale_listing_asking_price AS price,
        COALESCE(sl.sale_listing_area_value, 0::float8) AS area,
        sl.sale_listing_room_layout AS room_layout,
        COALESCE(sl.sale_listing_url, fa.frontdoor_ad_url) AS url,
        sl.sale_listing_last_seen_at AS last_seen_at,
        sl.sale_listing_search_text AS searchable,
        NULL::text AS listing_type,
        sl.sale_listing_published_at AS published_at
    FROM public.frontdoor_ads fa
    JOIN public.property_source_offerings sl ON sl.frontdoor_ad_id = fa.frontdoor_ad_id
    LEFT JOIN LATERAL (
        SELECT
            source_link.target_id,
            source_link.link_status,
            source_link.link_method,
            source_link.link_score
        FROM public.target_sources source_link
        WHERE source_link.target_type = 'listing'
            AND source_link.source_type = 'source_listing'
            AND source_link.source_id = sl.sale_listing_id
            AND source_link.link_status <> 'rejected'
        ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC NULLS LAST
        LIMIT 1
    ) source_link ON true
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        sl.sale_listing_native_id AS native_id,
        sl.sale_listing_id::text AS canonical_id,
        sl.sale_listing_id::text AS listing_id,
        COALESCE(source_link.target_id::text, '') AS offering_id,
        sl.sale_listing_latitude AS latitude,
        sl.sale_listing_longitude AS longitude,
        COALESCE(source_link.link_status, '') AS link_status,
        COALESCE(source_link.link_method, '') AS link_method,
        source_link.link_score::int4 AS link_score,
        COALESCE(fba.frontdoor_building_announcement_published, false) AS external_url_available,
        COALESCE(sl.sale_listing_headline, sl.sale_listing_street_address, sl.sale_listing_native_id) AS headline,
        sl.sale_listing_street_address AS address,
        sl.sale_listing_city AS city,
        sl.sale_listing_postal AS postal,
        sl.sale_listing_asking_price AS price,
        COALESCE(sl.sale_listing_area_value, 0::float8) AS area,
        sl.sale_listing_room_layout AS room_layout,
        sl.sale_listing_url AS url,
        sl.sale_listing_last_seen_at AS last_seen_at,
        sl.sale_listing_search_text AS searchable,
        NULL::text AS listing_type,
        sl.sale_listing_published_at AS published_at
    FROM public.property_source_offerings sl
    LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    LEFT JOIN LATERAL (
        SELECT
            source_link.target_id,
            source_link.link_status,
            source_link.link_method,
            source_link.link_score
        FROM public.target_sources source_link
        WHERE source_link.target_type = 'listing'
            AND source_link.source_type = 'source_listing'
            AND source_link.source_id = sl.sale_listing_id
            AND source_link.link_status <> 'rejected'
        ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC NULLS LAST
        LIMIT 1
    ) source_link ON true
    WHERE sl.frontdoor_building_announcement_id IS NOT NULL
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'building'::text AS kind,
        fb.frontdoor_building_id::text AS native_id,
        ('frontdoor:building:' || fb.frontdoor_building_id::text) AS canonical_id,
        ''::text AS listing_id,
        ''::text AS offering_id,
        fb.frontdoor_building_latitude AS latitude,
        fb.frontdoor_building_longitude AS longitude,
        ''::text AS link_status,
        ''::text AS link_method,
        NULL::int4 AS link_score,
        (fb.frontdoor_building_url IS NOT NULL AND fb.frontdoor_building_url <> '') AS external_url_available,
        COALESCE(fb.frontdoor_building_company_name, concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number), fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_id::text) AS headline,
        concat_ws(' ', fb.frontdoor_building_street_address, fb.frontdoor_building_house_number) AS address,
        COALESCE(fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        NULL::text AS room_layout,
        fb.frontdoor_building_url AS url,
        COALESCE(fb.frontdoor_building_last_seen_at, now()) AS last_seen_at,
        concat_ws(' ', fb.frontdoor_building_id::text, fb.frontdoor_building_url, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_company_name, fb.frontdoor_building_street_address, fb.frontdoor_building_house_number, fb.frontdoor_building_postcode, fb.frontdoor_building_post_area, fb.frontdoor_building_municipality) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.frontdoor_buildings fb
), filtered AS (
    SELECT *
    FROM unified u
    WHERE (sqlc.arg(source_filter) = 'all' OR u.source = sqlc.arg(source_filter))
      AND (sqlc.arg(kind_filter) = 'all' OR u.kind = sqlc.arg(kind_filter))
      AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
      AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
      AND (sqlc.narg('postal_filter')::text IS NULL OR trim(sqlc.narg('postal_filter')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal_filter')::text)) || '%'))
      AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
      AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
      AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
      AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
      AND (sqlc.narg('listing_type_filter')::text IS NULL OR u.listing_type IS NULL OR u.listing_type = sqlc.narg('listing_type_filter')::text)
      AND (sqlc.arg(grouping_filter) = 'all' OR (sqlc.arg(grouping_filter) = 'grouped' AND u.offering_id <> '') OR (sqlc.arg(grouping_filter) = 'ungrouped' AND u.offering_id = ''))
      AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
      AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz)
)
SELECT
    u.source,
    u.kind,
    u.native_id,
    u.canonical_id::text AS canonical_id,
    u.listing_id::text AS listing_id,
    u.offering_id::text AS offering_id,
    u.latitude,
    u.longitude,
    COALESCE(hc.housing_company_id::text, '')::text AS housing_company_id,
    COALESCE(hc.housing_company_name, '') AS housing_company_name,
    u.link_status,
    u.link_method,
    COALESCE(u.link_score, 0)::int4 AS link_score,
    COALESCE(u.external_url_available, false)::bool AS external_url_available,
    COALESCE(price_match.transaction_id::text, '')::text AS price_match_transaction_id,
    COALESCE(price_match.match_scope, '') AS price_match_scope,
    COALESCE(price_match.match_status, '') AS price_match_status,
    COALESCE(price_match.match_method, '') AS price_match_method,
    COALESCE(price_match.match_score, 0)::int4 AS price_match_score,
    COALESCE(price_match.price_eur, 0)::bigint AS price_match_price_eur,
    COALESCE(insight_stats.insight_count, 0)::int4 AS insight_count,
    COALESCE(insight_stats.top_severity, '') AS insight_top_severity,
    u.headline,
    u.address,
    u.city,
    u.postal,
    u.price,
    u.area,
    u.room_layout::text AS room_layout,
    u.url,
    u.last_seen_at
FROM filtered u
LEFT JOIN public.property_offerings po ON po.property_offering_id::text = u.offering_id
LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
LEFT JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
LEFT JOIN LATERAL (
    SELECT
        pt.prices_transaction_id AS transaction_id,
        price_link.target_type AS match_scope,
        price_link.link_status AS match_status,
        price_link.link_method AS match_method,
        price_link.link_score::int4 AS match_score,
        pt.prices_transaction_price AS price_eur
    FROM public.price_links price_link
    JOIN public.prices_transactions pt ON pt.prices_transaction_id = price_link.prices_transaction_id
    WHERE price_link.link_status <> 'rejected'
        AND (
            (price_link.target_type = 'source_listing' AND price_link.target_id::text = u.listing_id)
            OR (price_link.target_type = 'listing' AND price_link.target_id::text = u.offering_id)
        )
    ORDER BY CASE WHEN price_link.target_type = 'source_listing' THEN 0 ELSE 1 END, price_link.link_score DESC NULLS LAST, pt.prices_transaction_updated_at DESC
    LIMIT 1
) price_match ON true
LEFT JOIN LATERAL (
    SELECT
        count(*)::int4 AS insight_count,
        max(observation.severity)::text AS top_severity
    FROM public.target_observations observation
    WHERE observation.source_type = 'source_listing'
        AND observation.source_id::text = u.listing_id
        AND observation.superseded_at IS NULL
) insight_stats ON true
ORDER BY
    CASE WHEN sqlc.arg(sort_mode) = 'price_asc' THEN u.price END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'price_desc' THEN u.price END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'area_asc' THEN u.area END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'area_desc' THEN u.area END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'seen_desc' THEN u.last_seen_at END DESC NULLS LAST,
    u.last_seen_at DESC,
    u.source,
    u.kind,
    u.native_id
LIMIT sqlc.arg(limit_count)::int
OFFSET sqlc.arg(offset_count)::int;

-- name: CountUnifiedEntities :one
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        COALESCE(linked.property_offering_id::text, '') AS offering_id,
        raw.city::text AS city,
        raw.postal::text AS postal,
        raw.price::bigint AS price,
        COALESCE(raw.area, 0::float8)::float8 AS area,
        trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable,
        sa.shortcut_ad_type AS listing_type,
        (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
    LEFT JOIN LATERAL (
        SELECT source_link.target_id AS property_offering_id
        FROM public.property_source_offerings sl
        JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
            AND source_link.target_type = 'listing'
            AND source_link.source_type = 'source_listing'
            AND source_link.link_status <> 'rejected'
        WHERE sl.shortcut_ad_id = sa.shortcut_ad_id
        ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC NULLS LAST
        LIMIT 1
    ) linked ON true
    CROSS JOIN LATERAL (
        SELECT
            public.fnc__shortcut_ad_street_address(sa.shortcut_ad_data) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerDay}')) AS price,
            COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')) AS area
    ) raw
    UNION ALL
    SELECT
        'shortcut'::text AS source,
        'building'::text AS kind,
        ''::text AS offering_id,
        NULL::text AS city,
        NULL::text AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        concat_ws(' ', sb.shortcut_building_id::text, sb.shortcut_building_external_id::text, sb.shortcut_building_url, sb.shortcut_building_address, sb.shortcut_building_housing_company, sb.shortcut_building_building_type, sb.shortcut_building_building_subtype) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.shortcut_buildings sb
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'ad'::text AS kind,
        COALESCE(source_link.target_id::text, '') AS offering_id,
        sl.sale_listing_city AS city,
        sl.sale_listing_postal AS postal,
        sl.sale_listing_asking_price AS price,
        COALESCE(sl.sale_listing_area_value, 0::float8) AS area,
        sl.sale_listing_search_text AS searchable,
        NULL::text AS listing_type,
        sl.sale_listing_published_at AS published_at
    FROM public.frontdoor_ads fa
    JOIN public.property_source_offerings sl ON sl.frontdoor_ad_id = fa.frontdoor_ad_id
    LEFT JOIN LATERAL (
        SELECT source_link.target_id
        FROM public.target_sources source_link
        WHERE source_link.target_type = 'listing'
            AND source_link.source_type = 'source_listing'
            AND source_link.source_id = sl.sale_listing_id
            AND source_link.link_status <> 'rejected'
        ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC NULLS LAST
        LIMIT 1
    ) source_link ON true
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        COALESCE(source_link.target_id::text, '') AS offering_id,
        sl.sale_listing_city AS city,
        sl.sale_listing_postal AS postal,
        sl.sale_listing_asking_price AS price,
        COALESCE(sl.sale_listing_area_value, 0::float8) AS area,
        sl.sale_listing_search_text AS searchable,
        NULL::text AS listing_type,
        sl.sale_listing_published_at AS published_at
    FROM public.property_source_offerings sl
    LEFT JOIN LATERAL (
        SELECT source_link.target_id
        FROM public.target_sources source_link
        WHERE source_link.target_type = 'listing'
            AND source_link.source_type = 'source_listing'
            AND source_link.source_id = sl.sale_listing_id
            AND source_link.link_status <> 'rejected'
        ORDER BY source_link.link_score DESC NULLS LAST, source_link.updated_at DESC NULLS LAST
        LIMIT 1
    ) source_link ON true
    WHERE sl.frontdoor_building_announcement_id IS NOT NULL
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'building'::text AS kind,
        ''::text AS offering_id,
        COALESCE(fb.frontdoor_building_municipality, fb.frontdoor_building_post_area) AS city,
        fb.frontdoor_building_postcode AS postal,
        NULL::bigint AS price,
        0::float8 AS area,
        concat_ws(' ', fb.frontdoor_building_id::text, fb.frontdoor_building_url, fb.frontdoor_building_housing_company_id::text, fb.frontdoor_building_housing_company_friendly_id, fb.frontdoor_building_company_name, fb.frontdoor_building_street_address, fb.frontdoor_building_house_number, fb.frontdoor_building_postcode, fb.frontdoor_building_post_area, fb.frontdoor_building_municipality) AS searchable,
        NULL::text AS listing_type,
        NULL::timestamptz AS published_at
    FROM public.frontdoor_buildings fb
)
SELECT COUNT(*)::bigint AS count
FROM unified u
WHERE (sqlc.arg(source_filter) = 'all' OR u.source = sqlc.arg(source_filter))
  AND (sqlc.arg(kind_filter) = 'all' OR u.kind = sqlc.arg(kind_filter))
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
  AND (sqlc.narg('postal_filter')::text IS NULL OR trim(sqlc.narg('postal_filter')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal_filter')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('listing_type_filter')::text IS NULL OR u.listing_type IS NULL OR u.listing_type = sqlc.narg('listing_type_filter')::text)
  AND (sqlc.arg(grouping_filter) = 'all' OR (sqlc.arg(grouping_filter) = 'grouped' AND u.offering_id <> '') OR (sqlc.arg(grouping_filter) = 'ungrouped' AND u.offering_id = ''))
  AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
  AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz);

-- name: FindCrossSourceAdMatches :many
SELECT
    sa.shortcut_ad_id,
    fa.frontdoor_ad_external_id,
    ssl.sale_listing_unit_match_key AS address_key,
    ssl.sale_listing_street_address AS shortcut_street,
    fsl.sale_listing_street_address AS frontdoor_street,
    ssl.sale_listing_postal AS shortcut_postal,
    fsl.sale_listing_postal AS frontdoor_postal,
    ssl.sale_listing_city AS shortcut_city,
    fsl.sale_listing_city AS frontdoor_city,
    ssl.sale_listing_asking_price AS shortcut_price,
    fsl.sale_listing_asking_price AS frontdoor_price,
    ssl.sale_listing_area_value AS shortcut_area,
    fsl.sale_listing_area_value AS frontdoor_area
FROM public.property_source_offerings ssl
JOIN public.property_source_offerings fsl ON fsl.sale_listing_unit_match_key = ssl.sale_listing_unit_match_key
JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = ssl.shortcut_ad_id
JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = fsl.frontdoor_ad_id
WHERE ssl.sale_listing_source_provider = 'shortcut'
  AND fsl.sale_listing_source_provider = 'frontdoor'
  AND ssl.sale_listing_source_kind = 'ad'
  AND fsl.sale_listing_source_kind = 'ad'
  AND ssl.sale_listing_unit_match_key IS NOT NULL
  AND ssl.sale_listing_unit_match_key <> ''
  AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(ssl.sale_listing_city, fsl.sale_listing_city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
  AND (
      sqlc.narg('max_price_delta')::bigint IS NULL
      OR (
          ssl.sale_listing_asking_price IS NOT NULL
          AND fsl.sale_listing_asking_price IS NOT NULL
          AND abs(ssl.sale_listing_asking_price - fsl.sale_listing_asking_price) <= sqlc.narg('max_price_delta')::bigint
      )
  )
  AND (
      sqlc.narg('max_area_delta')::float8 IS NULL
      OR (
          ssl.sale_listing_area_value IS NOT NULL
          AND fsl.sale_listing_area_value IS NOT NULL
          AND abs(ssl.sale_listing_area_value - fsl.sale_listing_area_value) <= sqlc.narg('max_area_delta')::float8
      )
  )
ORDER BY
    abs(COALESCE(ssl.sale_listing_asking_price, 0) - COALESCE(fsl.sale_listing_asking_price, 0)) ASC,
    abs(COALESCE(ssl.sale_listing_area_value, 0) - COALESCE(fsl.sale_listing_area_value, 0)) ASC,
    ssl.sale_listing_last_seen_at DESC,
    fsl.sale_listing_last_seen_at DESC
LIMIT sqlc.arg(limit_count)::int;

-- name: GetShortcutAdUnifiedDetail :one
SELECT
    sa.shortcut_ad_id,
    sa.shortcut_ad_url,
    sa.shortcut_ad_type,
    sa.shortcut_ad_last_seen_at,
    sa.shortcut_building_id,
    COALESCE(sl.sale_listing_street_address, public.fnc__shortcut_ad_street_address(sa.shortcut_ad_data), sb.shortcut_building_address) AS ad_address,
    COALESCE(sl.sale_listing_city, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '')) AS ad_city,
    COALESCE(sl.sale_listing_postal, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '')) AS ad_postal,
    COALESCE(sl.sale_listing_latitude, sb.shortcut_building_latitude) AS ad_latitude,
    COALESCE(sl.sale_listing_longitude, sb.shortcut_building_longitude) AS ad_longitude,
    COALESCE(sl.sale_listing_room_layout, sa.shortcut_ad_data #>> '{adData,roomConfiguration}')::text AS ad_room_layout,
    COALESCE(sl.sale_listing_asking_price, COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,rentPerDay}'))) AS ad_price,
    COALESCE(sl.sale_listing_area_value, COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')), 0::float8) AS ad_area,
    COALESCE(sl.sale_listing_description_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,description}', sa.shortcut_ad_data #>> '{description}', sa.shortcut_ad_data #>> '{text}')), '')) AS shortcut_ad_description_text,
    COALESCE(sl.sale_listing_availability_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,availabilityDescription}', sa.shortcut_ad_data #>> '{availabilityDescription}', sa.shortcut_ad_data #>> '{adData,availableFrom}')), '')) AS shortcut_ad_availability_text,
    COALESCE(sl.sale_listing_renovations_done_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{property,renovationsDoneDescription}')), '')) AS shortcut_ad_renovations_done_text,
    COALESCE(sl.sale_listing_renovations_planned_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{property,renovationsPlannedDescription}')), '')) AS shortcut_ad_renovations_planned_text,
    COALESCE(sl.sale_listing_additional_info_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,additionalInfo}', sa.shortcut_ad_data #>> '{moreInformationAvailableFrom}', sa.shortcut_ad_data #>> '{property,otherInfo}')), '')) AS shortcut_ad_additional_info_text,
    COALESCE(sl.sale_listing_charges_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{priceData,chargesText}', sa.shortcut_ad_data #>> '{priceData,additionalInfo}', sa.shortcut_ad_data #>> '{property,periodicChargesAdditionalInfo}', sa.shortcut_ad_data #>> '{property,managementChargesAdditionalInfo}')), '')) AS shortcut_ad_charges_text,
    COALESCE(sl.sale_listing_maintenance_charge_monthly, COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,maintenanceCharge}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,monthlyFee}'))) AS shortcut_ad_maintenance_charge_monthly,
    COALESCE(sl.sale_listing_total_charge_monthly, COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,totalCharge}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,monthlyFee}'))) AS shortcut_ad_total_charge_monthly,
    COALESCE(sl.sale_listing_water_charge, public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,waterFee}')) AS shortcut_ad_water_charge,
    COALESCE(sl.sale_listing_debt_free_price, COALESCE(public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceDebtFree}'), public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,priceSell}'))) AS shortcut_ad_debt_free_price,
    COALESCE(sl.sale_listing_debt_share_amount, public.fnc__try_parse_bigint(sa.shortcut_ad_data #>> '{priceData,debtShare}')) AS shortcut_ad_debt_share_amount,
    COALESCE(sl.sale_listing_price_per_m2, COALESCE(public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,pricePerSqm}'), public.fnc__try_parse_float8(sa.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}'))) AS shortcut_ad_price_per_m2,
    COALESCE(sl.sale_listing_floor_level, COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,floor}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{floor}'))) AS shortcut_ad_floor_level,
    COALESCE(sl.sale_listing_total_floors, COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,totalFloors}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{buildingData,floors}'))) AS shortcut_ad_total_floors,
    COALESCE(sl.sale_listing_build_year, COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{buildingData,year}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,constructionYear}'))) AS shortcut_ad_build_year,
    COALESCE(sl.sale_listing_condition, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,condition}', sa.shortcut_ad_data #>> '{property,condition}')), '')) AS shortcut_ad_condition,
    COALESCE(sl.sale_listing_energy_class, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}')), '')) AS shortcut_ad_energy_class,
    COALESCE(sl.sale_listing_plot_type_raw, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sa.shortcut_ad_data #>> '{adData,buildingOverrideLotOwnership}', sb.shortcut_building_plot_type)), '')) AS shortcut_ad_plot_type,
    COALESCE(sl.sale_listing_elevator, public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,elevator}'), public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,hasElevator}')) AS shortcut_ad_elevator,
    COALESCE(public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,sauna}'), public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,hasSauna}'))::boolean AS shortcut_ad_sauna,
    COALESCE(sl.sale_listing_rooms_count, COALESCE(public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{adData,rooms}'), public.fnc__try_parse_int4(sa.shortcut_ad_data #>> '{rooms}'))) AS shortcut_ad_rooms_count,
    sa.shortcut_ad_data,
    sb.shortcut_building_external_id,
    sb.shortcut_building_url,
    sb.shortcut_building_address,
    sb.shortcut_building_housing_company,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id) AS building_listing_count,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id) AS building_rental_count
FROM public.shortcut_ads sa
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
LEFT JOIN public.property_source_offerings sl ON sl.shortcut_ad_id = sa.shortcut_ad_id
WHERE sa.shortcut_ad_id = sqlc.arg(ad_id)
LIMIT 1;

-- name: GetShortcutBuildingUnifiedDetail :one
SELECT
    sb.shortcut_building_id,
    sb.shortcut_building_external_id,
    sb.shortcut_building_url,
    sb.shortcut_building_address,
    sb.shortcut_building_housing_company,
    sb.shortcut_building_building_type,
    sb.shortcut_building_building_subtype,
    sb.shortcut_building_construction_year,
    sb.shortcut_building_floor_count,
    sb.shortcut_building_apartment_count,
    sb.shortcut_building_heating_system,
    sb.shortcut_building_building_material,
    sb.shortcut_building_plot_type,
    sb.shortcut_building_wall_structure,
    sb.shortcut_building_heat_source,
    sb.shortcut_building_has_elevator,
    sb.shortcut_building_has_sauna,
    sb.shortcut_building_latitude,
    sb.shortcut_building_longitude,
    sb.shortcut_building_updated_at,
    sb.shortcut_building_processed_at,
    sb.shortcut_building_page_not_found,
    (SELECT COUNT(*)::bigint FROM public.shortcut_ads sa WHERE sa.shortcut_building_id = sb.shortcut_building_id) AS ad_count,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id) AS listing_count,
    (SELECT COUNT(*)::bigint FROM public.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id) AS rental_count,
    jsonb_build_object(
        'building_id', sb.shortcut_building_id,
        'external_id', sb.shortcut_building_external_id,
        'address', sb.shortcut_building_address,
        'housing_company', sb.shortcut_building_housing_company,
        'building_type', sb.shortcut_building_building_type,
        'building_subtype', sb.shortcut_building_building_subtype,
        'construction_year', sb.shortcut_building_construction_year,
        'floor_count', sb.shortcut_building_floor_count,
        'apartment_count', sb.shortcut_building_apartment_count,
        'heating_system', sb.shortcut_building_heating_system,
        'building_material', sb.shortcut_building_building_material,
        'plot_type', sb.shortcut_building_plot_type,
        'wall_structure', sb.shortcut_building_wall_structure,
        'heat_source', sb.shortcut_building_heat_source,
        'has_elevator', sb.shortcut_building_has_elevator,
        'has_sauna', sb.shortcut_building_has_sauna,
        'latitude', sb.shortcut_building_latitude,
        'longitude', sb.shortcut_building_longitude,
        'updated_at', sb.shortcut_building_updated_at,
        'processed_at', sb.shortcut_building_processed_at,
        'page_not_found', sb.shortcut_building_page_not_found
    ) AS raw_json
FROM public.shortcut_buildings sb
WHERE sb.shortcut_building_id = sqlc.arg(building_id)
LIMIT 1;

-- name: GetFrontdoorAdUnifiedDetail :one
SELECT
    fa.frontdoor_ad_id,
    fa.frontdoor_ad_external_id,
    fa.frontdoor_ad_url,
    fa.frontdoor_ad_last_seen_at,
    fa.frontdoor_ad_page_not_found,
    sl.sale_listing_street_address AS ad_address,
    sl.sale_listing_city AS ad_city,
    sl.sale_listing_postal AS ad_postal,
    sl.sale_listing_latitude AS ad_latitude,
    sl.sale_listing_longitude AS ad_longitude,
    sl.sale_listing_asking_price AS ad_price,
    COALESCE(sl.sale_listing_area_value, 0::float8) AS ad_area,
    COALESCE(sl.sale_listing_room_layout, '')::text AS ad_room_layout,
    COALESCE(sl.sale_listing_property_type_raw, '')::text AS ad_property_type,
    COALESCE(sl.sale_listing_condition, '')::text AS ad_condition,
    COALESCE(sl.sale_listing_description_text, NULLIF(trim(fa.frontdoor_ad_data #>> '{basicDetails,description}'), '')) AS frontdoor_ad_description_text,
    COALESCE(sl.sale_listing_availability_text, NULLIF(trim(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,freeingDescription}'), '')) AS frontdoor_ad_availability_text,
    COALESCE(sl.sale_listing_renovations_done_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDone}')), '')) AS frontdoor_ad_renovations_done_text,
    COALESCE(sl.sale_listing_renovations_planned_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlanned}')), '')) AS frontdoor_ad_renovations_planned_text,
    COALESCE(sl.sale_listing_additional_info_text, NULLIF(trim(fa.frontdoor_ad_data #>> '{property,additionalInfo}'), '')) AS frontdoor_ad_additional_info_text,
    COALESCE(sl.sale_listing_charges_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,periodicChargesAdditionalInfo}', fa.frontdoor_ad_data #>> '{property,managementChargesAdditionalInfo}')), '')) AS frontdoor_ad_charges_text,
    COALESCE(sl.sale_listing_maintenance_charge_monthly, public.fnc__jsonb_periodic_charge_price(fa.frontdoor_ad_data, 'HOUSING_COMPANY_MAINTENANCE_CHARGE')) AS frontdoor_ad_maintenance_charge_monthly,
    COALESCE(sl.sale_listing_total_charge_monthly, public.fnc__jsonb_periodic_charge_price(fa.frontdoor_ad_data, 'HOUSING_COMPANY_TOTAL_CHARGE')) AS frontdoor_ad_total_charge_monthly,
    COALESCE(sl.sale_listing_water_charge, public.fnc__jsonb_periodic_charge_price(fa.frontdoor_ad_data, 'WATER')) AS frontdoor_ad_water_charge,
    sl.sale_listing_debt_free_price AS frontdoor_ad_debt_free_price,
    sl.sale_listing_debt_share_amount AS frontdoor_ad_debt_share_amount,
    sl.sale_listing_price_per_m2 AS frontdoor_ad_price_per_m2,
    sl.sale_listing_floor_level AS frontdoor_ad_floor_level,
    sl.sale_listing_total_floors AS frontdoor_ad_total_floors,
    sl.sale_listing_build_year AS frontdoor_ad_build_year,
    sl.sale_listing_energy_class AS frontdoor_ad_energy_class,
    sl.sale_listing_plot_type_raw AS frontdoor_ad_plot_type,
    sl.sale_listing_elevator AS frontdoor_ad_elevator,
    COALESCE(sl.sale_listing_sauna, CASE
        WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
        WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
        ELSE NULL
    END, false)::boolean AS frontdoor_ad_sauna,
    sl.sale_listing_rooms_count AS frontdoor_ad_rooms_count,
    fa.frontdoor_ad_data
FROM public.frontdoor_ads fa
LEFT JOIN public.property_source_offerings sl ON sl.frontdoor_ad_id = fa.frontdoor_ad_id
WHERE fa.frontdoor_ad_external_id = sqlc.arg(external_id)
LIMIT 1;

-- name: GetFrontdoorAnnouncementUnifiedDetail :one
SELECT
    fba.frontdoor_building_announcement_id,
    fba.frontdoor_building_announcement_external_id,
    fba.frontdoor_building_announcement_friendly_id,
    fba.frontdoor_building_announcement_last_seen_at,
    fba.frontdoor_building_announcement_address_line1,
    fba.frontdoor_building_announcement_address_line2,
    fba.frontdoor_building_announcement_location,
    fba.frontdoor_building_announcement_search_price,
    fba.frontdoor_building_announcement_area,
    fba.frontdoor_building_announcement_room_structure,
    fba.frontdoor_building_announcement_property_type,
    fba.frontdoor_building_announcement_property_subtype,
    fba.frontdoor_building_announcement_main_image_uri,
    fba.frontdoor_building_announcement_published,
    fba.frontdoor_building_announcement_rent_period,
    fba.frontdoor_building_announcement_rental_unique_no,
    fb.frontdoor_building_id,
    fb.frontdoor_building_url,
    fb.frontdoor_building_housing_company_id,
    fb.frontdoor_building_housing_company_friendly_id,
    fb.frontdoor_building_company_name,
    fb.frontdoor_building_street_address,
    fb.frontdoor_building_house_number,
    fb.frontdoor_building_postcode,
    fb.frontdoor_building_post_area,
    fb.frontdoor_building_municipality,
    fb.frontdoor_building_latitude,
    fb.frontdoor_building_longitude,
    fb.frontdoor_building_energy_certificate_code,
    jsonb_build_object(
        'announcement_id', fba.frontdoor_building_announcement_id,
        'external_id', fba.frontdoor_building_announcement_external_id,
        'friendly_id', fba.frontdoor_building_announcement_friendly_id,
        'address_line1', fba.frontdoor_building_announcement_address_line1,
        'address_line2', fba.frontdoor_building_announcement_address_line2,
        'location', fba.frontdoor_building_announcement_location,
        'search_price', fba.frontdoor_building_announcement_search_price,
        'area', fba.frontdoor_building_announcement_area,
        'room_structure', fba.frontdoor_building_announcement_room_structure,
        'property_type', fba.frontdoor_building_announcement_property_type,
        'property_subtype', fba.frontdoor_building_announcement_property_subtype,
        'published', fba.frontdoor_building_announcement_published,
        'building', jsonb_build_object(
            'building_id', fb.frontdoor_building_id,
            'building_url', fb.frontdoor_building_url,
            'housing_company_id', fb.frontdoor_building_housing_company_id,
            'housing_company_friendly_id', fb.frontdoor_building_housing_company_friendly_id,
            'company_name', fb.frontdoor_building_company_name,
            'street_address', fb.frontdoor_building_street_address,
            'house_number', fb.frontdoor_building_house_number,
            'postcode', fb.frontdoor_building_postcode,
            'post_area', fb.frontdoor_building_post_area,
            'municipality', fb.frontdoor_building_municipality
        )
    ) AS raw_json
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE fba.frontdoor_building_announcement_id = sqlc.arg(announcement_id)
LIMIT 1;

-- name: GetFrontdoorBuildingUnifiedDetail :one
SELECT
    fb.frontdoor_building_id,
    fb.frontdoor_building_url,
    fb.frontdoor_building_last_seen_at,
    fb.frontdoor_building_company_name,
    fb.frontdoor_building_business_id,
    fb.frontdoor_building_apartment_count,
    fb.frontdoor_building_floor_count,
    fb.frontdoor_building_build_year,
    fb.frontdoor_building_has_elevator,
    fb.frontdoor_building_has_sauna,
    fb.frontdoor_building_energy_certificate_code,
    fb.frontdoor_building_heating,
    fb.frontdoor_building_street_address,
    fb.frontdoor_building_house_number,
    fb.frontdoor_building_postcode,
    fb.frontdoor_building_post_area,
    fb.frontdoor_building_municipality,
    fb.frontdoor_building_latitude,
    fb.frontdoor_building_longitude,
    fb.frontdoor_building_housing_company_id,
    fb.frontdoor_building_housing_company_friendly_id,
    (SELECT COUNT(*)::bigint FROM public.frontdoor_building_announcements fba WHERE fba.frontdoor_building_id = fb.frontdoor_building_id) AS announcement_count,
    fb.frontdoor_building_data
FROM public.frontdoor_buildings fb
WHERE fb.frontdoor_building_id = sqlc.arg(building_id)
LIMIT 1;

-- name: GetPropertySourceOfferingDetail :one
SELECT
    sl.sale_listing_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    COALESCE(sl.sale_listing_url, '') AS url,
    COALESCE(sl.sale_listing_headline, '') AS headline,
    COALESCE(sl.sale_listing_street_address, '') AS street_address,
    COALESCE(sl.sale_listing_city, '') AS city,
    COALESCE(sl.sale_listing_postal, '') AS postal,
    sl.sale_listing_latitude,
    sl.sale_listing_longitude,
    COALESCE(sl.sale_listing_room_layout, '') AS room_layout,
    sl.sale_listing_rooms_count,
    sl.sale_listing_area_value,
    sl.sale_listing_floor_level,
    COALESCE(sl.sale_listing_property_type_raw, '') AS property_type_raw,
    COALESCE(sl.sale_listing_condition, '') AS condition,
    sl.sale_listing_elevator,
    COALESCE(sl.sale_listing_energy_class, '') AS energy_class,
    COALESCE(sl.sale_listing_energy_efficiency_label, '') AS energy_efficiency_label,
    COALESCE(sl.sale_listing_plot_type_raw, '') AS plot_type_raw,
    COALESCE(sl.sale_listing_plot_type_code, '') AS plot_type_code,
    sl.sale_listing_plot_owned,
    sl.sale_listing_asking_price,
    sl.sale_listing_debt_free_price,
    sl.sale_listing_debt_share_amount,
    sl.sale_listing_price_per_m2,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_published_at,
    sl.sale_listing_build_year,
    sl.sale_listing_total_floors,
    sl.sale_listing_apartment_count,
    sl.sale_listing_living_area_value,
    sl.sale_listing_total_area_value,
    sl.sale_listing_other_area_value,
    sl.sale_listing_bedrooms_count,
    sl.sale_listing_sauna,
    sl.sale_listing_balcony,
    COALESCE(sl.sale_listing_parking_text, '') AS parking_text,
    COALESCE(sl.sale_listing_kitchen_description_text, '') AS kitchen_description_text,
    COALESCE(sl.sale_listing_bathroom_description_text, '') AS bathroom_description_text,
    COALESCE(sl.sale_listing_storage_description_text, '') AS storage_description_text,
    COALESCE(sl.sale_listing_floor_materials_description_text, '') AS floor_materials_description_text,
    COALESCE(sl.sale_listing_wall_materials_description_text, '') AS wall_materials_description_text,
    COALESCE(sl.sale_listing_balcony_description_text, '') AS balcony_description_text,
    COALESCE(sl.sale_listing_sauna_description_text, '') AS sauna_description_text,
    COALESCE(sl.sale_listing_views_description_text, '') AS views_description_text,
    COALESCE(sl.sale_listing_appliances, ARRAY[]::text[]) AS appliances,
    COALESCE(sl.sale_listing_features, ARRAY[]::text[]) AS features,
    sl.sale_listing_plot_area_value,
    COALESCE(sl.sale_listing_services_text, '') AS services_text,
    COALESCE(sl.sale_listing_transport_text, '') AS transport_text,
    sl.sale_listing_previous_asking_price,
    sl.sale_listing_previous_debt_free_price,
    sl.sale_listing_new_development,
    COALESCE(sl.sale_listing_description_text, '') AS description_text,
    COALESCE(sl.sale_listing_availability_text, '') AS availability_text,
    COALESCE(sl.sale_listing_renovations_done_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDone}', sa.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{property,renovationsDoneDescription}')), ''), '') AS renovations_done_text,
    COALESCE(sl.sale_listing_renovations_planned_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlanned}', sa.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{property,renovationsPlannedDescription}')), ''), '') AS renovations_planned_text,
    COALESCE(sl.sale_listing_additional_info_text, '') AS additional_info_text,
    COALESCE(sl.sale_listing_charges_text, '') AS charges_text,
    sl.sale_listing_maintenance_charge_monthly,
    sl.sale_listing_total_charge_monthly,
    sl.sale_listing_water_charge,
    COALESCE(sl.sale_listing_housing_company_name, '') AS housing_company_name,
    COALESCE(sl.sale_listing_housing_company_business_id, '') AS housing_company_business_id,
    COALESCE(sl.sale_listing_building_material, '') AS building_material,
    COALESCE(sl.sale_listing_heating_system, '') AS heating_system,
    COALESCE(sl.sale_listing_roof_type, '') AS roof_type,
    COALESCE(sl.sale_listing_roof_material, '') AS roof_material,
    COALESCE(sl.sale_listing_car_storage_text, '') AS car_storage_text,
    COALESCE(sl.sale_listing_building_description_text, '') AS building_description_text,
    COALESCE(sl.sale_listing_building_other_info_text, '') AS building_other_info_text
FROM public.property_source_offerings sl
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
    AND sl.sale_listing_source_kind IN ('ad', 'announcement')
LIMIT 1;

-- name: RelinkPropertyUnitBuilding :one
SELECT public.fnc__relink_property_unit_building(
    sqlc.arg(property_unit_id)::uuid,
    sqlc.arg(physical_building_id)::uuid,
    sqlc.arg(reason)::text
)::jsonb;

-- name: RelinkPhysicalBuildingHousingCompany :one
SELECT public.fnc__relink_physical_building_housing_company(
    sqlc.arg(physical_building_id)::uuid,
    sqlc.arg(housing_company_id)::uuid,
    sqlc.arg(reason)::text
)::jsonb;
