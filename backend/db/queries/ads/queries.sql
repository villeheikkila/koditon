-- name: DeleteSaleListingForShortcutAd :exec
DELETE FROM public.property_source_offerings
WHERE shortcut_ad_id = sqlc.arg(shortcut_ad_id);

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
DELETE FROM public.property_source_offerings
WHERE frontdoor_building_announcement_id = sqlc.arg(frontdoor_building_announcement_id);

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

-- name: RefreshHousingCompanyFactsForPropertySourceOffering :exec
SELECT public.fnc__refresh_housing_company_facts_for_property_source_offering(sqlc.arg(sale_listing_id));

-- name: GetApartmentProfileForSaleListing :one
WITH linked AS (
    SELECT pu.property_unit_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
)
SELECT ap.*
FROM public.apartment_profiles ap
JOIN linked ON linked.property_unit_id = ap.property_unit_id
LIMIT 1;

-- name: EnsurePhysicalBuildingForSaleListing :exec
WITH linked AS (
    SELECT
        pos.sale_listing_id,
        pu.property_unit_id,
        pu.housing_company_id,
        hc.housing_company_identity_key
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
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
    WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
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
)
UPDATE public.property_units pu
SET physical_building_id = inserted.physical_building_id,
    property_unit_updated_at = now()
FROM listing, inserted
WHERE pu.property_unit_id = listing.property_unit_id;

-- name: UpsertSaleListingProviderClaims :exec
WITH source_values AS (
    SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()) AS observed_at, 'unit'::text AS claim_namespace, 'area_m2'::text AS claim_key, 'number'::text AS value_kind, NULL::text AS value_text, sl.sale_listing_area_value AS value_number, NULL::boolean AS value_bool, 'sale_listing_area_value'::text AS source_path, sl.sale_listing_area_value::text AS evidence_text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND sl.sale_listing_area_value IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'unit', 'living_area_m2', 'number', NULL::text, sl.sale_listing_living_area_value, NULL::boolean, 'sale_listing_living_area_value', sl.sale_listing_living_area_value::text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND sl.sale_listing_living_area_value IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'layout', 'room_layout', 'text', sl.sale_listing_room_layout, NULL::double precision, NULL::boolean, 'sale_listing_room_layout', sl.sale_listing_room_layout FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND NULLIF(trim(sl.sale_listing_room_layout), '') IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'layout', 'room_count', 'number', NULL::text, sl.sale_listing_rooms_count::double precision, NULL::boolean, 'sale_listing_rooms_count', sl.sale_listing_rooms_count::text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND sl.sale_listing_rooms_count IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'condition', 'condition', 'text', sl.sale_listing_condition, NULL::double precision, NULL::boolean, 'sale_listing_condition', sl.sale_listing_condition FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND NULLIF(trim(sl.sale_listing_condition), '') IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'unit', 'sauna', 'bool', NULL::text, NULL::double precision, sl.sale_listing_sauna, 'sale_listing_sauna', sl.sale_listing_sauna::text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND sl.sale_listing_sauna IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'unit', 'balcony', 'bool', NULL::text, NULL::double precision, sl.sale_listing_balcony, 'sale_listing_balcony', sl.sale_listing_balcony::text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND sl.sale_listing_balcony IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'parking', 'parking_text', 'text', sl.sale_listing_parking_text, NULL::double precision, NULL::boolean, 'sale_listing_parking_text', sl.sale_listing_parking_text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND NULLIF(trim(sl.sale_listing_parking_text), '') IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'building', 'build_year', 'number', NULL::text, sl.sale_listing_build_year::double precision, NULL::boolean, 'sale_listing_build_year', sl.sale_listing_build_year::text FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND sl.sale_listing_build_year IS NOT NULL
    UNION ALL SELECT sl.sale_listing_id, COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_published_at, now()), 'building', 'heating_method', 'text', sl.sale_listing_heating_system, NULL::double precision, NULL::boolean, 'sale_listing_heating_system', sl.sale_listing_heating_system FROM public.property_source_offerings sl WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id) AND NULLIF(trim(sl.sale_listing_heating_system), '') IS NOT NULL
)
INSERT INTO public.property_claims (
    property_claim_target_type,
    property_claim_target_id,
    property_claim_namespace,
    property_claim_key,
    property_claim_value_kind,
    property_claim_value_text,
    property_claim_value_number,
    property_claim_value_bool,
    property_claim_source_record_table,
    property_claim_source_record_id,
    property_claim_source_path,
    property_claim_evidence_text,
    property_claim_method,
    property_claim_confidence,
    property_claim_source_reliability,
    property_claim_observed_at
)
SELECT
    'sale_listing',
    sale_listing_id,
    claim_namespace,
    claim_key,
    value_kind,
    NULLIF(value_text, ''),
    value_number,
    value_bool,
    'property_source_offerings',
    sale_listing_id,
    source_path,
    evidence_text,
    'provider_field',
    1,
    0.85,
    observed_at
FROM source_values
ON CONFLICT (
    property_claim_target_type,
    property_claim_target_id,
    property_claim_namespace,
    property_claim_key,
    property_claim_source_record_table,
    property_claim_source_record_id,
    COALESCE(property_claim_source_path, ''),
    property_claim_method
) DO UPDATE SET
    property_claim_value_kind = EXCLUDED.property_claim_value_kind,
    property_claim_value_text = EXCLUDED.property_claim_value_text,
    property_claim_value_number = EXCLUDED.property_claim_value_number,
    property_claim_value_bool = EXCLUDED.property_claim_value_bool,
    property_claim_evidence_text = EXCLUDED.property_claim_evidence_text,
    property_claim_confidence = EXCLUDED.property_claim_confidence,
    property_claim_source_reliability = EXCLUDED.property_claim_source_reliability,
    property_claim_observed_at = EXCLUDED.property_claim_observed_at,
    property_claim_updated_at = now();

