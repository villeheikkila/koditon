-- name: DeleteSaleListingForShortcutAd :exec
WITH deleted AS (
    DELETE FROM public.property_source_offerings
    WHERE shortcut_ad_id = sqlc.arg(shortcut_ad_id)
    RETURNING sale_listing_id
)
DELETE FROM origin.source_listings sl
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
    COALESCE(raw.price_per_m2, CASE WHEN raw.price IS NOT NULL AND raw.area IS NOT NULL AND raw.area > 0 THEN raw.price::double precision / raw.area ELSE NULL END),
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
FROM origin.shortcut_ads sa
LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
CROSS JOIN LATERAL (
    SELECT
        COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')) AS street_address,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerDay}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceDebtFree}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS debt_free_price,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,debtShare}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS debt_share_amount,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,pricePerSqm}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price_per_m2,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,rooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{rooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS rooms_count,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,floor}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{floor}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS floor_level,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,totalFloors}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,floors}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS total_floors,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,year}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,constructionYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS build_year,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,condition}', sa.shortcut_ad_data #>> '{property,condition}')), '') AS condition,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}')), '') AS energy_class,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,description}', sa.shortcut_ad_data #>> '{description}', sa.shortcut_ad_data #>> '{text}')), '') AS description_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,availabilityDescription}', sa.shortcut_ad_data #>> '{availabilityDescription}', sa.shortcut_ad_data #>> '{adData,availableFrom}')), '') AS availability_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{property,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{adData,renovationInfo}', sa.shortcut_ad_data #>> '{buildingData,renovationInfo}')), '') AS renovations_done_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{property,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{adData,renovationFutureInfo}', sa.shortcut_ad_data #>> '{buildingData,renovationFutureInfo}')), '') AS renovations_planned_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,additionalInfo}', sa.shortcut_ad_data #>> '{moreInformationAvailableFrom}', sa.shortcut_ad_data #>> '{property,otherInfo}')), '') AS additional_info_text,
        NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{priceData,chargesText}', sa.shortcut_ad_data #>> '{priceData,additionalInfo}', sa.shortcut_ad_data #>> '{property,periodicChargesAdditionalInfo}', sa.shortcut_ad_data #>> '{property,managementChargesAdditionalInfo}')), '') AS charges_text,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,maintenanceCharge}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,monthlyFee}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS maintenance_charge_monthly,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,totalCharge}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,monthlyFee}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS total_charge_monthly,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,waterFee}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS water_charge,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS living_area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS total_area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeOther}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS other_area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,bedrooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS bedrooms_count,
        COALESCE(CASE WHEN sa.shortcut_ad_data #>> '{adData,sauna}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,sauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,sauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END, CASE WHEN sa.shortcut_ad_data #>> '{adData,hasSauna}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasSauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasSauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END) AS sauna,
        CASE WHEN sa.shortcut_ad_data #>> '{adData,balcony}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,balcony}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,balcony}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END AS balcony,
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
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,plotArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,plotArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS plot_area,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,servicesInfo}'), '') AS services_text,
        NULLIF(trim(sa.shortcut_ad_data #>> '{adData,connectionsInfo}'), '') AS transport_text,
        CASE WHEN sa.shortcut_ad_data #>> '{adData,newDevelopment}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,newDevelopment}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,newDevelopment}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END AS new_development
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
INSERT INTO origin.source_listings (
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
LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
WHERE sl.sale_listing_id = @sale_listing_id
ON CONFLICT (canonical_source_id) DO UPDATE SET
    source_listing_id = EXCLUDED.source_listing_id,
    provider = EXCLUDED.provider,
    source_kind = EXCLUDED.source_kind,
    native_id = EXCLUDED.native_id,
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
    CASE
        WHEN (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{publishingTime}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) IS NULL THEN NULL
        ELSE to_timestamp((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{publishingTime}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) / 1000.0)
    END,
    trim(concat_ws(' ', fa.frontdoor_ad_external_id, fa.frontdoor_ad_url, raw.street_address, raw.city, raw.postal, fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}')),
    COALESCE(raw.price_per_m2, CASE WHEN raw.price IS NOT NULL AND raw.area IS NOT NULL AND raw.area > 0 THEN raw.price::double precision / raw.area ELSE NULL END),
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
FROM origin.frontdoor_ads fa
CROSS JOIN LATERAL (
    SELECT
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', fa.frontdoor_ad_data #>> '{property,address}', fa.frontdoor_ad_data #>> '{property,streetNameFreeForm}')), '') AS street_address,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,municipalityNameFreeForm}', fa.frontdoor_ad_data #>> '{property,municipality}', fa.frontdoor_ad_data #>> '{property,city}', fa.frontdoor_ad_data #>> '{property,postCode,postArea}')), '') AS city,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,postalCode}', fa.frontdoor_ad_data #>> '{property,addressPostalCode}', fa.frontdoor_ad_data #>> '{property,postCode,postCode}')), '') AS postal,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{debfFreePrice}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{preparsed,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{preparsed,area}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,livingArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{debfFreePrice}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS debt_free_price,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{debtShareAmount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS debt_share_amount,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{preparsed,pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price_per_m2,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,totalRoomCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS rooms_count,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,housingCompanyApartmentInformationDTO,floorLevel}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,floorLevel}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS floor_level,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,floorCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,floorCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS total_floors,
        COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,constructionFinishedYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,usageStartYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS build_year,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,inspection,overallCondition}', fa.frontdoor_ad_data #>> '{property,condition}')), '') AS condition,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}')), '') AS energy_class,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{text}', fa.frontdoor_ad_data #>> '{property,description}')), '') AS description_text,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{availabilityDescription}'), '') AS availability_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDone}')), '') AS renovations_done_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlanned}')), '') AS renovations_planned_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{moreInformationAvailableFrom}', fa.frontdoor_ad_data #>> '{property,housingCompany,otherInfo}', fa.frontdoor_ad_data #>> '{additionalItemsIncludedInSale}')), '') AS additional_info_text,
        NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,periodicChargesAdditionalInfo}', fa.frontdoor_ad_data #>> '{property,managementChargesAdditionalInfo}')), '') AS charges_text,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(jsonb_path_query_first(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price', jsonb_build_object('charge', to_jsonb('HOUSING_COMPANY_MAINTENANCE_CHARGE'::text))) #>> '{}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS maintenance_charge_monthly,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(jsonb_path_query_first(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price', jsonb_build_object('charge', to_jsonb('HOUSING_COMPANY_TOTAL_CHARGE'::text))) #>> '{}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS total_charge_monthly,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(jsonb_path_query_first(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price', jsonb_build_object('charge', to_jsonb('WATER'::text))) #>> '{}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS water_charge,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,livingArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS living_area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,totalArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS total_area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,otherArea}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS other_area,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,bedroomCount}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS bedrooms_count,
        CASE
            WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
            WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
            ELSE CASE WHEN fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}' IS NULL THEN NULL WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,housingCompany,hasSauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END
        END AS sauna,
        COALESCE(CASE WHEN fa.frontdoor_ad_data #>> '{property,hasBalcony}' IS NULL THEN NULL WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,hasBalcony}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(fa.frontdoor_ad_data #>> '{property,hasBalcony}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END, CASE WHEN NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{residenceDetailsDTO,balconyDescription}', fa.frontdoor_ad_data #>> '{property,balconyDescription}')), '') IS NOT NULL THEN true ELSE NULL::boolean END) AS balcony,
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
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,area}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS plot_area,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{property,nearbyAmenitiesDescription}'), '') AS services_text,
        NULLIF(trim(fa.frontdoor_ad_data #>> '{property,transportationServicesDescription}'), '') AS transport_text,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{previousPrice}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS previous_asking_price,
        (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(fa.frontdoor_ad_data #>> '{previousDebtFreePrice}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value) AS previous_debt_free_price,
        CASE WHEN fa.frontdoor_ad_data #>> '{newProperty}' IS NULL THEN NULL WHEN lower(trim(fa.frontdoor_ad_data #>> '{newProperty}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(fa.frontdoor_ad_data #>> '{newProperty}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END AS new_development
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
DELETE FROM origin.source_listings sl
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
FROM origin.frontdoor_building_announcements fba
JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
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
    SELECT
        sl.sale_listing_id,
        fb.frontdoor_building_elevator_renovated,
        fb.frontdoor_building_elevator_renovated_year,
        fb.frontdoor_building_facade_renovated,
        fb.frontdoor_building_facade_renovated_year,
        fb.frontdoor_building_window_renovated,
        fb.frontdoor_building_window_renovated_year,
        fb.frontdoor_building_roof_renovated,
        fb.frontdoor_building_roof_renovated_year,
        fb.frontdoor_building_pipe_renovated,
        fb.frontdoor_building_pipe_renovated_year,
        fb.frontdoor_building_balcony_renovated,
        fb.frontdoor_building_balcony_renovated_year,
        fb.frontdoor_building_electricity_renovated,
        fb.frontdoor_building_electricity_renovated_year
    FROM public.property_source_offerings sl
    JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
    JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
    WHERE sl.sale_listing_id = @sale_listing_id
),
deleted AS (
    DELETE FROM public.property_source_offering_renovations
    WHERE sale_listing_id = @sale_listing_id
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

-- name: GetSaleListingRenovationExtractionTexts :one
SELECT
    COALESCE(pso.sale_listing_renovations_done_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsDone}', sa.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{property,renovationsDoneDescription}')), ''), '')::text AS done_text,
    COALESCE(pso.sale_listing_renovations_planned_text, NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlanned}', sa.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{property,renovationsPlannedDescription}')), ''), '')::text AS planned_text
FROM public.property_source_offerings pso
LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = pso.frontdoor_ad_id
LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = pso.shortcut_ad_id
WHERE pso.sale_listing_id = @sale_listing_id
LIMIT 1;

-- name: DeleteLLMPropertySourceOfferingRenovations :exec
DELETE FROM public.property_source_offering_renovations
WHERE sale_listing_id = @sale_listing_id
    AND property_source_offering_renovation_source_field IN ('llm_renovations_done_text', 'llm_renovations_planned_text');

-- name: InsertLLMPropertySourceOfferingRenovation :exec
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
) VALUES (
    @sale_listing_id,
    sqlc.arg(source_field),
    sqlc.arg(category),
    sqlc.arg(status),
    sqlc.narg(year),
    NULLIF(sqlc.arg(component)::text, ''),
    NULLIF(sqlc.arg(scope)::text, ''),
    NULLIF(sqlc.arg(stage)::text, ''),
    NULLIF(sqlc.arg(responsibility)::text, ''),
    sqlc.narg(cost_estimate_eur),
    NULLIF(sqlc.arg(summary)::text, ''),
    sqlc.arg(confidence)
);

