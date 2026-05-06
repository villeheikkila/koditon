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

-- name: GetSaleListingApartmentProfile :one
SELECT *
FROM public.sale_listing_apartment_profiles
WHERE sale_listing_id = sqlc.arg(sale_listing_id)
LIMIT 1;

-- name: ProjectSaleListingApartmentProfile :exec
WITH linked AS (
    SELECT
        pos.sale_listing_id,
        pu.property_unit_id,
        pu.housing_company_id
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
        linked.property_unit_id
    FROM public.property_source_offerings sl
    LEFT JOIN linked ON linked.sale_listing_id = sl.sale_listing_id
    WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
)
INSERT INTO public.sale_listing_apartment_profiles (
    sale_listing_id,
    housing_company_id,
    property_unit_id,
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
    apartment_profile_confidence,
    apartment_profile_updated_at
)
SELECT
    sale_listing_id,
    housing_company_id,
    property_unit_id,
    sale_listing_area_value,
    sale_listing_living_area_value,
    sale_listing_room_layout,
    sale_listing_rooms_count,
    sale_listing_bedrooms_count,
    sale_listing_floor_level,
    sale_listing_total_floors,
    CASE
        WHEN lower(COALESCE(sale_listing_room_layout, '')) LIKE '%avok%' OR lower(COALESCE(sale_listing_kitchen_description_text, '')) LIKE '%avokeitti%' THEN 'open'
        WHEN lower(COALESCE(sale_listing_room_layout, '')) LIKE '%kk%' OR lower(COALESCE(sale_listing_room_layout, '')) LIKE '%keittonurk%' THEN 'kitchenette'
        WHEN lower(COALESCE(sale_listing_room_layout, '')) LIKE '%k%' OR lower(COALESCE(sale_listing_kitchen_description_text, '')) LIKE '%erillinen%' THEN 'separate'
        ELSE NULL
    END,
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
    CASE
        WHEN sale_listing_area_value IS NOT NULL AND sale_listing_room_layout IS NOT NULL THEN 'medium'
        ELSE 'low'
    END,
    now()
FROM listing
ON CONFLICT (sale_listing_id) DO UPDATE SET
    housing_company_id = EXCLUDED.housing_company_id,
    property_unit_id = EXCLUDED.property_unit_id,
    apartment_profile_area_m2 = EXCLUDED.apartment_profile_area_m2,
    apartment_profile_living_area_m2 = EXCLUDED.apartment_profile_living_area_m2,
    apartment_profile_room_layout = EXCLUDED.apartment_profile_room_layout,
    apartment_profile_room_count = EXCLUDED.apartment_profile_room_count,
    apartment_profile_bedroom_count = EXCLUDED.apartment_profile_bedroom_count,
    apartment_profile_floor_level = EXCLUDED.apartment_profile_floor_level,
    apartment_profile_total_floors = EXCLUDED.apartment_profile_total_floors,
    apartment_profile_kitchen_type = COALESCE(public.sale_listing_apartment_profiles.apartment_profile_kitchen_type, EXCLUDED.apartment_profile_kitchen_type),
    apartment_profile_condition = EXCLUDED.apartment_profile_condition,
    apartment_profile_sauna = EXCLUDED.apartment_profile_sauna,
    apartment_profile_balcony = EXCLUDED.apartment_profile_balcony,
    apartment_profile_parking_type = EXCLUDED.apartment_profile_parking_type,
    apartment_profile_confidence = EXCLUDED.apartment_profile_confidence,
    apartment_profile_updated_at = now();