-- name: ProjectApartmentProfileForSaleListing :exec
WITH linked AS (
    SELECT
        pos.sale_listing_id,
        pu.property_unit_id,
        pu.housing_company_id,
        pu.physical_building_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
),
listing AS (
    SELECT
        sl.*,
        linked.housing_company_id,
        linked.property_unit_id,
        linked.physical_building_id
    FROM public.property_source_offerings sl
    JOIN linked ON linked.sale_listing_id = sl.sale_listing_id
    WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
),
claims AS (
    SELECT
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace = 'balcony' AND property_claim_key IN ('glazing','balcony_glazing')) AS balcony_glazing,
        max(property_claim_value_text) FILTER (WHERE property_claim_namespace = 'layout' AND property_claim_key = 'kitchen_type' AND property_claim_value_text = ANY (ARRAY['separate','open','kitchenette','unknown']::text[])) AS kitchen_type,
        max(property_claim_value_text) FILTER (WHERE property_claim_namespace = 'layout' AND property_claim_key = 'layout_quality' AND property_claim_value_text = ANY (ARRAY['weak','average','good','excellent','unknown']::text[])) AS layout_quality,
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace = 'layout' AND property_claim_key = 'awkward_layout') AS awkward_layout,
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace IN ('condition','unit') AND property_claim_key = 'surface_renovation_need') AS surface_renovation_need,
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace IN ('condition','unit') AND property_claim_key = 'modernization_need') AS modernization_need,
        max(property_claim_value_text) FILTER (WHERE property_claim_namespace = 'storage' AND property_claim_key = 'storage_quality' AND property_claim_value_text = ANY (ARRAY['weak','normal','good','unknown']::text[])) AS storage_quality,
        max(property_claim_value_text) FILTER (WHERE property_claim_namespace = 'views' AND property_claim_key = 'view_quality' AND property_claim_value_text = ANY (ARRAY['weak','normal','good','excellent','unknown']::text[])) AS view_quality,
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace = 'views' AND property_claim_key = 'noise_risk') AS noise_risk,
        max(property_claim_value_text) FILTER (WHERE property_claim_namespace IN ('building','unit') AND property_claim_key = 'accessibility' AND property_claim_value_text = ANY (ARRAY['poor','average','good','unknown']::text[])) AS accessibility,
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace = 'kitchen' AND property_claim_key = 'renovated') AS kitchen_renovated,
        bool_or(property_claim_value_bool) FILTER (WHERE property_claim_namespace = 'bathroom' AND property_claim_key = 'renovated') AS bathroom_renovated,
        max(property_claim_confidence * property_claim_source_reliability) AS evidence_score
    FROM public.property_claims
    WHERE property_claim_target_type = 'sale_listing'
        AND property_claim_target_id = sqlc.arg(sale_listing_id)
)
INSERT INTO public.apartment_profiles (
    property_unit_id,
    housing_company_id,
    physical_building_id,
    source_sale_listing_id,
    apartment_profile_area_m2,
    apartment_profile_living_area_m2,
    apartment_profile_room_layout,
    apartment_profile_room_count,
    apartment_profile_bedroom_count,
    apartment_profile_floor_level,
    apartment_profile_total_floors,
    apartment_profile_kitchen_type,
    apartment_profile_condition,
    apartment_profile_sauna,
    apartment_profile_balcony,
    apartment_profile_parking_type,
    apartment_profile_balcony_glazing,
    apartment_profile_layout_quality,
    apartment_profile_awkward_layout,
    apartment_profile_surface_renovation_need,
    apartment_profile_modernization_need,
    apartment_profile_storage_quality,
    apartment_profile_view_quality,
    apartment_profile_noise_risk,
    apartment_profile_accessibility,
    apartment_profile_kitchen_condition,
    apartment_profile_bathroom_condition,
    apartment_profile_confidence,
    apartment_profile_updated_at
)
SELECT
    property_unit_id,
    housing_company_id,
    physical_building_id,
    sale_listing_id,
    sale_listing_area_value,
    sale_listing_living_area_value,
    sale_listing_room_layout,
    sale_listing_rooms_count,
    sale_listing_bedrooms_count,
    sale_listing_floor_level,
    sale_listing_total_floors,
    COALESCE(claims.kitchen_type, CASE
        WHEN lower(COALESCE(sale_listing_room_layout, '')) LIKE '%avok%' OR lower(COALESCE(sale_listing_kitchen_description_text, '')) LIKE '%avokeitti%' THEN 'open'
        WHEN lower(COALESCE(sale_listing_room_layout, '')) LIKE '%kk%' OR lower(COALESCE(sale_listing_room_layout, '')) LIKE '%keittonurk%' THEN 'kitchenette'
        WHEN lower(COALESCE(sale_listing_room_layout, '')) LIKE '%k%' OR lower(COALESCE(sale_listing_kitchen_description_text, '')) LIKE '%erillinen%' THEN 'separate'
        ELSE NULL
    END),
    CASE
        WHEN lower(COALESCE(sale_listing_condition, '')) LIKE '%uusi%' OR lower(COALESCE(sale_listing_condition, '')) LIKE '%new%' THEN 'new'
        WHEN lower(COALESCE(sale_listing_condition, '')) LIKE '%erinomain%' OR lower(COALESCE(sale_listing_condition, '')) LIKE '%excellent%' THEN 'excellent'
        WHEN lower(COALESCE(sale_listing_condition, '')) LIKE '%hyv%' OR lower(COALESCE(sale_listing_condition, '')) LIKE '%good%' THEN 'good'
        WHEN lower(COALESCE(sale_listing_condition, '')) LIKE '%tyyd%' OR lower(COALESCE(sale_listing_condition, '')) LIKE '%fair%' THEN 'fair'
        WHEN lower(COALESCE(sale_listing_condition, '')) LIKE '%huono%' OR lower(COALESCE(sale_listing_condition, '')) LIKE '%poor%' THEN 'poor'
        ELSE NULL
    END,
    sale_listing_sauna,
    sale_listing_balcony,
    CASE
        WHEN sale_listing_parking_text IS NULL OR trim(sale_listing_parking_text) = '' THEN NULL
        WHEN lower(sale_listing_parking_text) LIKE '%autotalli%' OR lower(sale_listing_parking_text) LIKE '%garage%' THEN 'garage'
        WHEN lower(sale_listing_parking_text) LIKE '%katos%' OR lower(sale_listing_parking_text) LIKE '%carport%' THEN 'carport'
        WHEN lower(sale_listing_parking_text) LIKE '%osake%' THEN 'separate_share'
        WHEN lower(sale_listing_parking_text) LIKE '%piha%' OR lower(sale_listing_parking_text) LIKE '%pihapaikka%' THEN 'yard'
        WHEN lower(sale_listing_parking_text) LIKE '%katu%' OR lower(sale_listing_parking_text) LIKE '%street%' THEN 'street'
        ELSE 'unknown'
    END,
    claims.balcony_glazing,
    claims.layout_quality,
    claims.awkward_layout,
    claims.surface_renovation_need,
    claims.modernization_need,
    claims.storage_quality,
    claims.view_quality,
    claims.noise_risk,
    claims.accessibility,
    CASE WHEN claims.kitchen_renovated IS TRUE THEN 'good' ELSE NULL END,
    CASE WHEN claims.bathroom_renovated IS TRUE THEN 'good' ELSE NULL END,
    CASE
        WHEN claims.evidence_score >= 0.8 AND sale_listing_area_value IS NOT NULL AND sale_listing_room_layout IS NOT NULL THEN 'high'
        WHEN sale_listing_area_value IS NOT NULL AND sale_listing_room_layout IS NOT NULL THEN 'medium'
        ELSE 'low'
    END,
    now()