-- name: ProjectListingRenovationEvents :one
WITH run AS (
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    ) VALUES (
        'renovation_events',
        sqlc.arg(projection_version),
        'property_source_offerings',
        @sale_listing_id,
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id
),
deleted AS (
    DELETE FROM public.property_renovation_events
    WHERE event_scope = 'source'
        AND source_table = 'property_source_offerings'
        AND source_id = @sale_listing_id
        AND projection_version = sqlc.arg(projection_version)
),
linked AS (
    SELECT
        pos.sale_listing_id,
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id,
        pu.physical_building_id,
        pos.sale_listing_last_seen_at,
        pos.sale_listing_updated_at,
        pos.sale_listing_created_at
    FROM public.property_source_offerings pos
    LEFT JOIN public.target_sources link
        ON link.source_id = pos.sale_listing_id
        AND link.target_type = 'listing'
        AND link.source_type = 'source_listing'
        AND link.link_status <> 'rejected'
    LEFT JOIN public.property_offerings po ON po.property_offering_id = link.target_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE pos.sale_listing_id = @sale_listing_id
    ORDER BY link.link_score DESC NULLS LAST, link.updated_at DESC NULLS LAST
    LIMIT 1
),
inserted AS (
    INSERT INTO public.property_renovation_events (
        property_dimension_projection_run_id,
        projection_version,
        event_scope,
        target_type,
        target_id,
        source_table,
        source_id,
        source_field,
        category,
        component,
        status,
        stage,
        scope,
        responsibility,
        year,
        start_year,
        end_year,
        cost_estimate_eur,
        summary,
        evidence,
        confidence,
        source_reliability,
        source_observed_at
    )
    SELECT
        run.property_dimension_projection_run_id,
        sqlc.arg(projection_version),
        'source',
        CASE WHEN linked.housing_company_id IS NOT NULL THEN 'housing_company' ELSE 'building' END,
        COALESCE(linked.housing_company_id, linked.physical_building_id),
        'property_source_offerings',
        linked.sale_listing_id,
        renovation.property_source_offering_renovation_source_field,
        renovation.property_source_offering_renovation_category,
        NULLIF(renovation.property_source_offering_renovation_component, ''),
        renovation.property_source_offering_renovation_status,
        NULLIF(renovation.property_source_offering_renovation_stage, ''),
        NULLIF(renovation.property_source_offering_renovation_scope, ''),
        NULLIF(renovation.property_source_offering_renovation_responsibility, ''),
        renovation.property_source_offering_renovation_year,
        NULL,
        NULL,
        renovation.property_source_offering_renovation_cost_estimate_eur,
        NULLIF(renovation.property_source_offering_renovation_text, ''),
        jsonb_build_object('evidence_level', CASE WHEN renovation.property_source_offering_renovation_source_field LIKE 'llm_%' THEN 'listing_llm' ELSE 'listing_field' END),
        GREATEST(0, LEAST(1, COALESCE(renovation.property_source_offering_renovation_confidence, 50)::double precision / 100)),
        CASE WHEN renovation.property_source_offering_renovation_source_field LIKE 'llm_%' THEN 0.75 ELSE 0.65 END,
        COALESCE(linked.sale_listing_last_seen_at, linked.sale_listing_updated_at, linked.sale_listing_created_at, now())
    FROM run
    JOIN linked ON COALESCE(linked.housing_company_id, linked.physical_building_id) IS NOT NULL
    JOIN public.property_source_offering_renovations renovation ON renovation.sale_listing_id = linked.sale_listing_id
    ON CONFLICT (
        event_scope,
        target_type,
        target_id,
        source_table,
        source_id,
        COALESCE(source_field, ''),
        category,
        status,
        COALESCE(stage, ''),
        COALESCE(scope, ''),
        COALESCE(year, -1),
        COALESCE(start_year, -1),
        COALESCE(end_year, -1),
        md5(COALESCE(summary, '')),
        projection_version
    ) DO UPDATE SET
        component = EXCLUDED.component,
        responsibility = EXCLUDED.responsibility,
        cost_estimate_eur = EXCLUDED.cost_estimate_eur,
        confidence = EXCLUDED.confidence,
        source_reliability = EXCLUDED.source_reliability,
        source_observed_at = EXCLUDED.source_observed_at,
        evidence = EXCLUDED.evidence
    RETURNING 1
)
SELECT count(*)::bigint AS projected
FROM inserted;

-- name: CreateManagerCertificateRenovationProjectionRun :one
INSERT INTO public.property_dimension_projection_runs (
    projection_type,
    projection_version,
    source_table,
    source_id,
    status,
    finished_at
) VALUES (
    'renovation_events',
    'manager-certificate-renovations-v1',
    'property_documents',
    sqlc.arg(property_document_id),
    'succeeded',
    now()
)
RETURNING property_dimension_projection_run_id;

-- name: DeleteManagerCertificateRenovationEvents :exec
DELETE FROM public.property_renovation_events
WHERE event_scope = 'source'
    AND target_type = 'housing_company'
    AND target_id = sqlc.arg(housing_company_id)
    AND source_table = 'property_documents'
    AND source_id = sqlc.arg(property_document_id)
    AND projection_version = 'manager-certificate-renovations-v1';

-- name: InsertManagerCertificateRenovationEvent :exec
INSERT INTO public.property_renovation_events (
    property_dimension_projection_run_id,
    projection_version,
    event_scope,
    target_type,
    target_id,
    source_table,
    source_id,
    source_field,
    category,
    component,
    status,
    stage,
    scope,
    responsibility,
    year,
    start_year,
    end_year,
    cost_estimate_eur,
    summary,
    evidence,
    confidence,
    source_reliability,
    source_observed_at
) VALUES (
    sqlc.arg(property_dimension_projection_run_id),
    'manager-certificate-renovations-v1',
    'source',
    'housing_company',
    sqlc.arg(housing_company_id),
    'property_documents',
    sqlc.arg(property_document_id),
    'manager_certificate',
    sqlc.arg(category),
    NULL,
    sqlc.arg(status),
    sqlc.arg(stage),
    sqlc.arg(scope),
    sqlc.arg(responsibility),
    sqlc.narg(year),
    sqlc.narg(start_year),
    sqlc.narg(end_year),
    sqlc.narg(cost_estimate_eur),
    sqlc.arg(summary),
    jsonb_build_object('evidence_level', 'manager_certificate', 'source_label', NULLIF(sqlc.arg(source_label)::text, ''), 'action', NULLIF(sqlc.arg(action)::text, ''), 'evidence', NULLIF(sqlc.arg(evidence_text)::text, '')),
    0.9,
    0.9,
    sqlc.narg(source_observed_at)
)
ON CONFLICT (
    event_scope,
    target_type,
    target_id,
    source_table,
    source_id,
    COALESCE(source_field, ''),
    category,
    status,
    COALESCE(stage, ''),
    COALESCE(scope, ''),
    COALESCE(year, -1),
    COALESCE(start_year, -1),
    COALESCE(end_year, -1),
    md5(COALESCE(summary, '')),
    projection_version
) DO UPDATE SET
    responsibility = EXCLUDED.responsibility,
    start_year = EXCLUDED.start_year,
    end_year = EXCLUDED.end_year,
    cost_estimate_eur = EXCLUDED.cost_estimate_eur,
    confidence = EXCLUDED.confidence,
    source_observed_at = EXCLUDED.source_observed_at,
    evidence = EXCLUDED.evidence;

-- name: MarkPropertyOfferingDimensionTargetsDirty :one
WITH targets AS (
    SELECT 'offering'::text AS target_type, po.property_offering_id AS target_id
    FROM public.property_offerings po
    WHERE po.property_offering_id = sqlc.arg(property_offering_id)::uuid
    UNION
    SELECT 'unit', po.property_unit_id
    FROM public.property_offerings po
    WHERE po.property_offering_id = sqlc.arg(property_offering_id)::uuid
        AND po.property_unit_id IS NOT NULL
    UNION
    SELECT 'house', po.property_house_id
    FROM public.property_offerings po
    WHERE po.property_offering_id = sqlc.arg(property_offering_id)::uuid
        AND po.property_house_id IS NOT NULL
    UNION
    SELECT 'building', pu.physical_building_id
    FROM public.property_offerings po
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE po.property_offering_id = sqlc.arg(property_offering_id)::uuid
        AND pu.physical_building_id IS NOT NULL
    UNION
    SELECT 'housing_company', COALESCE(pu.housing_company_id, pb.housing_company_id)
    FROM public.property_offerings po
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE po.property_offering_id = sqlc.arg(property_offering_id)::uuid
        AND COALESCE(pu.housing_company_id, pb.housing_company_id) IS NOT NULL
    UNION
    SELECT 'listing', source_link.source_id
    FROM public.target_sources source_link
    WHERE source_link.target_type = 'listing'
        AND source_link.target_id = sqlc.arg(property_offering_id)::uuid
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
), marked AS (
    INSERT INTO public.property_dimension_dirty_targets (
        target_type,
        target_id,
        dirty_reasons,
        dirty_at,
        resolved_at
    )
    SELECT
        target_type,
        target_id,
        ARRAY[COALESCE(NULLIF(sqlc.arg(reason)::text, ''), 'changed')],
        now(),
        NULL::timestamptz
    FROM targets
    WHERE target_id IS NOT NULL
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = (
            SELECT array_agg(DISTINCT reason ORDER BY reason)
            FROM unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons) AS reason
        ),
        dirty_at = now(),
        resolved_at = NULL
    RETURNING 1
)
SELECT count(*)::integer AS count FROM marked;

-- name: RebuildListingDimensionLayer :one
SELECT jsonb_build_object('deprecated', false)::jsonb AS payload;

-- name: RebuildListingDimensionLayerAt :one
SELECT jsonb_build_object('deprecated', false)::jsonb AS payload;

