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
    property_source_offering_renovation_text,
    property_source_offering_renovation_confidence
)
SELECT
    listing.sale_listing_id,
    renovation.source_field,
    renovation.category,
    'done',
    renovation.year,
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
    COALESCE(sl.sale_listing_renovations_done_text, NULLIF(trim(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDone}'), '')) AS frontdoor_ad_renovations_done_text,
    COALESCE(sl.sale_listing_renovations_planned_text, NULLIF(trim(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlanned}'), '')) AS frontdoor_ad_renovations_planned_text,
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