-- name: UpsertSaleListingApartmentProfileProviderFieldSources :exec
WITH source_values AS (
    SELECT *
    FROM public.property_source_offerings sl
    CROSS JOIN LATERAL (
        VALUES
            ('apartment_profile_area_m2', 'sale_listing_area_value', sl.sale_listing_area_value::text),
            ('apartment_profile_living_area_m2', 'sale_listing_living_area_value', sl.sale_listing_living_area_value::text),
            ('apartment_profile_room_layout', 'sale_listing_room_layout', sl.sale_listing_room_layout),
            ('apartment_profile_room_count', 'sale_listing_rooms_count', sl.sale_listing_rooms_count::text),
            ('apartment_profile_bedroom_count', 'sale_listing_bedrooms_count', sl.sale_listing_bedrooms_count::text),
            ('apartment_profile_floor_level', 'sale_listing_floor_level', sl.sale_listing_floor_level::text),
            ('apartment_profile_total_floors', 'sale_listing_total_floors', sl.sale_listing_total_floors::text),
            ('apartment_profile_condition', 'sale_listing_condition', sl.sale_listing_condition),
            ('apartment_profile_sauna', 'sale_listing_sauna', sl.sale_listing_sauna::text),
            ('apartment_profile_balcony', 'sale_listing_balcony', sl.sale_listing_balcony::text),
            ('apartment_profile_parking_type', 'sale_listing_parking_text', sl.sale_listing_parking_text)
    ) AS source(target_field, source_path, evidence_text)
    WHERE sl.sale_listing_id = sqlc.arg(sale_listing_id)
        AND source.evidence_text IS NOT NULL
        AND trim(source.evidence_text) <> ''
)
INSERT INTO public.field_sources (
    field_source_target_table,
    field_source_target_id,
    field_source_target_field,
    field_source_source_record_table,
    field_source_source_record_id,
    field_source_source_path,
    field_source_evidence_text,
    field_source_method,
    field_source_confidence,
    field_source_observed_at
)
SELECT
    'sale_listing_apartment_profiles',
    sale_listing_id,
    target_field,
    'property_source_offerings',
    sale_listing_id,
    source_path,
    evidence_text,
    'provider_field',
    1,
    COALESCE(sale_listing_last_seen_at, sale_listing_first_seen_at, sale_listing_published_at, now())
FROM source_values
ON CONFLICT (
    field_source_target_table,
    field_source_target_id,
    field_source_target_field,
    field_source_source_record_table,
    field_source_source_record_id,
    COALESCE(field_source_source_path, ''),
    field_source_method
) DO UPDATE SET
    field_source_evidence_text = EXCLUDED.field_source_evidence_text,
    field_source_confidence = EXCLUDED.field_source_confidence,
    field_source_observed_at = EXCLUDED.field_source_observed_at;

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