-- name: ProjectListingProviderDimensionClaims :one
WITH deleted AS (
    DELETE FROM public.dimension_claims
    WHERE claim_scope = 'source'
        AND source_table = 'property_source_offerings'
        AND source_id = @sale_listing_id::uuid
        AND projection_version = 'listing-provider-v1'
),
run AS (
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
        'listing-provider-v1',
        'property_source_offerings',
        @sale_listing_id::uuid,
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id
),
inserted AS (
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
        evidence
    )
    SELECT
        run.property_dimension_projection_run_id,
        'listing-provider-v1',
        'source',
        'listing',
        sl.sale_listing_id,
        v.dimension_key,
        v.value,
        v.value_kind,
        c.unit,
        'property_source_offerings',
        sl.sale_listing_id,
        v.source_field,
        COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_updated_at, sl.sale_listing_created_at, now()),
        v.confidence,
        COALESCE(sp.default_reliability, v.source_reliability),
        jsonb_build_object('provider', sl.sale_listing_source_provider, 'source_kind', sl.sale_listing_source_kind)
    FROM public.property_source_offerings sl
    CROSS JOIN run
    CROSS JOIN LATERAL (
        VALUES
            ('sale_listing_area_value','unit.area_m2','number',to_jsonb(sl.sale_listing_area_value),0.95,0.75),
            ('sale_listing_living_area_value','unit.living_area_m2','number',to_jsonb(sl.sale_listing_living_area_value),0.95,0.75),
            ('sale_listing_total_area_value','unit.total_area_m2','number',to_jsonb(sl.sale_listing_total_area_value),0.9,0.75),
            ('sale_listing_other_area_value','unit.other_area_m2','number',to_jsonb(sl.sale_listing_other_area_value),0.9,0.75),
            ('sale_listing_room_layout','layout.room_layout','string',to_jsonb(NULLIF(sl.sale_listing_room_layout, '')),0.9,0.7),
            ('sale_listing_rooms_count','layout.room_count','number',to_jsonb(sl.sale_listing_rooms_count),0.95,0.75),
            ('sale_listing_bedrooms_count','layout.bedroom_count','number',to_jsonb(sl.sale_listing_bedrooms_count),0.85,0.7),
            ('sale_listing_floor_level','unit.floor_level','number',to_jsonb(sl.sale_listing_floor_level),0.9,0.7),
            ('sale_listing_total_floors','building.floor_count','number',to_jsonb(sl.sale_listing_total_floors),0.85,0.65),
            ('sale_listing_condition','condition.unit_condition','string',to_jsonb(NULLIF(sl.sale_listing_condition, '')),0.75,0.6),
            ('sale_listing_sauna','features.sauna','boolean',to_jsonb(sl.sale_listing_sauna),0.9,0.75),
            ('sale_listing_balcony','features.balcony','boolean',to_jsonb(sl.sale_listing_balcony),0.9,0.75),
            ('sale_listing_parking_text','features.parking_type','string',to_jsonb(NULLIF(sl.sale_listing_parking_text, '')),0.65,0.55),
            ('sale_listing_maintenance_charge_monthly','charges.maintenance_monthly_eur','number',to_jsonb(sl.sale_listing_maintenance_charge_monthly),0.9,0.7),
            ('sale_listing_total_charge_monthly','charges.total_monthly_eur','number',to_jsonb(sl.sale_listing_total_charge_monthly),0.9,0.7),
            ('sale_listing_water_charge','charges.water_monthly_eur','number',to_jsonb(sl.sale_listing_water_charge),0.8,0.6),
            ('sale_listing_debt_share_amount','charges.debt_share_eur','number',to_jsonb(sl.sale_listing_debt_share_amount),0.9,0.7),
            ('sale_listing_build_year','building.build_year','number',to_jsonb(sl.sale_listing_build_year),0.85,0.65),
            ('sale_listing_elevator','building.elevator','boolean',to_jsonb(sl.sale_listing_elevator),0.8,0.65),
            ('sale_listing_heating_system','building.heating_method','string',to_jsonb(NULLIF(sl.sale_listing_heating_system, '')),0.75,0.6),
            ('sale_listing_energy_efficiency_label','building.energy_class','string',to_jsonb(NULLIF(sl.sale_listing_energy_efficiency_label, '')),0.75,0.6),
            ('sale_listing_building_material','building.material','string',to_jsonb(NULLIF(sl.sale_listing_building_material, '')),0.75,0.6),
            ('sale_listing_roof_type','building.roof_type','string',to_jsonb(NULLIF(sl.sale_listing_roof_type, '')),0.75,0.6),
            ('sale_listing_roof_material','building.roof_material','string',to_jsonb(NULLIF(sl.sale_listing_roof_material, '')),0.75,0.6),
            ('sale_listing_apartment_count','housing_company.apartment_count','number',to_jsonb(sl.sale_listing_apartment_count),0.75,0.6),
            ('sale_listing_housing_company_name','housing_company.name','string',to_jsonb(NULLIF(sl.sale_listing_housing_company_name, '')),0.8,0.6),
            ('sale_listing_housing_company_business_id','housing_company.business_id','string',to_jsonb(NULLIF(sl.sale_listing_housing_company_business_id, '')),0.9,0.65),
            ('sale_listing_plot_type_code','site.plot_ownership_type','string',to_jsonb(NULLIF(sl.sale_listing_plot_type_code, '')),0.75,0.6)
    ) AS v(source_field, dimension_key, value_kind, value, confidence, source_reliability)
    JOIN public.property_dimension_catalog c ON c.dimension_key = v.dimension_key
    LEFT JOIN public.property_dimension_source_priorities sp
        ON sp.dimension_key = v.dimension_key
        AND sp.source_table = 'property_source_offerings'
        AND sp.source_field = v.source_field
    WHERE sl.sale_listing_id = @sale_listing_id::uuid
        AND v.value IS NOT NULL
    RETURNING 1
),
counts AS (
    SELECT count(*)::integer AS count FROM inserted
),
updated_run AS (
    UPDATE public.property_dimension_projection_runs
    SET result = jsonb_build_object('claim_count', counts.count)
    FROM run, counts
    WHERE property_dimension_projection_runs.property_dimension_projection_run_id = run.property_dimension_projection_run_id
)
SELECT count FROM counts;