FROM listing
LEFT JOIN claims ON true
ON CONFLICT (property_unit_id) DO UPDATE SET
    housing_company_id = EXCLUDED.housing_company_id,
    physical_building_id = EXCLUDED.physical_building_id,
    source_sale_listing_id = EXCLUDED.source_sale_listing_id,
    apartment_profile_area_m2 = EXCLUDED.apartment_profile_area_m2,
    apartment_profile_living_area_m2 = EXCLUDED.apartment_profile_living_area_m2,
    apartment_profile_room_layout = EXCLUDED.apartment_profile_room_layout,
    apartment_profile_room_count = EXCLUDED.apartment_profile_room_count,
    apartment_profile_bedroom_count = EXCLUDED.apartment_profile_bedroom_count,
    apartment_profile_floor_level = EXCLUDED.apartment_profile_floor_level,
    apartment_profile_total_floors = EXCLUDED.apartment_profile_total_floors,
    apartment_profile_kitchen_type = COALESCE(EXCLUDED.apartment_profile_kitchen_type, public.apartment_profiles.apartment_profile_kitchen_type),
    apartment_profile_condition = EXCLUDED.apartment_profile_condition,
    apartment_profile_sauna = EXCLUDED.apartment_profile_sauna,
    apartment_profile_balcony = EXCLUDED.apartment_profile_balcony,
    apartment_profile_parking_type = EXCLUDED.apartment_profile_parking_type,
    apartment_profile_balcony_glazing = COALESCE(EXCLUDED.apartment_profile_balcony_glazing, public.apartment_profiles.apartment_profile_balcony_glazing),
    apartment_profile_layout_quality = COALESCE(EXCLUDED.apartment_profile_layout_quality, public.apartment_profiles.apartment_profile_layout_quality),
    apartment_profile_awkward_layout = COALESCE(EXCLUDED.apartment_profile_awkward_layout, public.apartment_profiles.apartment_profile_awkward_layout),
    apartment_profile_surface_renovation_need = COALESCE(EXCLUDED.apartment_profile_surface_renovation_need, public.apartment_profiles.apartment_profile_surface_renovation_need),
    apartment_profile_modernization_need = COALESCE(EXCLUDED.apartment_profile_modernization_need, public.apartment_profiles.apartment_profile_modernization_need),
    apartment_profile_storage_quality = COALESCE(EXCLUDED.apartment_profile_storage_quality, public.apartment_profiles.apartment_profile_storage_quality),
    apartment_profile_view_quality = COALESCE(EXCLUDED.apartment_profile_view_quality, public.apartment_profiles.apartment_profile_view_quality),
    apartment_profile_noise_risk = COALESCE(EXCLUDED.apartment_profile_noise_risk, public.apartment_profiles.apartment_profile_noise_risk),
    apartment_profile_accessibility = COALESCE(EXCLUDED.apartment_profile_accessibility, public.apartment_profiles.apartment_profile_accessibility),
    apartment_profile_kitchen_condition = COALESCE(EXCLUDED.apartment_profile_kitchen_condition, public.apartment_profiles.apartment_profile_kitchen_condition),
    apartment_profile_bathroom_condition = COALESCE(EXCLUDED.apartment_profile_bathroom_condition, public.apartment_profiles.apartment_profile_bathroom_condition),
    apartment_profile_confidence = EXCLUDED.apartment_profile_confidence,
    apartment_profile_updated_at = now();

-- name: ProjectHousingCompanyRenovationsForSaleListing :exec
WITH linked AS (
    SELECT pu.housing_company_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
),
renovations AS (
    SELECT
        linked.housing_company_id,
        r.sale_listing_id,
        CASE r.property_source_offering_renovation_category
            WHEN 'electricity' THEN 'electricity'
            WHEN 'windows' THEN 'window'
            WHEN 'pipes' THEN 'pipe'
            ELSE r.property_source_offering_renovation_category
        END AS category,
        CASE r.property_source_offering_renovation_status
            WHEN 'done' THEN 'done'
            WHEN 'planned' THEN 'planned'
            ELSE 'unknown'
        END AS status,
        COALESCE(NULLIF(r.property_source_offering_renovation_stage, ''), 'unknown') AS stage,
        COALESCE(NULLIF(r.property_source_offering_renovation_scope, ''), 'unknown') AS scope,
        COALESCE(NULLIF(r.property_source_offering_renovation_responsibility, ''), 'unknown') AS responsibility,
        r.property_source_offering_renovation_year AS year,
        r.property_source_offering_renovation_cost_estimate_eur AS cost_estimate_eur,
        CASE
            WHEN r.property_source_offering_renovation_confidence >= 80 THEN 'high'
            WHEN r.property_source_offering_renovation_confidence >= 60 THEN 'medium'
            ELSE 'low'
        END AS confidence,
        COALESCE(NULLIF(r.property_source_offering_renovation_text, ''), r.property_source_offering_renovation_category) AS summary
    FROM public.property_source_offering_renovations r
    JOIN linked ON linked.housing_company_id IS NOT NULL
    WHERE r.sale_listing_id = sqlc.arg(sale_listing_id)
        AND r.property_source_offering_renovation_category = ANY (ARRAY['pipe','pipes','water_supply','sewer','roof','facade','window','windows','balcony','elevator','heating','ventilation','drainage','electricity','yard','common_areas','other']::text[])
)
INSERT INTO public.housing_company_renovations (
    housing_company_id,
    source_sale_listing_id,
    housing_company_renovation_category,
    housing_company_renovation_status,
    housing_company_renovation_stage,
    housing_company_renovation_scope,
    housing_company_renovation_responsibility,
    housing_company_renovation_year,
    housing_company_renovation_cost_estimate_eur,
    housing_company_renovation_confidence,
    housing_company_renovation_evidence_level,
    housing_company_renovation_summary,
    housing_company_renovation_updated_at
)
SELECT
    housing_company_id,
    sale_listing_id,
    category,
    status,
    stage,
    scope,
    responsibility,
    year,
    cost_estimate_eur,
    confidence,
    'ad_only',
    summary,
    now()
FROM renovations
ON CONFLICT (
    housing_company_id,
    source_sale_listing_id,
    housing_company_renovation_category,
    housing_company_renovation_status,
    housing_company_renovation_stage,
    housing_company_renovation_scope,
    COALESCE(housing_company_renovation_year, -1),
    md5(COALESCE(housing_company_renovation_summary, ''))
) WHERE source_sale_listing_id IS NOT NULL DO UPDATE SET
    housing_company_renovation_responsibility = EXCLUDED.housing_company_renovation_responsibility,
    housing_company_renovation_cost_estimate_eur = EXCLUDED.housing_company_renovation_cost_estimate_eur,
    housing_company_renovation_confidence = EXCLUDED.housing_company_renovation_confidence,
    housing_company_renovation_evidence_level = EXCLUDED.housing_company_renovation_evidence_level,
    housing_company_renovation_summary = EXCLUDED.housing_company_renovation_summary,
    housing_company_renovation_updated_at = now();