-- name: ProjectSaleListingApartmentProfileLLMFacts :exec
WITH linked AS (
    SELECT
        pos.sale_listing_id,
        pu.property_unit_id,
        pu.housing_company_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE pos.sale_listing_id = sqlc.arg(sale_listing_id)
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1
),
facts AS (
    SELECT
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section IN ('balcony','unit') AND property_valuation_fact_key IN ('glazing','balcony_glazing')) AS balcony_glazing,
        max(property_valuation_fact_value_text) FILTER (WHERE property_valuation_fact_section = 'layout' AND property_valuation_fact_key = 'kitchen_type' AND property_valuation_fact_value_text = ANY (ARRAY['separate','open','kitchenette','unknown']::text[])) AS kitchen_type,
        max(property_valuation_fact_value_text) FILTER (WHERE property_valuation_fact_section = 'layout' AND property_valuation_fact_key = 'layout_quality' AND property_valuation_fact_value_text = ANY (ARRAY['weak','average','good','excellent','unknown']::text[])) AS layout_quality,
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section = 'layout' AND property_valuation_fact_key = 'awkward_layout') AS awkward_layout,
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section IN ('condition','unit') AND property_valuation_fact_key = 'surface_renovation_need') AS surface_renovation_need,
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section IN ('condition','unit') AND property_valuation_fact_key = 'modernization_need') AS modernization_need,
        max(property_valuation_fact_value_text) FILTER (WHERE property_valuation_fact_section = 'storage' AND property_valuation_fact_key = 'storage_quality' AND property_valuation_fact_value_text = ANY (ARRAY['weak','normal','good','unknown']::text[])) AS storage_quality,
        max(property_valuation_fact_value_text) FILTER (WHERE property_valuation_fact_section = 'views' AND property_valuation_fact_key = 'view_quality' AND property_valuation_fact_value_text = ANY (ARRAY['weak','normal','good','excellent','unknown']::text[])) AS view_quality,
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section = 'views' AND property_valuation_fact_key = 'noise_risk') AS noise_risk,
        max(property_valuation_fact_value_text) FILTER (WHERE property_valuation_fact_section IN ('building','unit') AND property_valuation_fact_key = 'accessibility' AND property_valuation_fact_value_text = ANY (ARRAY['poor','average','good','unknown']::text[])) AS accessibility,
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section = 'kitchen' AND property_valuation_fact_key = 'renovated') AS kitchen_renovated,
        bool_or(property_valuation_fact_value_bool) FILTER (WHERE property_valuation_fact_section = 'bathroom' AND property_valuation_fact_key = 'renovated') AS bathroom_renovated
    FROM public.property_valuation_facts
    WHERE property_valuation_fact_entity_type = 'sale_listing'
        AND property_valuation_fact_entity_id = sqlc.arg(sale_listing_id)
        AND property_valuation_fact_source_field LIKE 'llm_%'
)
INSERT INTO public.sale_listing_apartment_profiles (
    sale_listing_id,
    housing_company_id,
    property_unit_id,
    apartment_profile_balcony_glazing,
    apartment_profile_kitchen_type,
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
    sqlc.arg(sale_listing_id),
    linked.housing_company_id,
    linked.property_unit_id,
    facts.balcony_glazing,
    facts.kitchen_type,
    facts.layout_quality,
    facts.awkward_layout,
    facts.surface_renovation_need,
    facts.modernization_need,
    facts.storage_quality,
    facts.view_quality,
    facts.noise_risk,
    facts.accessibility,
    CASE WHEN facts.kitchen_renovated IS TRUE THEN 'good' ELSE NULL END,
    CASE WHEN facts.bathroom_renovated IS TRUE THEN 'good' ELSE NULL END,
    'medium',
    now()
FROM facts
LEFT JOIN linked ON true
ON CONFLICT (sale_listing_id) DO UPDATE SET
    housing_company_id = COALESCE(public.sale_listing_apartment_profiles.housing_company_id, EXCLUDED.housing_company_id),
    property_unit_id = COALESCE(public.sale_listing_apartment_profiles.property_unit_id, EXCLUDED.property_unit_id),
    apartment_profile_balcony_glazing = COALESCE(EXCLUDED.apartment_profile_balcony_glazing, public.sale_listing_apartment_profiles.apartment_profile_balcony_glazing),
    apartment_profile_kitchen_type = COALESCE(EXCLUDED.apartment_profile_kitchen_type, public.sale_listing_apartment_profiles.apartment_profile_kitchen_type),
    apartment_profile_layout_quality = COALESCE(EXCLUDED.apartment_profile_layout_quality, public.sale_listing_apartment_profiles.apartment_profile_layout_quality),
    apartment_profile_awkward_layout = COALESCE(EXCLUDED.apartment_profile_awkward_layout, public.sale_listing_apartment_profiles.apartment_profile_awkward_layout),
    apartment_profile_surface_renovation_need = COALESCE(EXCLUDED.apartment_profile_surface_renovation_need, public.sale_listing_apartment_profiles.apartment_profile_surface_renovation_need),
    apartment_profile_modernization_need = COALESCE(EXCLUDED.apartment_profile_modernization_need, public.sale_listing_apartment_profiles.apartment_profile_modernization_need),
    apartment_profile_storage_quality = COALESCE(EXCLUDED.apartment_profile_storage_quality, public.sale_listing_apartment_profiles.apartment_profile_storage_quality),
    apartment_profile_view_quality = COALESCE(EXCLUDED.apartment_profile_view_quality, public.sale_listing_apartment_profiles.apartment_profile_view_quality),
    apartment_profile_noise_risk = COALESCE(EXCLUDED.apartment_profile_noise_risk, public.sale_listing_apartment_profiles.apartment_profile_noise_risk),
    apartment_profile_accessibility = COALESCE(EXCLUDED.apartment_profile_accessibility, public.sale_listing_apartment_profiles.apartment_profile_accessibility),
    apartment_profile_kitchen_condition = COALESCE(EXCLUDED.apartment_profile_kitchen_condition, public.sale_listing_apartment_profiles.apartment_profile_kitchen_condition),
    apartment_profile_bathroom_condition = COALESCE(EXCLUDED.apartment_profile_bathroom_condition, public.sale_listing_apartment_profiles.apartment_profile_bathroom_condition),
    apartment_profile_confidence = CASE WHEN public.sale_listing_apartment_profiles.apartment_profile_confidence = 'high' THEN 'high' ELSE 'medium' END,
    apartment_profile_updated_at = now();

-- name: UpsertSaleListingApartmentProfileLLMFieldSources :exec
WITH facts AS (
    SELECT
        property_valuation_fact_entity_id AS sale_listing_id,
        property_valuation_fact_source_field,
        property_valuation_fact_evidence_text,
        property_valuation_fact_confidence,
        CASE
            WHEN property_valuation_fact_section IN ('balcony','unit') AND property_valuation_fact_key IN ('glazing','balcony_glazing') THEN 'apartment_profile_balcony_glazing'
            WHEN property_valuation_fact_section = 'layout' AND property_valuation_fact_key = 'kitchen_type' THEN 'apartment_profile_kitchen_type'
            WHEN property_valuation_fact_section = 'layout' AND property_valuation_fact_key = 'layout_quality' THEN 'apartment_profile_layout_quality'
            WHEN property_valuation_fact_section = 'layout' AND property_valuation_fact_key = 'awkward_layout' THEN 'apartment_profile_awkward_layout'
            WHEN property_valuation_fact_section IN ('condition','unit') AND property_valuation_fact_key = 'surface_renovation_need' THEN 'apartment_profile_surface_renovation_need'
            WHEN property_valuation_fact_section IN ('condition','unit') AND property_valuation_fact_key = 'modernization_need' THEN 'apartment_profile_modernization_need'
            WHEN property_valuation_fact_section = 'storage' AND property_valuation_fact_key = 'storage_quality' THEN 'apartment_profile_storage_quality'
            WHEN property_valuation_fact_section = 'views' AND property_valuation_fact_key = 'view_quality' THEN 'apartment_profile_view_quality'
            WHEN property_valuation_fact_section = 'views' AND property_valuation_fact_key = 'noise_risk' THEN 'apartment_profile_noise_risk'
            WHEN property_valuation_fact_section IN ('building','unit') AND property_valuation_fact_key = 'accessibility' THEN 'apartment_profile_accessibility'
            WHEN property_valuation_fact_section = 'kitchen' AND property_valuation_fact_key = 'renovated' THEN 'apartment_profile_kitchen_condition'
            WHEN property_valuation_fact_section = 'bathroom' AND property_valuation_fact_key = 'renovated' THEN 'apartment_profile_bathroom_condition'
            ELSE NULL
        END AS target_field
    FROM public.property_valuation_facts
    WHERE property_valuation_fact_entity_type = 'sale_listing'
        AND property_valuation_fact_entity_id = sqlc.arg(sale_listing_id)
        AND property_valuation_fact_source_field LIKE 'llm_%'
)
INSERT INTO public.field_sources (
    field_source_target_table,
    field_source_target_id,
    field_source_target_field,
    field_source_source_record_table,
    field_source_source_record_id,
    field_source_source_path,
    field_source_evidence_text,
    field_source_method,
    field_source_confidence,
    field_source_observed_at
)
SELECT
    'sale_listing_apartment_profiles',
    sale_listing_id,
    target_field,
    'property_source_offerings',
    sale_listing_id,
    property_valuation_fact_source_field,
    property_valuation_fact_evidence_text,
    'llm',
    property_valuation_fact_confidence::double precision / 100,
    now()
FROM facts
WHERE target_field IS NOT NULL
ON CONFLICT (
    field_source_target_table,
    field_source_target_id,
    field_source_target_field,
    field_source_source_record_table,
    field_source_source_record_id,
    COALESCE(field_source_source_path, ''),
    field_source_method
) DO UPDATE SET
    field_source_evidence_text = EXCLUDED.field_source_evidence_text,
    field_source_confidence = EXCLUDED.field_source_confidence,
    field_source_observed_at = EXCLUDED.field_source_observed_at;

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

-- name: ListPropertyValuationFactsForEntity :many
SELECT
    property_valuation_fact_source_field,
    property_valuation_fact_section,
    property_valuation_fact_key,
    property_valuation_fact_value_kind,
    COALESCE(property_valuation_fact_value_text, '') AS property_valuation_fact_value_text,
    property_valuation_fact_value_number,
    property_valuation_fact_value_bool,
    property_valuation_fact_confidence,
    COALESCE(property_valuation_fact_evidence_text, '') AS property_valuation_fact_evidence_text,
    COALESCE(property_valuation_fact_model, '') AS property_valuation_fact_model,
    COALESCE(property_valuation_fact_prompt_version, '') AS property_valuation_fact_prompt_version
FROM public.property_valuation_facts
WHERE property_valuation_fact_entity_type = sqlc.arg(entity_type)
    AND property_valuation_fact_entity_id = sqlc.arg(entity_id)
ORDER BY property_valuation_fact_section, property_valuation_fact_key;

-- name: DeleteLLMPropertySourceOfferingInsights :exec
DELETE FROM public.property_source_offering_insights
WHERE sale_listing_id = sqlc.arg(sale_listing_id)
    AND property_source_offering_insight_source_field LIKE 'llm_%';

-- name: DeleteLLMPropertyValuationFactsForEntity :exec
DELETE FROM public.property_valuation_facts
WHERE property_valuation_fact_entity_type = sqlc.arg(entity_type)
    AND property_valuation_fact_entity_id = sqlc.arg(entity_id)
    AND property_valuation_fact_source_field LIKE 'llm_%';

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

-- name: InsertPropertyValuationFact :exec
INSERT INTO public.property_valuation_facts (
    property_valuation_fact_entity_type,
    property_valuation_fact_entity_id,
    property_valuation_fact_source_field,
    property_valuation_fact_section,
    property_valuation_fact_key,
    property_valuation_fact_value_kind,
    property_valuation_fact_value_text,
    property_valuation_fact_value_number,
    property_valuation_fact_value_bool,
    property_valuation_fact_confidence,
    property_valuation_fact_evidence_text,
    property_valuation_fact_model,
    property_valuation_fact_prompt_version
) VALUES (
    sqlc.arg(entity_type),
    sqlc.arg(entity_id),
    sqlc.arg(source_field),
    sqlc.arg(section),
    sqlc.arg(key),
    sqlc.arg(value_kind),
    NULLIF(sqlc.arg(value_text), ''),
    sqlc.narg(value_number),
    sqlc.narg(value_bool),
    sqlc.arg(confidence),
    NULLIF(sqlc.arg(evidence_text), ''),
    NULLIF(sqlc.arg(model), ''),
    NULLIF(sqlc.arg(prompt_version), '')
) ON CONFLICT (
    property_valuation_fact_entity_type,
    property_valuation_fact_entity_id,
    property_valuation_fact_source_field,
    property_valuation_fact_section,
    property_valuation_fact_key
) DO UPDATE SET
    property_valuation_fact_value_kind = EXCLUDED.property_valuation_fact_value_kind,
    property_valuation_fact_value_text = EXCLUDED.property_valuation_fact_value_text,
    property_valuation_fact_value_number = EXCLUDED.property_valuation_fact_value_number,
    property_valuation_fact_value_bool = EXCLUDED.property_valuation_fact_value_bool,
    property_valuation_fact_confidence = EXCLUDED.property_valuation_fact_confidence,
    property_valuation_fact_evidence_text = EXCLUDED.property_valuation_fact_evidence_text,
    property_valuation_fact_model = EXCLUDED.property_valuation_fact_model,
    property_valuation_fact_prompt_version = EXCLUDED.property_valuation_fact_prompt_version,
    property_valuation_fact_updated_at = now();

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