-- name: ResolveDimensionValuesForTarget :one
WITH args AS (
    SELECT v.target_type, v.target_id FROM (VALUES ($1::text, $2::uuid)) AS v(target_type, target_id)
),
deleted AS (
    DELETE FROM public.dimension_values
    WHERE target_type = (SELECT target_type FROM args)
        AND target_id = (SELECT target_id FROM args)
),
run AS (
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    VALUES (
        'resolved_values',
        'dimension-resolver-v3',
        'dimension_claims',
        (SELECT target_id FROM args),
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id
),
linked_listings AS (
    SELECT DISTINCT
        source_link.source_id AS sale_listing_id,
        po.property_offering_id,
        pu.property_unit_id,
        pu.physical_building_id,
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
        AND (SELECT target_id FROM args) = CASE (SELECT target_type FROM args)
            WHEN 'offering' THEN po.property_offering_id
            WHEN 'unit' THEN pu.property_unit_id
            WHEN 'building' THEN pu.physical_building_id
            WHEN 'housing_company' THEN COALESCE(pu.housing_company_id, pb.housing_company_id)
        END
),
raw_candidates AS (
    SELECT c.property_dimension_claim_id, c.dimension_key, c.value, c.value_kind, c.unit, c.confidence, c.source_reliability, c.claim_scope, c.source_table, c.source_field, c.source_observed_at, c.created_at
    FROM public.dimension_claims c
    WHERE c.claim_scope = 'manual'
        AND c.target_type = (SELECT target_type FROM args)
        AND c.target_id = (SELECT target_id FROM args)
    UNION ALL
    SELECT c.property_dimension_claim_id, c.dimension_key, c.value, c.value_kind, c.unit, c.confidence, c.source_reliability, c.claim_scope, c.source_table, c.source_field, c.source_observed_at, c.created_at
    FROM public.dimension_claims c
    JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
    WHERE c.claim_scope = 'source'
        AND c.target_type = (SELECT target_type FROM args)
        AND c.target_id = (SELECT target_id FROM args)
        AND catalog.target_type = (SELECT target_type FROM args)
    UNION ALL
    SELECT c.property_dimension_claim_id, c.dimension_key, c.value, c.value_kind, c.unit, c.confidence, c.source_reliability, c.claim_scope, c.source_table, c.source_field, c.source_observed_at, c.created_at
    FROM linked_listings linked
    JOIN public.dimension_claims c
        ON c.claim_scope = 'source'
        AND c.target_type = 'listing'
        AND c.target_id = linked.sale_listing_id
    JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
    WHERE catalog.target_type = (SELECT target_type FROM args)
    UNION ALL
    SELECT c.property_dimension_claim_id, c.dimension_key, c.value, c.value_kind, c.unit, c.confidence, c.source_reliability, c.claim_scope, c.source_table, c.source_field, c.source_observed_at, c.created_at
    FROM public.property_documents d
    JOIN public.dimension_claims c
        ON c.claim_scope = 'source'
        AND c.source_table = 'property_documents'
        AND c.source_id = d.property_document_id
    JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
    WHERE catalog.target_type = (SELECT target_type FROM args)
        AND (SELECT target_id FROM args) = CASE (SELECT target_type FROM args)
            WHEN 'offering' THEN d.property_offering_id
            WHEN 'unit' THEN d.property_unit_id
            WHEN 'building' THEN d.physical_building_id
            WHEN 'housing_company' THEN d.housing_company_id
        END
),
candidates AS (
    SELECT DISTINCT ON (property_dimension_claim_id)
        property_dimension_claim_id,
        (SELECT target_type FROM args) AS target_type,
        (SELECT target_id FROM args) AS target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        confidence,
        source_reliability,
        claim_scope,
        source_table,
        source_field,
        source_observed_at,
        created_at
    FROM raw_candidates
    ORDER BY property_dimension_claim_id
),
scored AS (
    SELECT
        c.property_dimension_claim_id,
        c.target_type,
        c.target_id,
        c.dimension_key,
        c.value,
        c.value_kind,
        c.unit,
        c.confidence,
        c.source_reliability,
        c.claim_scope,
        c.source_table,
        c.source_field,
        c.source_observed_at,
        c.created_at,
        COALESCE(sp.priority, CASE WHEN c.claim_scope = 'manual' THEN 1000 ELSE 50 END) AS source_priority,
        COALESCE(sp.default_reliability, c.source_reliability) AS effective_reliability,
        p.strategy,
        p.freshness_half_life_days,
        CASE
            WHEN c.claim_scope = 'manual' THEN 1::double precision
            WHEN p.strategy IN ('stable_identity','document_preferred') AND p.freshness_half_life_days IS NULL THEN 1::double precision
            WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
            ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
        END AS freshness_factor,
        CASE
            WHEN c.claim_scope = 'manual' THEN 1::double precision
            WHEN p.strategy = 'document_preferred' AND c.source_table = 'property_documents' THEN 1.45::double precision
            WHEN p.strategy = 'stable_identity' AND c.source_table = 'property_documents' THEN 1.2::double precision
            WHEN p.strategy = 'latest_reliable' AND c.source_table = 'property_source_offerings' THEN 1.05::double precision
            ELSE 1::double precision
        END AS authority_factor,
        CASE
            WHEN c.claim_scope = 'manual' THEN 1000000::double precision
            ELSE COALESCE(sp.priority, 50)::double precision *
                COALESCE(sp.default_reliability, c.source_reliability) *
                c.confidence *
                CASE
                    WHEN p.strategy IN ('stable_identity','document_preferred') AND p.freshness_half_life_days IS NULL THEN 1::double precision
                    WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
                    ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
                END *
                CASE
                    WHEN p.strategy = 'document_preferred' AND c.source_table = 'property_documents' THEN 1.45::double precision
                    WHEN p.strategy = 'stable_identity' AND c.source_table = 'property_documents' THEN 1.2::double precision
                    WHEN p.strategy = 'latest_reliable' AND c.source_table = 'property_source_offerings' THEN 1.05::double precision
                    ELSE 1::double precision
                END
        END AS score
    FROM candidates c
    JOIN public.property_dimension_resolution_policies p ON p.dimension_key = c.dimension_key
    LEFT JOIN LATERAL (
        SELECT priority, default_reliability
        FROM public.property_dimension_source_priorities candidate_priority
        WHERE candidate_priority.dimension_key = c.dimension_key
            AND candidate_priority.source_table = c.source_table
            AND (
                candidate_priority.source_field IS NULL
                OR COALESCE(candidate_priority.source_field, '') = COALESCE(c.source_field, '')
            )
        ORDER BY CASE WHEN COALESCE(candidate_priority.source_field, '') = COALESCE(c.source_field, '') THEN 0 ELSE 1 END
        LIMIT 1
    ) sp ON true
),
ranked AS (
    SELECT
        property_dimension_claim_id,
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        confidence,
        source_reliability,
        claim_scope,
        source_table,
        source_field,
        source_observed_at,
        created_at,
        source_priority,
        effective_reliability,
        strategy,
        freshness_half_life_days,
        freshness_factor,
        authority_factor,
        score,
        row_number() OVER (
            PARTITION BY dimension_key
            ORDER BY score DESC, source_observed_at DESC NULLS LAST, created_at DESC, property_dimension_claim_id
        ) AS selected_rank
    FROM scored
),
stats AS (
    SELECT
        dimension_key,
        count(*) AS claim_count,
        count(DISTINCT value::text) AS distinct_value_count
    FROM scored
    GROUP BY dimension_key
),
selected AS (
    SELECT
        ranked.property_dimension_claim_id,
        ranked.target_type,
        ranked.target_id,
        ranked.dimension_key,
        ranked.value,
        ranked.value_kind,
        ranked.unit,
        ranked.confidence,
        ranked.source_reliability,
        ranked.claim_scope,
        ranked.source_table,
        ranked.source_field,
        ranked.source_observed_at,
        ranked.created_at,
        ranked.source_priority,
        ranked.effective_reliability,
        ranked.strategy,
        ranked.freshness_half_life_days,
        ranked.freshness_factor,
        ranked.authority_factor,
        ranked.score,
        ranked.selected_rank,
        stats.claim_count,
        stats.distinct_value_count
    FROM ranked
    JOIN stats ON stats.dimension_key = ranked.dimension_key
    WHERE ranked.selected_rank = 1
),
grouped AS (
    SELECT
        s.target_type,
        s.target_id,
        s.dimension_key,
        s.value,
        s.value_kind,
        s.unit,
        s.confidence,
        s.property_dimension_claim_id AS selected_claim_id,
        CASE
            WHEN s.claim_scope = 'manual' THEN 'manual override'
            ELSE concat(s.strategy, ' score=', round(s.score::numeric, 4)::text, ' priority=', s.source_priority::text, ' reliability=', round(s.effective_reliability::numeric, 3)::text, ' freshness=', round(s.freshness_factor::numeric, 3)::text, ' authority=', round(s.authority_factor::numeric, 3)::text)
        END AS selected_reason,
        CASE
            WHEN s.claim_scope = 'manual' THEN 'manual_override'
            WHEN s.distinct_value_count > 1 THEN 'conflicting'
            WHEN s.claim_count > 1 THEN 'compatible'
            ELSE 'none'
        END AS conflict_status,
        array_remove(array_agg(r.property_dimension_claim_id) FILTER (WHERE r.value::text = s.value::text), NULL) AS supporting_claim_ids,
        array_remove(array_agg(r.property_dimension_claim_id) FILTER (WHERE r.value::text <> s.value::text), NULL) AS rejected_claim_ids
    FROM selected s
    JOIN ranked r ON r.dimension_key = s.dimension_key
    GROUP BY
        s.target_type,
        s.target_id,
        s.dimension_key,
        s.value,
        s.value_kind,
        s.unit,
        s.confidence,
        s.property_dimension_claim_id,
        s.claim_scope,
        s.strategy,
        s.score,
        s.source_priority,
        s.effective_reliability,
        s.freshness_factor,
        s.authority_factor,
        s.distinct_value_count,
        s.claim_count
),
inserted AS (
    INSERT INTO public.dimension_values (
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        confidence,
        selected_claim_id,
        selected_reason,
        conflict_status,
        supporting_claim_ids,
        rejected_claim_ids,
        resolved_at
    )
    SELECT
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        confidence,
        selected_claim_id,
        selected_reason,
        conflict_status,
        COALESCE(supporting_claim_ids, ARRAY[]::uuid[]),
        COALESCE(rejected_claim_ids, ARRAY[]::uuid[]),
        now()
    FROM grouped
    RETURNING 1
),
counts AS (
    SELECT count(*)::integer AS count FROM inserted
),
updated_run AS (
    UPDATE public.property_dimension_projection_runs
    SET result = jsonb_build_object('resolved_values', counts.count),
        finished_at = now()
    FROM run, counts
    WHERE property_dimension_projection_runs.property_dimension_projection_run_id = run.property_dimension_projection_run_id
)
SELECT count FROM counts;

-- name: ProjectDimensionProfileForTarget :one
WITH args AS (
    SELECT v.target_type, v.target_id FROM (VALUES ($1::text, $2::uuid)) AS v(target_type, target_id)
),
sections AS (
    SELECT
        c.profile_section,
        jsonb_object_agg(c.profile_key, v.value ORDER BY c.profile_key) AS section_json
    FROM public.dimension_values v
    JOIN public.property_dimension_catalog c ON c.dimension_key = v.dimension_key
    WHERE v.target_type = (SELECT target_type FROM args)
        AND v.target_id = (SELECT target_id FROM args)
    GROUP BY c.profile_section
),
dimensions AS (
    SELECT COALESCE(jsonb_object_agg(profile_section, section_json ORDER BY profile_section), '{}'::jsonb) AS payload
    FROM sections
),
conflicts AS (
    SELECT COALESCE(jsonb_object_agg(dimension_key, conflict_status ORDER BY dimension_key), '{}'::jsonb) AS payload
    FROM public.dimension_values
    WHERE target_type = (SELECT target_type FROM args)
        AND target_id = (SELECT target_id FROM args)
        AND conflict_status <> 'none'
),
upserted AS (
    INSERT INTO public.dimension_profiles (
        target_type,
        target_id,
        dimensions,
        metadata,
        conflicts,
        resolved_at
    )
    SELECT
        (SELECT target_type FROM args),
        (SELECT target_id FROM args),
        dimensions.payload,
        jsonb_build_object('projection_version', 'dimension-profile-v1'),
        conflicts.payload,
        now()
    FROM dimensions, conflicts
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dimensions = EXCLUDED.dimensions,
        metadata = EXCLUDED.metadata,
        conflicts = EXCLUDED.conflicts,
        resolved_at = EXCLUDED.resolved_at
    RETURNING 1
)
SELECT count(*)::integer AS count FROM upserted;

-- name: ResolveDimensionTarget :one
SELECT jsonb_build_object('deprecated', false)::jsonb AS payload;

-- name: ListDimensionTargetsForListing :many
WITH linked AS (
    SELECT
        po.property_offering_id,
        pu.property_unit_id,
        pu.physical_building_id,
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = @sale_listing_id::uuid
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
),
target_candidates AS (
    SELECT
        catalog.target_type,
        CASE catalog.target_type
            WHEN 'offering' THEN linked.property_offering_id
            WHEN 'unit' THEN linked.property_unit_id
            WHEN 'building' THEN linked.physical_building_id
            WHEN 'housing_company' THEN linked.housing_company_id
        END AS target_id
    FROM linked
    JOIN public.dimension_claims c
        ON c.claim_scope = 'source'
        AND c.target_type = 'listing'
        AND c.target_id = @sale_listing_id::uuid
    JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
    UNION
    SELECT c.target_type, c.target_id
    FROM public.dimension_claims c
    JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
    WHERE c.claim_scope IN ('source','manual')
        AND c.source_table = 'property_source_offerings'
        AND c.source_id = @sale_listing_id::uuid
        AND c.target_type = catalog.target_type
)
SELECT DISTINCT target_type, target_id
FROM target_candidates
WHERE target_id IS NOT NULL;

-- name: ClearListingDimensionTargetsDirty :one
WITH linked AS (
    SELECT
        po.property_offering_id,
        pu.property_unit_id,
        pu.physical_building_id,
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = @sale_listing_id::uuid
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
),
targets AS (
    SELECT 'listing'::text AS target_type, @sale_listing_id::uuid AS target_id
    UNION ALL SELECT 'offering', property_offering_id FROM linked WHERE property_offering_id IS NOT NULL
    UNION ALL SELECT 'unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
    UNION ALL SELECT 'building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
    UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
),
cleared AS (
    UPDATE public.property_dimension_dirty_targets dirty
    SET resolved_at = now()
    FROM targets
    WHERE dirty.target_type = targets.target_type
        AND dirty.target_id = targets.target_id
        AND dirty.resolved_at IS NULL
        AND (
            sqlc.narg('expected_dirty_at')::timestamptz IS NULL
            OR dirty.dirty_at <= sqlc.narg('expected_dirty_at')::timestamptz
        )
    RETURNING 1
)
SELECT count(*)::integer AS count FROM cleared;

-- name: ClearPropertyDimensionTargetDirty :one
WITH args AS (
    SELECT v.target_type, v.target_id FROM (VALUES ($1::text, $2::uuid)) AS v(target_type, target_id)
),
cleared AS (
    UPDATE public.property_dimension_dirty_targets
    SET resolved_at = now()
    WHERE target_type = (SELECT target_type FROM args)
        AND target_id = (SELECT target_id FROM args)
        AND resolved_at IS NULL
        AND (
            $3::timestamptz IS NULL
            OR dirty_at <= $3::timestamptz
        )
    RETURNING 1
)
SELECT count(*)::integer AS count FROM cleared;

-- name: ClearPropertyDimensionTargetDirtyAny :one
WITH args AS (
    SELECT v.target_type, v.target_id FROM (VALUES ($1::text, $2::uuid)) AS v(target_type, target_id)
),
cleared AS (
    UPDATE public.property_dimension_dirty_targets
    SET resolved_at = now()
    WHERE target_type = (SELECT target_type FROM args)
        AND target_id = (SELECT target_id FROM args)
        AND resolved_at IS NULL
    RETURNING 1
)
SELECT count(*)::integer AS count FROM cleared;

-- name: MarkListingDimensionTargetsDirty :one
WITH linked AS (
    SELECT
        po.property_offering_id,
        po.property_unit_id,
        po.property_house_id,
        pu.physical_building_id,
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.source_id = @sale_listing_id::uuid
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
), targets AS (
    SELECT 'listing'::text AS target_type, @sale_listing_id::uuid AS target_id
    UNION ALL SELECT 'offering', property_offering_id FROM linked WHERE property_offering_id IS NOT NULL
    UNION ALL SELECT 'unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
    UNION ALL SELECT 'house', property_house_id FROM linked WHERE property_house_id IS NOT NULL
    UNION ALL SELECT 'building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
    UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
), marked AS (
    INSERT INTO public.property_dimension_dirty_targets (
        target_type,
        target_id,
        dirty_reasons,
        dirty_at,
        resolved_at
    )
    SELECT
        target_type,
        target_id,
        ARRAY[COALESCE(NULLIF(sqlc.arg(reason)::text, ''), 'changed')],
        now(),
        NULL::timestamptz
    FROM targets
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = (
            SELECT array_agg(DISTINCT reason ORDER BY reason)
            FROM unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons) AS reason
        ),
        dirty_at = now(),
        resolved_at = NULL
    RETURNING 1
)
SELECT count(*)::integer AS count FROM marked;

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
        AND source_link.source_id = @sale_listing_id::uuid
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
),
listing AS (
    SELECT
        sl.sale_listing_address_norm,
        sl.sale_listing_postal_norm,
        sl.sale_listing_city_norm,
        sl.sale_listing_build_year,
        sl.sale_listing_total_floors,
        sl.sale_listing_apartment_count,
        sl.sale_listing_elevator,
        sl.sale_listing_latitude,
        sl.sale_listing_longitude,
        linked.housing_company_id,
        linked.property_unit_id,
        linked.housing_company_identity_key
    FROM public.property_source_offerings sl
    JOIN linked ON linked.sale_listing_id = sl.sale_listing_id
    WHERE sl.sale_listing_id = @sale_listing_id::uuid
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
        housing_company_identity_key || ':building:' || COALESCE(NULLIF(trim(BOTH '_' FROM regexp_replace(lower(trim(COALESCE(sale_listing_address_norm, ''))), '[^[:alnum:]åäö]+', '_', 'g')), ''), 'main'),
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
WITH linked_offering AS (
    SELECT po.property_offering_id
    FROM public.target_sources source_link
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    WHERE source_link.source_id = $1::uuid
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    ORDER BY source_link.link_score DESC, source_link.updated_at DESC
    LIMIT 1
),
listing AS (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        sl.sale_listing_native_id,
        sl.sale_listing_address_norm,
        sl.sale_listing_postal_norm,
        sl.sale_listing_city_norm,
        sl.sale_listing_building_match_key,
        sl.sale_listing_area_value,
        sl.sale_listing_living_area_value,
        sl.sale_listing_plot_area_value,
        sl.sale_listing_rooms_count,
        sl.sale_listing_latitude,
        sl.sale_listing_longitude,
        sl.sale_listing_build_year,
        sl.sale_listing_first_seen_at,
        sl.sale_listing_last_seen_at,
        linked_offering.property_offering_id,
        COALESCE(
            'detached_address:' || NULLIF(trim(BOTH '_' FROM regexp_replace(lower(trim(COALESCE(concat_ws('|', sl.sale_listing_postal_norm, sl.sale_listing_city_norm, sl.sale_listing_building_match_key, sl.sale_listing_area_value::text), ''))), '[^[:alnum:]åäö]+', '_', 'g')), ''),
            'detached_source:' || sl.sale_listing_source_provider || ':' || sl.sale_listing_source_kind || ':' || sl.sale_listing_native_id
        ) AS house_key
    FROM public.property_source_offerings sl
    JOIN linked_offering ON true
    WHERE sl.sale_listing_id = $1::uuid
        AND sl.sale_listing_property_type_code = 'detached_house'
),
synced AS (
    INSERT INTO public.property_houses (
        property_house_identity_key,
        property_house_address_norm,
        property_house_postal_norm,
        property_house_city_norm,
        property_house_build_year,
        property_house_area_value,
        property_house_plot_area_value,
        property_house_rooms_count,
        property_house_latitude,
        property_house_longitude,
        property_house_match_reasons,
        primary_sale_listing_id,
        property_house_updated_at
    )
    SELECT
        listing.house_key,
        listing.sale_listing_address_norm,
        listing.sale_listing_postal_norm,
        listing.sale_listing_city_norm,
        listing.sale_listing_build_year,
        COALESCE(listing.sale_listing_living_area_value, listing.sale_listing_area_value),
        listing.sale_listing_plot_area_value,
        listing.sale_listing_rooms_count,
        COALESCE(listing.sale_listing_latitude, pb.physical_building_latitude, postgis.ST_Y(hc.housing_company_geom)),
        COALESCE(listing.sale_listing_longitude, pb.physical_building_longitude, postgis.ST_X(hc.housing_company_geom)),
        jsonb_build_object('source', listing.sale_listing_source_provider, 'method', $2::text, 'source_listing_id', listing.sale_listing_id),
        listing.sale_listing_id,
        now()
    FROM listing
    LEFT JOIN public.property_offerings current_po ON current_po.property_offering_id = listing.property_offering_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = current_po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
    WHERE listing.house_key IS NOT NULL
    ON CONFLICT (property_house_identity_key) DO UPDATE SET
        property_house_address_norm = COALESCE(public.property_houses.property_house_address_norm, EXCLUDED.property_house_address_norm),
        property_house_postal_norm = COALESCE(public.property_houses.property_house_postal_norm, EXCLUDED.property_house_postal_norm),
        property_house_city_norm = COALESCE(public.property_houses.property_house_city_norm, EXCLUDED.property_house_city_norm),
        property_house_build_year = COALESCE(public.property_houses.property_house_build_year, EXCLUDED.property_house_build_year),
        property_house_area_value = COALESCE(public.property_houses.property_house_area_value, EXCLUDED.property_house_area_value),
        property_house_plot_area_value = COALESCE(public.property_houses.property_house_plot_area_value, EXCLUDED.property_house_plot_area_value),
        property_house_rooms_count = COALESCE(public.property_houses.property_house_rooms_count, EXCLUDED.property_house_rooms_count),
        property_house_latitude = COALESCE(public.property_houses.property_house_latitude, EXCLUDED.property_house_latitude),
        property_house_longitude = COALESCE(public.property_houses.property_house_longitude, EXCLUDED.property_house_longitude),
        primary_sale_listing_id = COALESCE(public.property_houses.primary_sale_listing_id, EXCLUDED.primary_sale_listing_id),
        property_house_match_reasons = public.property_houses.property_house_match_reasons || EXCLUDED.property_house_match_reasons,
        property_house_updated_at = now()
    RETURNING property_house_id
),
updated_offering AS (
    UPDATE public.property_offerings
    SET property_house_id = synced.property_house_id,
        property_unit_id = NULL,
        property_offering_updated_at = now()
    FROM synced, linked_offering
    WHERE property_offerings.property_offering_id = linked_offering.property_offering_id
    RETURNING synced.property_house_id
),
target_source AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at
    )
    SELECT
        'house',
        updated_offering.property_house_id,
        'source_listing',
        listing.sale_listing_id,
        'confirmed',
        $2::text,
        100,
        jsonb_build_object('source', 'detached_house_listing', 'identity_key', listing.house_key),
        listing.sale_listing_first_seen_at,
        listing.sale_listing_last_seen_at
    FROM listing
    JOIN updated_offering ON true
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = EXCLUDED.link_status,
        link_method = EXCLUDED.link_method,
        link_score = EXCLUDED.link_score,
        link_reasons = target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_id
),
dirty AS (
    INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at)
    SELECT 'house', property_house_id, ARRAY['detached_house_regroup'], now()
    FROM updated_offering
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = ARRAY(SELECT DISTINCT unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons)),
        dirty_at = GREATEST(property_dimension_dirty_targets.dirty_at, EXCLUDED.dirty_at)
    RETURNING target_id
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
SELECT COALESCE((SELECT property_house_id FROM synced), '00000000-0000-0000-0000-000000000000'::uuid)::uuid;