-- name: ProjectHousingCompanySystemsFromRenovationsForSaleListing :exec
WITH linked AS (
    SELECT pu.housing_company_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
),
source AS (
    SELECT
        r.housing_company_id,
        CASE r.housing_company_renovation_category
            WHEN 'pipe' THEN 'pipes'
            WHEN 'window' THEN 'windows'
            WHEN 'balcony' THEN 'balconies'
            WHEN 'electricity' THEN 'electrical'
            ELSE r.housing_company_renovation_category
        END AS system_type,
        max(r.housing_company_renovation_year) FILTER (WHERE r.housing_company_renovation_status = 'done') AS last_renovated_year,
        min(COALESCE(r.housing_company_renovation_window_start_year, r.housing_company_renovation_year)) FILTER (WHERE r.housing_company_renovation_status IN ('planned','forecast','suspected')) AS next_expected_start_year,
        max(COALESCE(r.housing_company_renovation_window_end_year, r.housing_company_renovation_year)) FILTER (WHERE r.housing_company_renovation_status IN ('planned','forecast','suspected')) AS next_expected_end_year,
        bool_or(r.housing_company_renovation_status = 'planned') AS has_planned,
        bool_or(r.housing_company_renovation_status = 'done') AS has_done,
        max(r.housing_company_renovation_confidence) AS confidence,
        max(r.housing_company_renovation_evidence_level) AS evidence_level
    FROM public.housing_company_renovations r
    JOIN linked ON linked.housing_company_id = r.housing_company_id
    WHERE r.source_sale_listing_id = sqlc.arg(sale_listing_id)
    GROUP BY r.housing_company_id, system_type
)
INSERT INTO public.housing_company_systems (
    housing_company_id,
    housing_company_system_type,
    housing_company_system_status,
    housing_company_system_last_renovated_year,
    housing_company_system_next_expected_start_year,
    housing_company_system_next_expected_end_year,
    housing_company_system_confidence,
    housing_company_system_evidence_level,
    housing_company_system_summary,
    housing_company_system_updated_at
)
SELECT
    housing_company_id,
    system_type,
    CASE
        WHEN has_planned THEN 'planned'
        WHEN has_done THEN 'renewed'
        ELSE 'unknown'
    END,
    last_renovated_year,
    next_expected_start_year,
    next_expected_end_year,
    COALESCE(confidence, 'low'),
    COALESCE(evidence_level, 'none'),
    concat_ws(' ', system_type, CASE WHEN has_planned THEN 'has planned renovation evidence' WHEN has_done THEN 'has completed renovation evidence' ELSE 'has renovation evidence' END),
    now()
FROM source
WHERE system_type = ANY (ARRAY['pipes','water_supply','sewer','roof','facade','windows','balconies','elevator','heating','ventilation','drainage','electrical','yard','common_areas']::text[])
ON CONFLICT (housing_company_id, housing_company_system_type) DO UPDATE SET
    housing_company_system_status = CASE
        WHEN EXCLUDED.housing_company_system_status = 'planned' THEN 'planned'
        WHEN public.housing_company_systems.housing_company_system_status = 'planned' THEN 'planned'
        ELSE EXCLUDED.housing_company_system_status
    END,
    housing_company_system_last_renovated_year = GREATEST(public.housing_company_systems.housing_company_system_last_renovated_year, EXCLUDED.housing_company_system_last_renovated_year),
    housing_company_system_next_expected_start_year = COALESCE(LEAST(public.housing_company_systems.housing_company_system_next_expected_start_year, EXCLUDED.housing_company_system_next_expected_start_year), public.housing_company_systems.housing_company_system_next_expected_start_year, EXCLUDED.housing_company_system_next_expected_start_year),
    housing_company_system_next_expected_end_year = COALESCE(GREATEST(public.housing_company_systems.housing_company_system_next_expected_end_year, EXCLUDED.housing_company_system_next_expected_end_year), public.housing_company_systems.housing_company_system_next_expected_end_year, EXCLUDED.housing_company_system_next_expected_end_year),
    housing_company_system_confidence = CASE
        WHEN public.housing_company_systems.housing_company_system_confidence = 'high' OR EXCLUDED.housing_company_system_confidence = 'high' THEN 'high'
        WHEN public.housing_company_systems.housing_company_system_confidence = 'medium' OR EXCLUDED.housing_company_system_confidence = 'medium' THEN 'medium'
        ELSE 'low'
    END,
    housing_company_system_evidence_level = EXCLUDED.housing_company_system_evidence_level,
    housing_company_system_summary = EXCLUDED.housing_company_system_summary,
    housing_company_system_updated_at = now();

-- name: ProjectBuildingProfileForSaleListing :exec
WITH linked AS (
    SELECT
        pu.physical_building_id,
        pu.housing_company_id,
        sl.*
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
        AND pu.physical_building_id IS NOT NULL
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
)
INSERT INTO public.building_profiles (
    physical_building_id,
    housing_company_id,
    building_profile_build_year,
    building_profile_floor_count,
    building_profile_apartment_count,
    building_profile_energy_class,
    building_profile_heating_method,
    building_profile_material,
    building_profile_roof_type,
    building_profile_roof_material,
    building_profile_elevator,
    building_profile_confidence,
    building_profile_updated_at
)
SELECT
    physical_building_id,
    housing_company_id,
    sale_listing_build_year,
    sale_listing_total_floors,
    sale_listing_apartment_count,
    COALESCE(sale_listing_energy_efficiency_label, sale_listing_energy_class),
    sale_listing_heating_system,
    sale_listing_building_material,
    sale_listing_roof_type,
    sale_listing_roof_material,
    sale_listing_elevator,
    CASE WHEN sale_listing_build_year IS NOT NULL AND sale_listing_total_floors IS NOT NULL THEN 'medium' ELSE 'low' END,
    now()