-- name: BackfillDetachedPropertyHouses :one
WITH candidates AS (
    SELECT sl.sale_listing_id, po.property_offering_id
    FROM public.property_source_offerings sl
    JOIN public.target_sources source_link ON source_link.source_id = sl.sale_listing_id
        AND source_link.target_type = 'listing'
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    JOIN public.property_offerings po ON po.property_offering_id = source_link.target_id
    WHERE sl.sale_listing_property_type_code = 'detached_house'
        AND po.property_house_id IS NULL
    ORDER BY sl.sale_listing_updated_at DESC NULLS LAST, sl.sale_listing_id
    LIMIT $1::int
),
listing AS (
    SELECT
        sl.sale_listing_id,
        sl.sale_listing_source_provider,
        sl.sale_listing_source_kind,
        sl.sale_listing_native_id,
        sl.sale_listing_address_norm,
        sl.sale_listing_postal_norm,
        sl.sale_listing_city_norm,
        sl.sale_listing_building_match_key,
        sl.sale_listing_area_value,
        sl.sale_listing_living_area_value,
        sl.sale_listing_plot_area_value,
        sl.sale_listing_rooms_count,
        sl.sale_listing_latitude,
        sl.sale_listing_longitude,
        sl.sale_listing_build_year,
        sl.sale_listing_first_seen_at,
        sl.sale_listing_last_seen_at,
        candidates.property_offering_id,
        COALESCE(
            'detached_address:' || NULLIF(trim(BOTH '_' FROM regexp_replace(lower(trim(COALESCE(concat_ws('|', sl.sale_listing_postal_norm, sl.sale_listing_city_norm, sl.sale_listing_building_match_key, sl.sale_listing_area_value::text), ''))), '[^[:alnum:]åäö]+', '_', 'g')), ''),
            'detached_source:' || sl.sale_listing_source_provider || ':' || sl.sale_listing_source_kind || ':' || sl.sale_listing_native_id
        ) AS house_key
    FROM candidates
    JOIN public.property_source_offerings sl ON sl.sale_listing_id = candidates.sale_listing_id
),
synced AS (
    INSERT INTO public.property_houses (
        property_house_identity_key,
        property_house_address_norm,
        property_house_postal_norm,
        property_house_city_norm,
        property_house_build_year,
        property_house_area_value,
        property_house_plot_area_value,
        property_house_rooms_count,
        property_house_latitude,
        property_house_longitude,
        property_house_match_reasons,
        primary_sale_listing_id,
        property_house_updated_at
    )
    SELECT
        listing.house_key,
        listing.sale_listing_address_norm,
        listing.sale_listing_postal_norm,
        listing.sale_listing_city_norm,
        listing.sale_listing_build_year,
        COALESCE(listing.sale_listing_living_area_value, listing.sale_listing_area_value),
        listing.sale_listing_plot_area_value,
        listing.sale_listing_rooms_count,
        COALESCE(listing.sale_listing_latitude, pb.physical_building_latitude, postgis.ST_Y(hc.housing_company_geom)),
        COALESCE(listing.sale_listing_longitude, pb.physical_building_longitude, postgis.ST_X(hc.housing_company_geom)),
        jsonb_build_object('source', listing.sale_listing_source_provider, 'method', 'regroup_v2_backfill', 'source_listing_id', listing.sale_listing_id),
        listing.sale_listing_id,
        now()
    FROM listing
    LEFT JOIN public.property_offerings current_po ON current_po.property_offering_id = listing.property_offering_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = current_po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
    WHERE listing.house_key IS NOT NULL
    ON CONFLICT (property_house_identity_key) DO UPDATE SET
        property_house_address_norm = COALESCE(public.property_houses.property_house_address_norm, EXCLUDED.property_house_address_norm),
        property_house_postal_norm = COALESCE(public.property_houses.property_house_postal_norm, EXCLUDED.property_house_postal_norm),
        property_house_city_norm = COALESCE(public.property_houses.property_house_city_norm, EXCLUDED.property_house_city_norm),
        property_house_build_year = COALESCE(public.property_houses.property_house_build_year, EXCLUDED.property_house_build_year),
        property_house_area_value = COALESCE(public.property_houses.property_house_area_value, EXCLUDED.property_house_area_value),
        property_house_plot_area_value = COALESCE(public.property_houses.property_house_plot_area_value, EXCLUDED.property_house_plot_area_value),
        property_house_rooms_count = COALESCE(public.property_houses.property_house_rooms_count, EXCLUDED.property_house_rooms_count),
        property_house_latitude = COALESCE(public.property_houses.property_house_latitude, EXCLUDED.property_house_latitude),
        property_house_longitude = COALESCE(public.property_houses.property_house_longitude, EXCLUDED.property_house_longitude),
        primary_sale_listing_id = COALESCE(public.property_houses.primary_sale_listing_id, EXCLUDED.primary_sale_listing_id),
        property_house_match_reasons = public.property_houses.property_house_match_reasons || EXCLUDED.property_house_match_reasons,
        property_house_updated_at = now()
    RETURNING primary_sale_listing_id AS sale_listing_id, property_house_id
),
updated_offerings AS (
    UPDATE public.property_offerings
    SET property_house_id = synced.property_house_id,
        property_unit_id = NULL,
        property_offering_updated_at = now()
    FROM synced
    JOIN listing ON listing.sale_listing_id = synced.sale_listing_id
    WHERE property_offerings.property_offering_id = listing.property_offering_id
    RETURNING listing.sale_listing_id, synced.property_house_id
),
target_sources AS (
    INSERT INTO public.target_sources (
        target_type,
        target_id,
        source_type,
        source_id,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at
    )
    SELECT
        'house',
        updated_offerings.property_house_id,
        'source_listing',
        listing.sale_listing_id,
        'confirmed',
        'backfill_auto',
        100,
        jsonb_build_object('source', 'detached_house_listing', 'identity_key', listing.house_key),
        listing.sale_listing_first_seen_at,
        listing.sale_listing_last_seen_at
    FROM updated_offerings
    JOIN listing ON listing.sale_listing_id = updated_offerings.sale_listing_id
    ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
        link_status = EXCLUDED.link_status,
        link_method = EXCLUDED.link_method,
        link_score = EXCLUDED.link_score,
        link_reasons = target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, target_sources.last_seen_at)),
        updated_at = now()
    RETURNING target_id
),
dirty AS (
    INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at)
    SELECT 'house', property_house_id, ARRAY['detached_house_regroup'], now()
    FROM updated_offerings
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = ARRAY(SELECT DISTINCT unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons)),
        dirty_at = GREATEST(property_dimension_dirty_targets.dirty_at, EXCLUDED.dirty_at)
    RETURNING target_id
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
WHERE sale_listing_id = @sale_listing_id
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
WHERE sale_listing_id = @sale_listing_id
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
    AND observation.source_id = @sale_listing_id
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
WHERE target_type = CASE sqlc.arg(entity_type)::text
        WHEN 'sale_listing' THEN 'listing'
        WHEN 'property_unit' THEN 'unit'
        WHEN 'physical_building' THEN 'building'
        ELSE sqlc.arg(entity_type)::text
    END
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
WITH old_document AS (
    SELECT property_offering_id, property_unit_id, physical_building_id, housing_company_id
    FROM public.property_documents
    WHERE property_document_id = $1::uuid
    FOR UPDATE
),
target_offering AS (
    SELECT po.property_offering_id, po.property_unit_id, pu.physical_building_id, pu.housing_company_id
    FROM public.property_offerings po
    JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    WHERE po.property_offering_id = $2::uuid
),
updated_document AS (
    UPDATE public.property_documents
    SET property_offering_id = target_offering.property_offering_id,
        property_unit_id = target_offering.property_unit_id,
        physical_building_id = target_offering.physical_building_id,
        housing_company_id = target_offering.housing_company_id,
        property_document_updated_at = now()
    FROM target_offering
    WHERE property_documents.property_document_id = $1::uuid
    RETURNING property_documents.property_document_id
),
dirty_targets AS (
    SELECT 'offering'::text AS target_type, old_document.property_offering_id AS target_id, $3::text || '_old' AS reason
    FROM old_document
    WHERE old_document.property_offering_id IS NOT NULL
    UNION
    SELECT 'unit', po.property_unit_id, $3::text || '_old'
    FROM old_document
    JOIN public.property_offerings po ON po.property_offering_id = old_document.property_offering_id
    WHERE po.property_unit_id IS NOT NULL
    UNION
    SELECT 'house', po.property_house_id, $3::text || '_old'
    FROM old_document
    JOIN public.property_offerings po ON po.property_offering_id = old_document.property_offering_id
    WHERE po.property_house_id IS NOT NULL
    UNION
    SELECT 'listing', source_link.source_id, $3::text || '_old'
    FROM old_document
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = old_document.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    WHERE old_document.property_offering_id IS NOT NULL
    UNION
    SELECT 'offering', target_offering.property_offering_id, $3::text || '_new'
    FROM target_offering
    UNION
    SELECT 'unit', target_offering.property_unit_id, $3::text || '_new'
    FROM target_offering
    WHERE target_offering.property_unit_id IS NOT NULL
    UNION
    SELECT 'listing', source_link.source_id, $3::text || '_new'
    FROM target_offering
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = target_offering.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    UNION
    SELECT 'building', old_document.physical_building_id, $3::text || '_old'
    FROM old_document
    WHERE old_document.physical_building_id IS NOT NULL
    UNION
    SELECT 'housing_company', old_document.housing_company_id, $3::text || '_old'
    FROM old_document
    WHERE old_document.housing_company_id IS NOT NULL
    UNION
    SELECT 'building', target_offering.physical_building_id, $3::text || '_new'
    FROM target_offering
    WHERE target_offering.physical_building_id IS NOT NULL
    UNION
    SELECT 'housing_company', target_offering.housing_company_id, $3::text || '_new'
    FROM target_offering
    WHERE target_offering.housing_company_id IS NOT NULL
),
marked AS (
    INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at, resolved_at)
    SELECT target_type, target_id, ARRAY[reason], now(), NULL::timestamptz
    FROM dirty_targets
    WHERE target_id IS NOT NULL
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = (
            SELECT array_agg(DISTINCT merged_reason.reason_value ORDER BY merged_reason.reason_value)
            FROM unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons) AS merged_reason(reason_value)
        ),
        dirty_at = now(),
        resolved_at = NULL
    RETURNING 1
)
SELECT
    property_documents.property_document_id,
    property_documents.property_offering_id,
    property_documents.property_unit_id,
    property_documents.physical_building_id,
    property_documents.housing_company_id,
    property_documents.property_document_type,
    property_documents.property_document_filename,
    property_documents.property_document_mime_type,
    property_documents.property_document_size_bytes,
    property_documents.property_document_sha256,
    property_documents.property_document_extraction_status,
    property_documents.property_document_extraction_error,
    property_documents.property_document_uploaded_at,
    property_documents.property_document_extracted_at