FROM linked
ON CONFLICT (physical_building_id) DO UPDATE SET
    housing_company_id = COALESCE(public.building_profiles.housing_company_id, EXCLUDED.housing_company_id),
    building_profile_build_year = COALESCE(public.building_profiles.building_profile_build_year, EXCLUDED.building_profile_build_year),
    building_profile_floor_count = COALESCE(public.building_profiles.building_profile_floor_count, EXCLUDED.building_profile_floor_count),
    building_profile_apartment_count = COALESCE(public.building_profiles.building_profile_apartment_count, EXCLUDED.building_profile_apartment_count),
    building_profile_energy_class = COALESCE(public.building_profiles.building_profile_energy_class, EXCLUDED.building_profile_energy_class),
    building_profile_heating_method = COALESCE(public.building_profiles.building_profile_heating_method, EXCLUDED.building_profile_heating_method),
    building_profile_material = COALESCE(public.building_profiles.building_profile_material, EXCLUDED.building_profile_material),
    building_profile_roof_type = COALESCE(public.building_profiles.building_profile_roof_type, EXCLUDED.building_profile_roof_type),
    building_profile_roof_material = COALESCE(public.building_profiles.building_profile_roof_material, EXCLUDED.building_profile_roof_material),
    building_profile_elevator = COALESCE(public.building_profiles.building_profile_elevator, EXCLUDED.building_profile_elevator),
    building_profile_confidence = EXCLUDED.building_profile_confidence,
    building_profile_updated_at = now();

-- name: ProjectHousingCompanyProfileForSaleListing :exec
WITH linked AS (
    SELECT
        pu.housing_company_id,
        hc.housing_company_name,
        hc.housing_company_business_id,
        hc.housing_company_build_year,
        hc.housing_company_apartment_count,
        hc.housing_company_energy_efficiency_label,
        sl.sale_listing_plot_type_code,
        sl.sale_listing_plot_type_raw
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.housing_companies hc ON hc.housing_company_id = pu.housing_company_id
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
),
repair AS (
    SELECT
        linked.housing_company_id,
        count(*) FILTER (WHERE r.housing_company_renovation_status IN ('planned','suspected','forecast')) AS upcoming_count
    FROM linked
    LEFT JOIN public.housing_company_renovations r ON r.housing_company_id = linked.housing_company_id
    GROUP BY linked.housing_company_id
)
INSERT INTO public.housing_company_profiles (
    housing_company_id,
    housing_company_profile_name,
    housing_company_profile_business_id,
    housing_company_profile_build_year,
    housing_company_profile_apartment_count,
    housing_company_profile_plot_ownership_type,
    housing_company_profile_energy_class,
    housing_company_profile_repair_backlog_risk,
    housing_company_profile_confidence,
    housing_company_profile_updated_at
)
SELECT
    linked.housing_company_id,
    housing_company_name,
    housing_company_business_id,
    housing_company_build_year,
    housing_company_apartment_count,
    COALESCE(NULLIF(sale_listing_plot_type_code, ''), NULLIF(sale_listing_plot_type_raw, '')),
    housing_company_energy_efficiency_label,
    CASE WHEN repair.upcoming_count >= 3 THEN 'high' WHEN repair.upcoming_count > 0 THEN 'medium' ELSE 'unknown' END,
    CASE WHEN housing_company_business_id IS NOT NULL OR housing_company_name IS NOT NULL THEN 'medium' ELSE 'low' END,
    now()
FROM linked
JOIN repair ON repair.housing_company_id = linked.housing_company_id
ON CONFLICT (housing_company_id) DO UPDATE SET
    housing_company_profile_name = COALESCE(public.housing_company_profiles.housing_company_profile_name, EXCLUDED.housing_company_profile_name),
    housing_company_profile_business_id = COALESCE(public.housing_company_profiles.housing_company_profile_business_id, EXCLUDED.housing_company_profile_business_id),
    housing_company_profile_build_year = COALESCE(public.housing_company_profiles.housing_company_profile_build_year, EXCLUDED.housing_company_profile_build_year),
    housing_company_profile_apartment_count = COALESCE(public.housing_company_profiles.housing_company_profile_apartment_count, EXCLUDED.housing_company_profile_apartment_count),
    housing_company_profile_plot_ownership_type = COALESCE(public.housing_company_profiles.housing_company_profile_plot_ownership_type, EXCLUDED.housing_company_profile_plot_ownership_type),
    housing_company_profile_energy_class = COALESCE(public.housing_company_profiles.housing_company_profile_energy_class, EXCLUDED.housing_company_profile_energy_class),
    housing_company_profile_repair_backlog_risk = EXCLUDED.housing_company_profile_repair_backlog_risk,
    housing_company_profile_confidence = EXCLUDED.housing_company_profile_confidence,
    housing_company_profile_updated_at = now();

-- name: ProjectQualityScoresForSaleListing :exec
WITH linked AS (
    SELECT
        pu.property_unit_id,
        po.property_offering_id,
        pu.physical_building_id,
        pu.housing_company_id,
        sl.sale_listing_asking_price,
        sl.sale_listing_debt_free_price,
        pt.prices_transaction_price,
        c.sale_listing_prices_transaction_match_score,
        ap.apartment_profile_condition,
        ap.apartment_profile_layout_quality,
        ap.apartment_profile_floor_level,
        ap.apartment_profile_total_floors,
        ap.apartment_profile_kitchen_condition,
        ap.apartment_profile_bathroom_condition,
        ap.apartment_profile_modernization_need,
        ap.apartment_profile_surface_renovation_need,
        ap.apartment_profile_balcony,
        ap.apartment_profile_balcony_glazing,
        ap.apartment_profile_storage_quality,
        ap.apartment_profile_parking_type,
        ap.apartment_profile_view_quality,
        ap.apartment_profile_noise_risk,
        ap.apartment_profile_accessibility,
        bp.building_profile_build_year,
        bp.building_profile_energy_class,
        bp.building_profile_heating_method,
        bp.building_profile_elevator,
        hcp.housing_company_profile_financial_risk,
        hcp.housing_company_profile_maintenance_risk,
        hcp.housing_company_profile_plot_ownership_type,
        hcp.housing_company_profile_repair_backlog_risk
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = pos.sale_listing_id
    LEFT JOIN public.prices_transactions pt ON pt.prices_transaction_id = sl.prices_transaction_id
    LEFT JOIN LATERAL (
        SELECT sale_listing_prices_transaction_match_score
        FROM public.sale_listing_prices_transaction_match_candidates c
        WHERE c.sale_listing_id = sl.sale_listing_id
            AND c.prices_transaction_id = sl.prices_transaction_id
        ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
        LIMIT 1
    ) c ON true
    LEFT JOIN public.apartment_profiles ap ON ap.property_unit_id = pu.property_unit_id
    LEFT JOIN public.building_profiles bp ON bp.physical_building_id = pu.physical_building_id
    LEFT JOIN public.housing_company_profiles hcp ON hcp.housing_company_id = pu.housing_company_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
),
scores AS (
    SELECT 'property_unit'::text AS target_type, property_unit_id AS target_id, 'apartment_condition'::text AS dimension,
        CASE
            WHEN apartment_profile_modernization_need IS TRUE OR apartment_profile_surface_renovation_need IS TRUE THEN 40
            WHEN apartment_profile_condition = 'new' THEN 95
            WHEN apartment_profile_condition = 'excellent' THEN 90
            WHEN apartment_profile_condition = 'good' THEN 75
            WHEN apartment_profile_condition = 'fair' THEN 45
            WHEN apartment_profile_condition = 'poor' THEN 20
            ELSE 50
        END AS score,
        jsonb_build_array(COALESCE(apartment_profile_condition, 'condition unknown'), CASE WHEN apartment_profile_modernization_need IS TRUE THEN 'modernization need' ELSE NULL END, CASE WHEN apartment_profile_surface_renovation_need IS TRUE THEN 'surface renovation need' ELSE NULL END) AS reasons
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'layout_efficiency',
        CASE apartment_profile_layout_quality WHEN 'excellent' THEN 95 WHEN 'good' THEN 80 WHEN 'average' THEN 60 WHEN 'weak' THEN 35 ELSE 55 END,
        jsonb_build_array(COALESCE(apartment_profile_layout_quality, 'layout quality unknown'))
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'wet_room_state',
        CASE apartment_profile_bathroom_condition WHEN 'new' THEN 95 WHEN 'excellent' THEN 90 WHEN 'good' THEN 75 WHEN 'fair' THEN 45 WHEN 'poor' THEN 20 ELSE 50 END,
        jsonb_build_array(COALESCE(apartment_profile_bathroom_condition, 'bathroom state unknown'))
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'kitchen_state',
        CASE apartment_profile_kitchen_condition WHEN 'new' THEN 95 WHEN 'excellent' THEN 90 WHEN 'good' THEN 75 WHEN 'fair' THEN 45 WHEN 'poor' THEN 20 ELSE 50 END,
        jsonb_build_array(COALESCE(apartment_profile_kitchen_condition, 'kitchen state unknown'))
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'floor_elevator_fit',
        CASE
            WHEN building_profile_elevator IS TRUE THEN 85
            WHEN apartment_profile_floor_level >= 4 THEN 25
            WHEN apartment_profile_floor_level >= 3 THEN 45
            WHEN apartment_profile_floor_level IS NULL THEN 50
            ELSE 70
        END,
        jsonb_build_array(COALESCE('floor ' || apartment_profile_floor_level::text, 'floor unknown'), CASE WHEN building_profile_elevator IS TRUE THEN 'elevator' WHEN building_profile_elevator IS FALSE THEN 'no elevator' ELSE 'elevator unknown' END)
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'balcony_storage_parking',
        LEAST(100, 45 + CASE WHEN apartment_profile_balcony IS TRUE THEN 15 ELSE 0 END + CASE WHEN apartment_profile_balcony_glazing IS TRUE THEN 10 ELSE 0 END + CASE apartment_profile_storage_quality WHEN 'good' THEN 15 WHEN 'normal' THEN 8 ELSE 0 END + CASE WHEN apartment_profile_parking_type IN ('garage','carport','yard','separate_share') THEN 15 ELSE 0 END),
        jsonb_build_array(CASE WHEN apartment_profile_balcony IS TRUE THEN 'balcony' ELSE 'balcony unknown or absent' END, COALESCE(apartment_profile_storage_quality, 'storage unknown'), COALESCE(apartment_profile_parking_type, 'parking unknown'))
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'view_noise_privacy',
        CASE
            WHEN apartment_profile_noise_risk IS TRUE THEN 35
            WHEN apartment_profile_view_quality = 'excellent' THEN 90
            WHEN apartment_profile_view_quality = 'good' THEN 75
            WHEN apartment_profile_view_quality = 'weak' THEN 40
            ELSE 55
        END,
        jsonb_build_array(COALESCE(apartment_profile_view_quality, 'view unknown'), CASE WHEN apartment_profile_noise_risk IS TRUE THEN 'noise risk' ELSE NULL END)
    FROM linked
    UNION ALL
    SELECT 'property_unit', property_unit_id, 'apartment_document_consistency',
        CASE WHEN apartment_profile_confidence = 'high' THEN 90 WHEN apartment_profile_confidence = 'medium' THEN 65 ELSE 35 END,
        jsonb_build_array('apartment profile confidence ' || COALESCE(apartment_profile_confidence, 'unknown'))
    FROM linked
    UNION ALL
    SELECT 'physical_building', physical_building_id, 'building_age',
        CASE WHEN building_profile_build_year IS NULL THEN 50 WHEN building_profile_build_year >= 2015 THEN 90 WHEN building_profile_build_year >= 1990 THEN 75 WHEN building_profile_build_year >= 1970 THEN 55 ELSE 40 END,
        jsonb_build_array(COALESCE(building_profile_build_year::text, 'build year unknown'))
    FROM linked
    WHERE physical_building_id IS NOT NULL
    UNION ALL
    SELECT 'physical_building', physical_building_id, 'energy_operating_cost',
        CASE
            WHEN building_profile_energy_class ~* '^[ABC]' THEN 85
            WHEN building_profile_energy_class ~* '^[D]' THEN 65
            WHEN building_profile_energy_class ~* '^[EFG]' THEN 35
            WHEN lower(COALESCE(building_profile_heating_method, '')) LIKE '%maalämp%' OR lower(COALESCE(building_profile_heating_method, '')) LIKE '%geothermal%' THEN 80
            ELSE 50
        END,
        jsonb_build_array(COALESCE(building_profile_energy_class, 'energy class unknown'), COALESCE(building_profile_heating_method, 'heating unknown'))
    FROM linked
    WHERE physical_building_id IS NOT NULL
    UNION ALL
    SELECT 'physical_building', physical_building_id, 'accessibility',
        CASE
            WHEN building_profile_elevator IS TRUE OR apartment_profile_accessibility = 'good' THEN 85
            WHEN apartment_profile_accessibility = 'poor' THEN 30
            ELSE 50
        END,
        jsonb_build_array(CASE WHEN building_profile_elevator IS TRUE THEN 'elevator' WHEN building_profile_elevator IS FALSE THEN 'no elevator' ELSE 'elevator unknown' END, COALESCE(apartment_profile_accessibility, 'accessibility unknown'))
    FROM linked
    WHERE physical_building_id IS NOT NULL
    UNION ALL
    SELECT 'physical_building', physical_building_id, 'repair_backlog',
        CASE housing_company_profile_repair_backlog_risk WHEN 'low' THEN 85 WHEN 'medium' THEN 55 WHEN 'high' THEN 25 ELSE 50 END,
        jsonb_build_array(COALESCE(housing_company_profile_repair_backlog_risk, 'repair backlog unknown'))
    FROM linked
    WHERE physical_building_id IS NOT NULL
    UNION ALL
    SELECT 'housing_company', housing_company_id, 'financial_health',
        CASE housing_company_profile_financial_risk WHEN 'low' THEN 85 WHEN 'medium' THEN 55 WHEN 'high' THEN 25 ELSE 50 END,
        jsonb_build_array(COALESCE(housing_company_profile_financial_risk, 'financial risk unknown'))
    FROM linked
    WHERE housing_company_id IS NOT NULL
    UNION ALL
    SELECT 'housing_company', housing_company_id, 'charge_pressure',
        CASE housing_company_profile_maintenance_risk WHEN 'low' THEN 85 WHEN 'medium' THEN 55 WHEN 'high' THEN 25 ELSE 50 END,
        jsonb_build_array(COALESCE(housing_company_profile_maintenance_risk, 'maintenance risk unknown'))
    FROM linked
    WHERE housing_company_id IS NOT NULL
    UNION ALL
    SELECT 'housing_company', housing_company_id, 'plot_tenure_risk',
        CASE
            WHEN lower(COALESCE(housing_company_profile_plot_ownership_type, '')) LIKE '%own%' OR lower(COALESCE(housing_company_profile_plot_ownership_type, '')) LIKE '%oma%' THEN 85
            WHEN lower(COALESCE(housing_company_profile_plot_ownership_type, '')) LIKE '%rent%' OR lower(COALESCE(housing_company_profile_plot_ownership_type, '')) LIKE '%vuokra%' THEN 45
            ELSE 50
        END,
        jsonb_build_array(COALESCE(housing_company_profile_plot_ownership_type, 'plot tenure unknown'))
    FROM linked
    WHERE housing_company_id IS NOT NULL
    UNION ALL
    SELECT 'housing_company', housing_company_id, 'administrative_legal_risk',
        50,
        jsonb_build_array('manager certificate restrictions not loaded')
    FROM linked
    WHERE housing_company_id IS NOT NULL
    UNION ALL
    SELECT 'housing_company', housing_company_id, 'document_freshness',
        25,
        jsonb_build_array('manager certificate not loaded')
    FROM linked
    WHERE housing_company_id IS NOT NULL
    UNION ALL
    SELECT 'property_offering', property_offering_id, 'comparable_support',
        CASE
            WHEN sale_listing_prices_transaction_match_score >= 95 THEN 90
            WHEN sale_listing_prices_transaction_match_score >= 85 THEN 75
            WHEN prices_transaction_price IS NOT NULL THEN 65
            ELSE 25
        END,
        jsonb_build_array(COALESCE('transaction match score ' || sale_listing_prices_transaction_match_score::text, 'no matched transaction'))
    FROM linked
    WHERE property_offering_id IS NOT NULL
    UNION ALL
    SELECT 'property_offering', property_offering_id, 'market_liquidity',
        CASE WHEN prices_transaction_price IS NOT NULL THEN 65 ELSE 35 END,
        jsonb_build_array(CASE WHEN prices_transaction_price IS NOT NULL THEN 'transaction anchor available' ELSE 'transaction anchor missing' END)
    FROM linked
    WHERE property_offering_id IS NOT NULL
    UNION ALL
    SELECT 'property_offering', property_offering_id, 'price_attractiveness',
        CASE
            WHEN prices_transaction_price IS NULL OR COALESCE(sale_listing_debt_free_price, sale_listing_asking_price) IS NULL THEN 50
            WHEN COALESCE(sale_listing_debt_free_price, sale_listing_asking_price) <= prices_transaction_price * 0.95 THEN 85
            WHEN COALESCE(sale_listing_debt_free_price, sale_listing_asking_price) <= prices_transaction_price * 1.03 THEN 65
            WHEN COALESCE(sale_listing_debt_free_price, sale_listing_asking_price) <= prices_transaction_price * 1.10 THEN 40
            ELSE 20
        END,
        jsonb_build_array(COALESCE('asking ' || COALESCE(sale_listing_debt_free_price, sale_listing_asking_price)::text, 'asking missing'), COALESCE('transaction ' || prices_transaction_price::text, 'transaction missing'))
    FROM linked
    WHERE property_offering_id IS NOT NULL
    UNION ALL
    SELECT 'property_offering', property_offering_id, 'renovation_adjusted_value',
        CASE housing_company_profile_repair_backlog_risk WHEN 'high' THEN 30 WHEN 'medium' THEN 50 WHEN 'low' THEN 75 ELSE 50 END,
        jsonb_build_array(COALESCE(housing_company_profile_repair_backlog_risk, 'repair backlog unknown'))
    FROM linked
    WHERE property_offering_id IS NOT NULL
)
INSERT INTO public.property_quality_scores (
    property_quality_score_target_type,
    property_quality_score_target_id,
    property_quality_score_dimension,
    property_quality_score_value,
    property_quality_score_confidence,
    property_quality_score_reasons,
    property_quality_score_updated_at
)
SELECT
    target_type,
    target_id,
    dimension,
    score,
    'medium',
    reasons,
    now()