FROM public.property_documents
JOIN updated_document ON updated_document.property_document_id = property_documents.property_document_id
WHERE property_documents.property_document_id = $1;

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
payload_input AS (
    SELECT
        sqlc.arg(entity_type)::text AS entity_type,
        sqlc.arg(section)::text AS section,
        sqlc.arg(key)::text AS key,
        sqlc.arg(value_kind)::text AS raw_value_kind,
        NULLIF(sqlc.arg(value_text), '') AS value_text,
        sqlc.narg(value_number)::double precision AS value_number,
        sqlc.narg(value_bool)::boolean AS value_bool,
        sqlc.narg(value_json)::jsonb AS value_json
),
payload AS (
    SELECT
        CASE entity_type
            WHEN 'sale_listing' THEN 'listing'
            WHEN 'property_unit' THEN 'unit'
            WHEN 'physical_building' THEN 'building'
            ELSE entity_type
        END AS target_type,
        CASE WHEN entity_type = 'manual' THEN 'manual' ELSE 'source' END AS claim_scope,
        CASE section || '.' || key
            WHEN 'unit.balcony' THEN 'features.balcony'
            WHEN 'balcony.has_balcony' THEN 'features.balcony'
            WHEN 'balcony.glazing' THEN 'features.balcony_glazing'
            WHEN 'unit.sauna' THEN 'features.sauna'
            WHEN 'sauna.has_sauna' THEN 'features.sauna'
            WHEN 'sauna.private_sauna' THEN 'features.private_sauna'
            WHEN 'parking.parking_text' THEN 'features.parking_type'
            WHEN 'storage.storage_quality' THEN 'features.storage_quality'
            WHEN 'views.view_quality' THEN 'features.view_quality'
            WHEN 'views.noise_risk' THEN 'features.noise_risk'
            WHEN 'condition.condition' THEN 'condition.unit_condition'
            WHEN 'layout.layout_quality' THEN 'layout.quality'
            WHEN 'layout.awkward_layout' THEN 'layout.awkward'
            WHEN 'heating.heating_method' THEN 'building.heating_method'
            WHEN 'charges.maintenance_charge_monthly' THEN 'charges.maintenance_monthly_eur'
            WHEN 'charges.capital_charge_monthly' THEN 'charges.capital_monthly_eur'
            WHEN 'charges.total_charge_monthly' THEN 'charges.total_monthly_eur'
            ELSE section || '.' || key
        END AS dimension_key,
        CASE raw_value_kind
            WHEN 'text' THEN 'string'
            WHEN 'bool' THEN 'boolean'
            WHEN 'json' THEN COALESCE(jsonb_typeof(value_json), 'object')
            ELSE raw_value_kind
        END AS value_kind,
        CASE raw_value_kind
            WHEN 'text' THEN to_jsonb(value_text)
            WHEN 'number' THEN to_jsonb(value_number)
            WHEN 'bool' THEN to_jsonb(value_bool)
            WHEN 'json' THEN value_json
            ELSE NULL::jsonb
        END AS value
    FROM payload_input
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
    AND source_id = @sale_listing_id
    AND superseded_at IS NULL
    AND evidence ->> 'source_field' LIKE 'llm_%';

-- name: DeleteLLMPropertyClaimsForEntity :exec
DELETE FROM public.dimension_claims claims
WHERE claims.target_type = CASE sqlc.arg(entity_type)::text
        WHEN 'sale_listing' THEN 'listing'
        WHEN 'property_unit' THEN 'unit'
        WHEN 'physical_building' THEN 'building'
        ELSE sqlc.arg(entity_type)::text
    END
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
    AND source_link.source_id = @sale_listing_id::uuid
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
payload_input AS (
    SELECT
        sqlc.arg(entity_type)::text AS entity_type,
        sqlc.arg(section)::text AS section,
        sqlc.arg(key)::text AS key,
        sqlc.arg(value_kind)::text AS raw_value_kind,
        NULLIF(sqlc.arg(value_text), '') AS value_text,
        sqlc.narg(value_number)::double precision AS value_number,
        sqlc.narg(value_bool)::boolean AS value_bool
),
payload AS (
    SELECT
        CASE entity_type
            WHEN 'sale_listing' THEN 'listing'
            WHEN 'property_unit' THEN 'unit'
            WHEN 'physical_building' THEN 'building'
            ELSE entity_type
        END AS target_type,
        CASE WHEN entity_type = 'manual' THEN 'manual' ELSE 'source' END AS claim_scope,
        CASE section || '.' || key
            WHEN 'unit.balcony' THEN 'features.balcony'
            WHEN 'balcony.has_balcony' THEN 'features.balcony'
            WHEN 'balcony.glazing' THEN 'features.balcony_glazing'
            WHEN 'unit.sauna' THEN 'features.sauna'
            WHEN 'sauna.has_sauna' THEN 'features.sauna'
            WHEN 'sauna.private_sauna' THEN 'features.private_sauna'
            WHEN 'parking.parking_text' THEN 'features.parking_type'
            WHEN 'storage.storage_quality' THEN 'features.storage_quality'
            WHEN 'views.view_quality' THEN 'features.view_quality'
            WHEN 'views.noise_risk' THEN 'features.noise_risk'
            WHEN 'condition.condition' THEN 'condition.unit_condition'
            WHEN 'layout.layout_quality' THEN 'layout.quality'
            WHEN 'layout.awkward_layout' THEN 'layout.awkward'
            WHEN 'heating.heating_method' THEN 'building.heating_method'
            WHEN 'charges.maintenance_charge_monthly' THEN 'charges.maintenance_monthly_eur'
            WHEN 'charges.capital_charge_monthly' THEN 'charges.capital_monthly_eur'
            WHEN 'charges.total_charge_monthly' THEN 'charges.total_monthly_eur'
            ELSE section || '.' || key
        END AS dimension_key,
        CASE raw_value_kind
            WHEN 'text' THEN 'string'
            WHEN 'bool' THEN 'boolean'
            ELSE raw_value_kind
        END AS value_kind,
        CASE raw_value_kind
            WHEN 'text' THEN to_jsonb(value_text)
            WHEN 'number' THEN to_jsonb(value_number)
            WHEN 'bool' THEN to_jsonb(value_bool)
            ELSE NULL::jsonb
        END AS value
    FROM payload_input
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
    FROM origin.shortcut_ads sa
    LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
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
            COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerDay}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area
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
    FROM origin.shortcut_buildings sb
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
    FROM origin.frontdoor_ads fa
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
    LEFT JOIN origin.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
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
    FROM origin.frontdoor_buildings fb
), filtered AS (
    SELECT
        u.source,
        u.kind,
        u.native_id,
        u.canonical_id,
        u.listing_id,
        u.offering_id,
        u.latitude,
        u.longitude,
        u.link_status,
        u.link_method,
        u.link_score,
        u.external_url_available,
        u.headline,
        u.address,
        u.city,
        u.postal,
        u.price,
        u.area,
        u.room_layout,
        u.url,
        u.last_seen_at,
        u.searchable,
        u.listing_type,
        u.published_at
    FROM unified u
    WHERE (@source_filter = 'all' OR u.source = @source_filter)
      AND (@kind_filter = 'all' OR u.kind = @kind_filter)
      AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
      AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
      AND (sqlc.narg('postal_filter')::text IS NULL OR trim(sqlc.narg('postal_filter')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal_filter')::text)) || '%'))
      AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
      AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
      AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
      AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
      AND (sqlc.narg('listing_type_filter')::text IS NULL OR u.listing_type IS NULL OR u.listing_type = sqlc.narg('listing_type_filter')::text)
      AND (@grouping_filter = 'all' OR (@grouping_filter = 'grouped' AND u.offering_id <> '') OR (@grouping_filter = 'ungrouped' AND u.offering_id = ''))
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
    JOIN origin.prices_transactions pt ON pt.prices_transaction_id = price_link.prices_transaction_id
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
    CASE WHEN @sort_mode = 'price_asc' THEN u.price END ASC NULLS LAST,
    CASE WHEN @sort_mode = 'price_desc' THEN u.price END DESC NULLS LAST,
    CASE WHEN @sort_mode = 'area_asc' THEN u.area END ASC NULLS LAST,
    CASE WHEN @sort_mode = 'area_desc' THEN u.area END DESC NULLS LAST,
    CASE WHEN @sort_mode = 'seen_desc' THEN u.last_seen_at END DESC NULLS LAST,
    u.last_seen_at DESC,
    u.source,
    u.kind,
    u.native_id
LIMIT @limit_count::int
OFFSET @offset_count::int;

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
    FROM origin.shortcut_ads sa
    LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
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
            COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')) AS street_address,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerDay}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS price,
            COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS area
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
    FROM origin.shortcut_buildings sb
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
    FROM origin.frontdoor_ads fa
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
    FROM origin.frontdoor_buildings fb
)
SELECT COUNT(*)::bigint AS count
FROM unified u
WHERE (@source_filter = 'all' OR u.source = @source_filter)
  AND (@kind_filter = 'all' OR u.kind = @kind_filter)
  AND (sqlc.narg('query_text')::text IS NULL OR trim(sqlc.narg('query_text')::text) = '' OR lower(u.searchable) LIKE ('%' || lower(trim(sqlc.narg('query_text')::text)) || '%'))
  AND (sqlc.narg('city_filter')::text IS NULL OR trim(sqlc.narg('city_filter')::text) = '' OR lower(COALESCE(u.city, '')) LIKE ('%' || lower(trim(sqlc.narg('city_filter')::text)) || '%'))
  AND (sqlc.narg('postal_filter')::text IS NULL OR trim(sqlc.narg('postal_filter')::text) = '' OR lower(COALESCE(u.postal, '')) LIKE ('%' || lower(trim(sqlc.narg('postal_filter')::text)) || '%'))
  AND (sqlc.narg('min_price')::bigint IS NULL OR u.price >= sqlc.narg('min_price')::bigint)
  AND (sqlc.narg('max_price')::bigint IS NULL OR u.price <= sqlc.narg('max_price')::bigint)
  AND (sqlc.narg('min_area')::float8 IS NULL OR u.area >= sqlc.narg('min_area')::float8)
  AND (sqlc.narg('max_area')::float8 IS NULL OR u.area <= sqlc.narg('max_area')::float8)
  AND (sqlc.narg('listing_type_filter')::text IS NULL OR u.listing_type IS NULL OR u.listing_type = sqlc.narg('listing_type_filter')::text)
  AND (@grouping_filter = 'all' OR (@grouping_filter = 'grouped' AND u.offering_id <> '') OR (@grouping_filter = 'ungrouped' AND u.offering_id = ''))
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
JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = ssl.shortcut_ad_id
JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = fsl.frontdoor_ad_id
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
LIMIT @limit_count::int;

-- name: GetShortcutAdUnifiedDetail :one
SELECT
    sa.shortcut_ad_id,
    sa.shortcut_ad_url,
    sa.shortcut_ad_type,
    sa.shortcut_ad_last_seen_at,
    sa.shortcut_building_id,
    COALESCE(sl.sale_listing_street_address, COALESCE(CASE WHEN NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '') IS NOT NULL AND NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), '') IS NOT NULL THEN concat_ws(' ', NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,streetNumber}'), ''), NULLIF(trim(sa.shortcut_ad_data #>> '{address,buildingLetter}'), '')) ELSE NULL END, NULLIF(trim(sa.shortcut_ad_data #>> '{address,formattedAddress}'), ''), NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,street,name}', sa.shortcut_ad_data #>> '{address,street}')), '')), sb.shortcut_building_address) AS ad_address,
    COALESCE(sl.sale_listing_city, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,city,name}', sa.shortcut_ad_data #>> '{address,city}')), '')) AS ad_city,
    COALESCE(sl.sale_listing_postal, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{address,zipCode,value}', sa.shortcut_ad_data #>> '{address,zipCode,name}', sa.shortcut_ad_data #>> '{address,zipCode}')), '')) AS ad_postal,
    COALESCE(sl.sale_listing_latitude, sb.shortcut_building_latitude) AS ad_latitude,
    COALESCE(sl.sale_listing_longitude, sb.shortcut_building_longitude) AS ad_longitude,
    COALESCE(sl.sale_listing_room_layout, sa.shortcut_ad_data #>> '{adData,roomConfiguration}')::text AS ad_room_layout,
    COALESCE(sl.sale_listing_asking_price, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,price}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerMonth}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerWeek}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,rentPerDay}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS ad_price,
    COALESCE(sl.sale_listing_area_value, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,size}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeTotal}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeLiving}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,sizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)), 0::float8) AS ad_area,
    COALESCE(sl.sale_listing_description_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,description}', sa.shortcut_ad_data #>> '{description}', sa.shortcut_ad_data #>> '{text}')), '')) AS shortcut_ad_description_text,
    COALESCE(sl.sale_listing_availability_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,availabilityDescription}', sa.shortcut_ad_data #>> '{availabilityDescription}', sa.shortcut_ad_data #>> '{adData,availableFrom}')), '')) AS shortcut_ad_availability_text,
    COALESCE(sl.sale_listing_renovations_done_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', sa.shortcut_ad_data #>> '{property,renovationsDoneDescription}')), '')) AS shortcut_ad_renovations_done_text,
    COALESCE(sl.sale_listing_renovations_planned_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', sa.shortcut_ad_data #>> '{property,renovationsPlannedDescription}')), '')) AS shortcut_ad_renovations_planned_text,
    COALESCE(sl.sale_listing_additional_info_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,additionalInfo}', sa.shortcut_ad_data #>> '{moreInformationAvailableFrom}', sa.shortcut_ad_data #>> '{property,otherInfo}')), '')) AS shortcut_ad_additional_info_text,
    COALESCE(sl.sale_listing_charges_text, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{priceData,chargesText}', sa.shortcut_ad_data #>> '{priceData,additionalInfo}', sa.shortcut_ad_data #>> '{property,periodicChargesAdditionalInfo}', sa.shortcut_ad_data #>> '{property,managementChargesAdditionalInfo}')), '')) AS shortcut_ad_charges_text,
    COALESCE(sl.sale_listing_maintenance_charge_monthly, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,maintenanceCharge}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,monthlyFee}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_maintenance_charge_monthly,
    COALESCE(sl.sale_listing_total_charge_monthly, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,totalCharge}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,monthlyFee}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_total_charge_monthly,
    COALESCE(sl.sale_listing_water_charge, (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,waterFee}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS shortcut_ad_water_charge,
    COALESCE(sl.sale_listing_debt_free_price, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceDebtFree}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,priceSell}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_debt_free_price,
    COALESCE(sl.sale_listing_debt_share_amount, (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,debtShare}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS shortcut_ad_debt_share_amount,
    COALESCE(sl.sale_listing_price_per_m2, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,pricePerSqm}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_price_per_m2,
    COALESCE(sl.sale_listing_floor_level, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,floor}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{floor}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_floor_level,
    COALESCE(sl.sale_listing_total_floors, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,totalFloors}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,floors}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_total_floors,
    COALESCE(sl.sale_listing_build_year, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{buildingData,year}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,constructionYear}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_build_year,
    COALESCE(sl.sale_listing_condition, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,condition}', sa.shortcut_ad_data #>> '{property,condition}')), '')) AS shortcut_ad_condition,
    COALESCE(sl.sale_listing_energy_class, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}')), '')) AS shortcut_ad_energy_class,
    COALESCE(sl.sale_listing_plot_type_raw, NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sa.shortcut_ad_data #>> '{adData,buildingOverrideLotOwnership}', sb.shortcut_building_plot_type)), '')) AS shortcut_ad_plot_type,
    COALESCE(sl.sale_listing_elevator, CASE WHEN sa.shortcut_ad_data #>> '{adData,elevator}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,elevator}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,elevator}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END, CASE WHEN sa.shortcut_ad_data #>> '{adData,hasElevator}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasElevator}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasElevator}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END) AS shortcut_ad_elevator,
    COALESCE(CASE WHEN sa.shortcut_ad_data #>> '{adData,sauna}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,sauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,sauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END, CASE WHEN sa.shortcut_ad_data #>> '{adData,hasSauna}' IS NULL THEN NULL WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasSauna}')) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true WHEN lower(trim(sa.shortcut_ad_data #>> '{adData,hasSauna}')) IN ('0', 'false', 'no', 'off', 'ei') THEN false ELSE NULL END)::boolean AS shortcut_ad_sauna,
    COALESCE(sl.sale_listing_rooms_count, COALESCE((SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{adData,rooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value), (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL WHEN length(parsed_value.value) - length(replace(parsed_value.value, '.', '')) > 1 THEN NULL ELSE (parsed_value.value::numeric)::int4 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(sa.shortcut_ad_data #>> '{rooms}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value))) AS shortcut_ad_rooms_count,
    sa.shortcut_ad_data,
    sb.shortcut_building_external_id,
    sb.shortcut_building_url,
    sb.shortcut_building_address,
    sb.shortcut_building_housing_company,
    (SELECT COUNT(*)::bigint FROM origin.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id) AS building_listing_count,
    (SELECT COUNT(*)::bigint FROM origin.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id) AS building_rental_count
FROM origin.shortcut_ads sa
LEFT JOIN origin.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
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
    (SELECT COUNT(*)::bigint FROM origin.shortcut_ads sa WHERE sa.shortcut_building_id = sb.shortcut_building_id) AS ad_count,
    (SELECT COUNT(*)::bigint FROM origin.shortcut_building_listings sbl WHERE sbl.shortcut_building_id = sb.shortcut_building_id) AS listing_count,
    (SELECT COUNT(*)::bigint FROM origin.shortcut_building_rentals sbr WHERE sbr.shortcut_building_id = sb.shortcut_building_id) AS rental_count,
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
FROM origin.shortcut_buildings sb
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
    COALESCE(sl.sale_listing_maintenance_charge_monthly, (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(jsonb_path_query_first(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price', jsonb_build_object('charge', to_jsonb('HOUSING_COMPANY_MAINTENANCE_CHARGE'::text))) #>> '{}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS frontdoor_ad_maintenance_charge_monthly,
    COALESCE(sl.sale_listing_total_charge_monthly, (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(jsonb_path_query_first(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price', jsonb_build_object('charge', to_jsonb('HOUSING_COMPANY_TOTAL_CHARGE'::text))) #>> '{}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS frontdoor_ad_total_charge_monthly,
    COALESCE(sl.sale_listing_water_charge, (SELECT CASE WHEN parsed_value.value IS NULL THEN NULL ELSE parsed_value.value::float8 END FROM (SELECT NULLIF(regexp_replace(replace(COALESCE(jsonb_path_query_first(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price', jsonb_build_object('charge', to_jsonb('WATER'::text))) #>> '{}', ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') AS value) parsed_value)) AS frontdoor_ad_water_charge,
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
FROM origin.frontdoor_ads fa
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
FROM origin.frontdoor_building_announcements fba
JOIN origin.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
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
    (SELECT COUNT(*)::bigint FROM origin.frontdoor_building_announcements fba WHERE fba.frontdoor_building_id = fb.frontdoor_building_id) AS announcement_count,
    fb.frontdoor_building_data
FROM origin.frontdoor_buildings fb
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
LEFT JOIN origin.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN origin.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
WHERE sl.sale_listing_id = @sale_listing_id
    AND sl.sale_listing_source_kind IN ('ad', 'announcement')
LIMIT 1;

-- name: RelinkPropertyUnitBuilding :one
WITH old_unit AS (
    SELECT physical_building_id, housing_company_id
    FROM public.property_units
    WHERE property_unit_id = $1::uuid
    FOR UPDATE
),
target_building AS (
    SELECT physical_building_id, housing_company_id
    FROM public.physical_buildings
    WHERE physical_building_id = $2::uuid
),
updated_unit AS (
    UPDATE public.property_units
    SET physical_building_id = target_building.physical_building_id,
        housing_company_id = COALESCE(target_building.housing_company_id, property_units.housing_company_id),
        property_unit_updated_at = now()
    FROM target_building
    WHERE property_units.property_unit_id = $1::uuid
    RETURNING property_units.property_unit_id
),
dirty_targets AS (
    SELECT 'unit'::text AS target_type, updated_unit.property_unit_id AS target_id, $3::text AS reason
    FROM updated_unit
    UNION
    SELECT 'offering', po.property_offering_id, $3::text
    FROM updated_unit
    JOIN public.property_offerings po ON po.property_unit_id = updated_unit.property_unit_id
    UNION
    SELECT 'listing', source_link.source_id, $3::text
    FROM updated_unit
    JOIN public.property_offerings po ON po.property_unit_id = updated_unit.property_unit_id
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = po.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
    UNION
    SELECT 'building', old_unit.physical_building_id, $3::text || '_old'
    FROM old_unit
    WHERE old_unit.physical_building_id IS NOT NULL
    UNION
    SELECT 'housing_company', old_unit.housing_company_id, $3::text || '_old'
    FROM old_unit
    WHERE old_unit.housing_company_id IS NOT NULL
),
marked AS (
    INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at, resolved_at)
    SELECT target_type, target_id, ARRAY[reason], now(), NULL::timestamptz
    FROM dirty_targets
    WHERE target_id IS NOT NULL
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = (
            SELECT array_agg(DISTINCT merged_reason.reason_value ORDER BY merged_reason.reason_value)
            FROM unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons) AS merged_reason(reason_value)
        ),
        dirty_at = now(),
        resolved_at = NULL
    RETURNING 1
)
SELECT jsonb_build_object(
    'property_unit_id', $1::uuid,
    'old_physical_building_id', (SELECT physical_building_id FROM old_unit),
    'new_physical_building_id', $2::uuid,
    'old_housing_company_id', (SELECT housing_company_id FROM old_unit),
    'new_housing_company_id', (SELECT housing_company_id FROM target_building),
    'dirty_targets', (SELECT count(*) FROM marked)
)::jsonb;

-- name: RelinkPhysicalBuildingHousingCompany :one
WITH old_building AS (
    SELECT housing_company_id
    FROM public.physical_buildings
    WHERE physical_building_id = $1::uuid
    FOR UPDATE
),
updated_building AS (
    UPDATE public.physical_buildings
    SET housing_company_id = $2::uuid,
        physical_building_updated_at = now()
    WHERE physical_building_id = $1::uuid
    RETURNING physical_building_id
),
updated_units AS (
    UPDATE public.property_units
    SET housing_company_id = $2::uuid,
        property_unit_updated_at = now()
    WHERE physical_building_id = $1::uuid
    RETURNING property_unit_id
),
dirty_targets AS (
    SELECT 'building'::text AS target_type, updated_building.physical_building_id AS target_id, $3::text AS reason
    FROM updated_building
    UNION
    SELECT 'housing_company', old_building.housing_company_id, $3::text || '_old'
    FROM old_building
    WHERE old_building.housing_company_id IS NOT NULL
    UNION
    SELECT 'housing_company', $2::uuid, $3::text || '_new'
    UNION
    SELECT 'unit', updated_units.property_unit_id, $3::text
    FROM updated_units
    UNION
    SELECT 'offering', po.property_offering_id, $3::text
    FROM updated_units
    JOIN public.property_offerings po ON po.property_unit_id = updated_units.property_unit_id
    UNION
    SELECT 'listing', source_link.source_id, $3::text
    FROM updated_units
    JOIN public.property_offerings po ON po.property_unit_id = updated_units.property_unit_id
    JOIN public.target_sources source_link ON source_link.target_type = 'listing'
        AND source_link.target_id = po.property_offering_id
        AND source_link.source_type = 'source_listing'
        AND source_link.link_status <> 'rejected'
),
marked AS (
    INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at, resolved_at)
    SELECT target_type, target_id, ARRAY[reason], now(), NULL::timestamptz
    FROM dirty_targets
    WHERE target_id IS NOT NULL
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = (
            SELECT array_agg(DISTINCT merged_reason.reason_value ORDER BY merged_reason.reason_value)
            FROM unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons) AS merged_reason(reason_value)
        ),
        dirty_at = now(),
        resolved_at = NULL
    RETURNING 1
)
SELECT jsonb_build_object(
    'physical_building_id', $1::uuid,
    'old_housing_company_id', (SELECT housing_company_id FROM old_building),
    'new_housing_company_id', $2::uuid,
    'dirty_targets', (SELECT count(*) FROM marked)
)::jsonb;