FROM scores
WHERE target_id IS NOT NULL
ON CONFLICT (
    property_quality_score_target_type,
    property_quality_score_target_id,
    property_quality_score_dimension
) DO UPDATE SET
    property_quality_score_value = EXCLUDED.property_quality_score_value,
    property_quality_score_confidence = EXCLUDED.property_quality_score_confidence,
    property_quality_score_reasons = EXCLUDED.property_quality_score_reasons,
    property_quality_score_updated_at = now();

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
    property_source_offering_insight_key,
    property_source_offering_insight_value,
    property_source_offering_insight_direction,
    property_source_offering_insight_severity,
    property_source_offering_insight_confidence,
    property_source_offering_insight_source_field,
    COALESCE(property_source_offering_insight_text, '') AS property_source_offering_insight_text
FROM public.property_source_offering_insights
WHERE sale_listing_id = sqlc.arg(sale_listing_id)
ORDER BY property_source_offering_insight_severity DESC, property_source_offering_insight_key;

-- name: ListPropertyClaimsForEntity :many
SELECT
    COALESCE(property_claim_source_path, property_claim_method) AS property_claim_source_field,
    property_claim_namespace,
    property_claim_key,
    property_claim_value_kind,
    COALESCE(property_claim_value_text, '') AS property_claim_value_text,
    property_claim_value_number,
    property_claim_value_bool,
    round(property_claim_confidence * 100)::integer AS property_claim_confidence,
    COALESCE(property_claim_evidence_text, '') AS property_claim_evidence_text,
    COALESCE(property_claim_model, '') AS property_claim_model,
    COALESCE(property_claim_prompt_version, '') AS property_claim_prompt_version
FROM public.property_claims
WHERE property_claim_target_type = sqlc.arg(entity_type)
    AND property_claim_target_id = sqlc.arg(entity_id)
ORDER BY property_claim_namespace, property_claim_key;

-- name: DeleteLLMPropertySourceOfferingInsights :exec
DELETE FROM public.property_source_offering_insights
WHERE sale_listing_id = sqlc.arg(sale_listing_id)
    AND property_source_offering_insight_source_field LIKE 'llm_%';

-- name: DeleteLLMPropertyClaimsForEntity :exec
DELETE FROM public.property_claims
WHERE property_claim_target_type = sqlc.arg(entity_type)
    AND property_claim_target_id = sqlc.arg(entity_id)
    AND property_claim_method = 'llm';

-- name: InsertPropertySourceOfferingInsight :exec
INSERT INTO public.property_source_offering_insights (
    sale_listing_id,
    property_source_offering_insight_source_field,
    property_source_offering_insight_key,
    property_source_offering_insight_value,
    property_source_offering_insight_direction,
    property_source_offering_insight_severity,
    property_source_offering_insight_confidence,
    property_source_offering_insight_text
) VALUES (
    sqlc.arg(sale_listing_id),
    sqlc.arg(source_field),
    sqlc.arg(key),
    sqlc.arg(value),
    sqlc.arg(direction),
    sqlc.arg(severity),
    sqlc.arg(confidence),
    NULLIF(sqlc.arg(text), '')
);

-- name: InsertPropertyClaim :exec
INSERT INTO public.property_claims (
    property_claim_target_type,
    property_claim_target_id,
    property_claim_namespace,
    property_claim_key,
    property_claim_value_kind,
    property_claim_value_text,
    property_claim_value_number,
    property_claim_value_bool,
    property_claim_source_record_table,
    property_claim_source_record_id,
    property_claim_source_path,
    property_claim_evidence_text,
    property_claim_method,
    property_claim_confidence,
    property_claim_source_reliability,
    property_claim_model,
    property_claim_prompt_version,
    property_claim_observed_at
) VALUES (
    sqlc.arg(entity_type),
    sqlc.arg(entity_id),
    sqlc.arg(section),
    sqlc.arg(key),
    sqlc.arg(value_kind),
    NULLIF(sqlc.arg(value_text), ''),
    sqlc.narg(value_number),
    sqlc.narg(value_bool),
    'property_source_offerings',
    sqlc.arg(entity_id),
    NULLIF(sqlc.arg(source_field), ''),
    NULLIF(sqlc.arg(evidence_text), ''),
    'llm',
    GREATEST(0, LEAST(1, sqlc.arg(confidence)::double precision / 100)),
    0.65,
    NULLIF(sqlc.arg(model), ''),
    NULLIF(sqlc.arg(prompt_version), ''),
    now()
) ON CONFLICT (
    property_claim_target_type,
    property_claim_target_id,
    property_claim_namespace,
    property_claim_key,
    property_claim_source_record_table,
    property_claim_source_record_id,
    COALESCE(property_claim_source_path, ''),
    property_claim_method
) DO UPDATE SET
    property_claim_value_kind = EXCLUDED.property_claim_value_kind,
    property_claim_value_text = EXCLUDED.property_claim_value_text,
    property_claim_value_number = EXCLUDED.property_claim_value_number,
    property_claim_value_bool = EXCLUDED.property_claim_value_bool,
    property_claim_evidence_text = EXCLUDED.property_claim_evidence_text,
    property_claim_confidence = EXCLUDED.property_claim_confidence,
    property_claim_source_reliability = EXCLUDED.property_claim_source_reliability,
    property_claim_model = EXCLUDED.property_claim_model,
    property_claim_prompt_version = EXCLUDED.property_claim_prompt_version,
    property_claim_observed_at = EXCLUDED.property_claim_observed_at,
    property_claim_updated_at = now();

-- name: SearchUnifiedEntities :many
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        sa.shortcut_ad_id::text AS native_id,
        ('shortcut:ad:' || sa.shortcut_ad_id::text) AS canonical_id,
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
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        sl.sale_listing_native_id AS native_id,
        sl.sale_listing_id::text AS canonical_id,
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
    WHERE sl.frontdoor_building_announcement_id IS NOT NULL
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'building'::text AS kind,
        fb.frontdoor_building_id::text AS native_id,
        ('frontdoor:building:' || fb.frontdoor_building_id::text) AS canonical_id,
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
      AND (sqlc.narg('published_after')::timestamptz IS NULL OR u.published_at >= sqlc.narg('published_after')::timestamptz)
      AND (sqlc.narg('published_before')::timestamptz IS NULL OR u.published_at <= sqlc.narg('published_before')::timestamptz)
)
SELECT
    source,
    kind,
    native_id,
    canonical_id::text AS canonical_id,
    headline,
    address,
    city,
    postal,
    price,
    area,
    room_layout::text AS room_layout,
    url,
    last_seen_at
FROM filtered
ORDER BY
    CASE WHEN sqlc.arg(sort_mode) = 'price_asc' THEN price END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'price_desc' THEN price END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'area_asc' THEN area END ASC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'area_desc' THEN area END DESC NULLS LAST,
    CASE WHEN sqlc.arg(sort_mode) = 'seen_desc' THEN last_seen_at END DESC NULLS LAST,
    last_seen_at DESC,
    source,
    kind,
    native_id
LIMIT sqlc.arg(limit_count)::int
OFFSET sqlc.arg(offset_count)::int;

-- name: CountUnifiedEntities :one
WITH unified AS (
    SELECT
        'shortcut'::text AS source,
        'ad'::text AS kind,
        raw.city::text AS city,
        raw.postal::text AS postal,
        raw.price::bigint AS price,
        COALESCE(raw.area, 0::float8)::float8 AS area,
        trim(concat_ws(' ', sa.shortcut_ad_id::text, sa.shortcut_ad_url, raw.street_address, raw.city, raw.postal, sa.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)) AS searchable,
        sa.shortcut_ad_type AS listing_type,
        (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz AS published_at
    FROM public.shortcut_ads sa
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
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
        sl.sale_listing_city AS city,
        sl.sale_listing_postal AS postal,
        sl.sale_listing_asking_price AS price,
        COALESCE(sl.sale_listing_area_value, 0::float8) AS area,
        sl.sale_listing_search_text AS searchable,
        NULL::text AS listing_type,
        sl.sale_listing_published_at AS published_at
    FROM public.frontdoor_ads fa
    JOIN public.property_source_offerings sl ON sl.frontdoor_ad_id = fa.frontdoor_ad_id
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'announcement'::text AS kind,
        sl.sale_listing_city AS city,
        sl.sale_listing_postal AS postal,
        sl.sale_listing_asking_price AS price,
        COALESCE(sl.sale_listing_area_value, 0::float8) AS area,
        sl.sale_listing_search_text AS searchable,
        NULL::text AS listing_type,
        sl.sale_listing_published_at AS published_at
    FROM public.property_source_offerings sl
    WHERE sl.frontdoor_building_announcement_id IS NOT NULL
    UNION ALL
    SELECT
        'frontdoor'::text AS source,
        'building'::text AS kind,
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
    (CASE
        WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
        WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
        ELSE NULL
    END)::boolean AS frontdoor_ad_sauna,
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
