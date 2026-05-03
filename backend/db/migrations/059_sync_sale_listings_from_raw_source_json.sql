CREATE OR REPLACE FUNCTION public.fnc__frontdoor_published_at(data jsonb)
RETURNS timestamp with time zone
LANGUAGE sql
STABLE
AS $$
    SELECT CASE
        WHEN public.fnc__try_parse_float8(data #>> '{publishingTime}') IS NULL THEN NULL
        ELSE to_timestamp(public.fnc__try_parse_float8(data #>> '{publishingTime}') / 1000.0)
    END
$$;

CREATE OR REPLACE FUNCTION public.fnc__sync_sale_listing_from_shortcut_ad()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.sale_listings WHERE shortcut_ad_id = OLD.shortcut_ad_id;
        RETURN OLD;
    END IF;
    IF NEW.shortcut_ad_type <> 'listing' THEN
        DELETE FROM public.sale_listings WHERE shortcut_ad_id = NEW.shortcut_ad_id;
        RETURN NEW;
    END IF;
    INSERT INTO public.sale_listings (
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
        sale_listing_updated_at
    )
    SELECT
        NEW.shortcut_ad_id,
        'shortcut',
        'ad',
        NEW.shortcut_ad_id::text,
        'shortcut:ad:' || NEW.shortcut_ad_id::text,
        NEW.shortcut_ad_url,
        COALESCE(raw.street_address, sb.shortcut_building_address, NEW.shortcut_ad_id::text),
        COALESCE(raw.street_address, sb.shortcut_building_address),
        NULLIF(trim(NEW.shortcut_ad_data #>> '{address,street,name}'), ''),
        NULLIF(trim(NEW.shortcut_ad_data #>> '{address,streetNumber}'), ''),
        NULLIF(trim(NEW.shortcut_ad_data #>> '{address,buildingLetter}'), ''),
        raw.city,
        raw.postal,
        raw.price,
        raw.area,
        NEW.shortcut_ad_data #>> '{adData,roomConfiguration}',
        NEW.shortcut_ad_last_seen_at,
        (NEW.shortcut_ad_data #>> '{adData,published}')::timestamptz,
        trim(concat_ws(' ', NEW.shortcut_ad_id::text, NEW.shortcut_ad_url, raw.street_address, raw.city, raw.postal, NEW.shortcut_ad_data #>> '{adData,roomConfiguration}', sb.shortcut_building_address, sb.shortcut_building_housing_company)),
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
        now()
    FROM (
        SELECT
            public.fnc__shortcut_ad_street_address(NEW.shortcut_ad_data) AS street_address,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{address,city,name}', NEW.shortcut_ad_data #>> '{address,city}')), '') AS city,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{address,zipCode,value}', NEW.shortcut_ad_data #>> '{address,zipCode,name}', NEW.shortcut_ad_data #>> '{address,zipCode}')), '') AS postal,
            COALESCE(public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceSell}'), public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,price}'), public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,rentPerMonth}'), public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,rentPerWeek}'), public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,rentPerDay}')) AS price,
            COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,size}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,sizeTotal}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,sizeLiving}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,sizeMin}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')) AS area,
            COALESCE(public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceDebtFree}'), public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceSell}')) AS debt_free_price,
            public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,debtShare}') AS debt_share_amount,
            COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,pricePerSqm}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}')) AS price_per_m2,
            COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,rooms}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{rooms}')) AS rooms_count,
            COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,floor}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{floor}')) AS floor_level,
            COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,totalFloors}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{buildingData,floors}')) AS total_floors,
            COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{buildingData,year}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,constructionYear}')) AS build_year,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,condition}', NEW.shortcut_ad_data #>> '{property,condition}')), '') AS condition,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,energyClass}', NEW.shortcut_ad_data #>> '{property,energyClass}')), '') AS energy_class,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,description}', NEW.shortcut_ad_data #>> '{description}', NEW.shortcut_ad_data #>> '{text}')), '') AS description_text,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,availabilityDescription}', NEW.shortcut_ad_data #>> '{availabilityDescription}', NEW.shortcut_ad_data #>> '{adData,availableFrom}')), '') AS availability_text,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', NEW.shortcut_ad_data #>> '{property,renovationsDoneDescription}')), '') AS renovations_done_text,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', NEW.shortcut_ad_data #>> '{property,renovationsPlannedDescription}')), '') AS renovations_planned_text,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,additionalInfo}', NEW.shortcut_ad_data #>> '{moreInformationAvailableFrom}', NEW.shortcut_ad_data #>> '{property,otherInfo}')), '') AS additional_info_text,
            NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{priceData,chargesText}', NEW.shortcut_ad_data #>> '{priceData,additionalInfo}', NEW.shortcut_ad_data #>> '{property,periodicChargesAdditionalInfo}', NEW.shortcut_ad_data #>> '{property,managementChargesAdditionalInfo}')), '') AS charges_text,
            COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,maintenanceCharge}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,monthlyFee}')) AS maintenance_charge_monthly,
            COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,totalCharge}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,monthlyFee}')) AS total_charge_monthly,
            public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,waterFee}') AS water_charge
    ) raw
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = NEW.shortcut_building_id
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
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__sync_sale_listing_from_frontdoor_ad()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.sale_listings WHERE frontdoor_ad_id = OLD.frontdoor_ad_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.sale_listings (
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
        sale_listing_updated_at
    )
    SELECT
        NEW.frontdoor_ad_id,
        'frontdoor',
        'ad',
        NEW.frontdoor_ad_external_id,
        'frontdoor:ad:' || NEW.frontdoor_ad_external_id,
        NEW.frontdoor_ad_url,
        COALESCE(raw.street_address, NEW.frontdoor_ad_external_id),
        raw.street_address,
        raw.city,
        raw.postal,
        raw.price,
        raw.area,
        NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}',
        NEW.frontdoor_ad_last_seen_at,
        COALESCE(NEW.frontdoor_ad_publishing_time, public.fnc__frontdoor_published_at(NEW.frontdoor_ad_data)),
        trim(concat_ws(' ', NEW.frontdoor_ad_external_id, NEW.frontdoor_ad_url, raw.street_address, raw.city, raw.postal, NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}')),
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
        now()
    FROM (
        SELECT
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,streetAddressFreeForm}', NEW.frontdoor_ad_data #>> '{property,address}', NEW.frontdoor_ad_data #>> '{property,streetNameFreeForm}')), '') AS street_address,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,municipalityNameFreeForm}', NEW.frontdoor_ad_data #>> '{property,municipality}', NEW.frontdoor_ad_data #>> '{property,city}', NEW.frontdoor_ad_data #>> '{property,postCode,postArea}')), '') AS city,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,postalCode}', NEW.frontdoor_ad_data #>> '{property,addressPostalCode}', NEW.frontdoor_ad_data #>> '{property,postCode,postCode}')), '') AS postal,
            COALESCE(public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debfFreePrice}'), public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{preparsed,price}')) AS price,
            COALESCE(public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{preparsed,area}'), public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{property,livingArea}')) AS area,
            public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debfFreePrice}') AS debt_free_price,
            public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debtShareAmount}') AS debt_share_amount,
            COALESCE(public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{pricePerSquareMeter}'), public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{preparsed,pricePerSquareMeter}')) AS price_per_m2,
            public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,totalRoomCount}') AS rooms_count,
            COALESCE(public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,housingCompanyApartmentInformationDTO,floorLevel}'), public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{property,floorLevel}')) AS floor_level,
            COALESCE(public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{property,housingCompany,floorCount}'), public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,floorCount}')) AS total_floors,
            COALESCE(public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,constructionFinishedYear}'), public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{property,housingCompany,usageStartYear}')) AS build_year,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,inspection,overallCondition}', NEW.frontdoor_ad_data #>> '{property,condition}')), '') AS condition,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', NEW.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}')), '') AS energy_class,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{text}', NEW.frontdoor_ad_data #>> '{property,description}')), '') AS description_text,
            NULLIF(trim(NEW.frontdoor_ad_data #>> '{availabilityDescription}'), '') AS availability_text,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,renovationsDoneDescription}', NEW.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}')), '') AS renovations_done_text,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,renovationsPlannedDescription}', NEW.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}')), '') AS renovations_planned_text,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{moreInformationAvailableFrom}', NEW.frontdoor_ad_data #>> '{property,housingCompany,otherInfo}', NEW.frontdoor_ad_data #>> '{additionalItemsIncludedInSale}')), '') AS additional_info_text,
            NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,periodicChargesAdditionalInfo}', NEW.frontdoor_ad_data #>> '{property,managementChargesAdditionalInfo}')), '') AS charges_text,
            public.fnc__jsonb_periodic_charge_price(NEW.frontdoor_ad_data, 'HOUSING_COMPANY_MAINTENANCE_CHARGE') AS maintenance_charge_monthly,
            public.fnc__jsonb_periodic_charge_price(NEW.frontdoor_ad_data, 'HOUSING_COMPANY_TOTAL_CHARGE') AS total_charge_monthly,
            public.fnc__jsonb_periodic_charge_price(NEW.frontdoor_ad_data, 'WATER') AS water_charge
    ) raw
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
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__sale_listings_set_transaction_match_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    property_raw text;
    plot_raw text;
    plot_owned boolean;
    elevator_value boolean;
    energy_label text;
    energy_match_label text;
    energy_normalized record;
BEGIN
    IF NEW.shortcut_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), ''),
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sa.shortcut_ad_data #>> '{adData,buildingOverrideLotOwnership}', sb.shortcut_building_plot_type)), ''),
            COALESCE(public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,elevator}'), public.fnc__try_parse_bool(sa.shortcut_ad_data #>> '{adData,hasElevator}'), public.fnc__try_parse_bool(sb.shortcut_building_has_elevator)),
            public.fnc__energy_efficiency_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}'),
            public.fnc__energy_efficiency_match_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}')
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.shortcut_ads sa
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        WHERE sa.shortcut_ad_id = NEW.shortcut_ad_id;
    ELSIF NEW.frontdoor_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), ''),
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,housingCompany,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_data #>> '{property,plot,plotType}')), ''),
            CASE
                WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.property.housingCompany.housingCompanyFeatures[*] ? (@ == "ELEVATOR")') THEN true
                WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_ELEVATOR")') THEN false
                WHEN jsonb_path_exists(COALESCE(fa.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_ELEVATOR")') THEN true
                ELSE NULL
            END,
            public.fnc__energy_efficiency_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}'),
            public.fnc__energy_efficiency_match_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}')
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.frontdoor_ads fa
        WHERE fa.frontdoor_ad_id = NEW.frontdoor_ad_id;
    ELSIF NEW.frontdoor_building_announcement_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
            NULL::text,
            fb.frontdoor_building_has_elevator,
            public.fnc__energy_efficiency_label(fb.frontdoor_building_energy_certificate_code),
            public.fnc__energy_efficiency_match_label(fb.frontdoor_building_energy_certificate_code)
        INTO property_raw, plot_raw, elevator_value, energy_label, energy_match_label
        FROM public.frontdoor_building_announcements fba
        JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE fba.frontdoor_building_announcement_id = NEW.frontdoor_building_announcement_id;
    END IF;
    SELECT * INTO energy_normalized FROM public.fnc__energy_efficiency_normalized(energy_match_label);
    plot_owned := public.fnc__plot_owned(plot_raw);
    NEW.sale_listing_property_type_raw := property_raw;
    NEW.sale_listing_property_type_code := public.fnc__sale_listing_property_type_code(property_raw);
    NEW.sale_listing_room_category_code := public.fnc__sale_listing_room_category_code(NEW.sale_listing_rooms_count, NEW.sale_listing_room_layout);
    NEW.sale_listing_floor_text := public.fnc__sale_listing_floor_text(NEW.sale_listing_floor_level, NEW.sale_listing_total_floors);
    NEW.sale_listing_elevator := elevator_value;
    NEW.sale_listing_plot_type_raw := plot_raw;
    NEW.sale_listing_plot_type_code := CASE WHEN plot_owned IS TRUE THEN 'own' WHEN plot_owned IS FALSE THEN 'rent' ELSE NULL END;
    NEW.sale_listing_plot_owned := plot_owned;
    NEW.sale_listing_energy_efficiency_label := energy_label;
    NEW.sale_listing_energy_efficiency_class_code := energy_normalized.energy_efficiency_class_code;
    NEW.sale_listing_energy_efficiency_standard_year := energy_normalized.energy_efficiency_standard_year;
    NEW.sale_listing_energy_efficiency_status := energy_normalized.energy_efficiency_status;
    NEW.sale_listing_energy_efficiency_match_code := energy_normalized.energy_efficiency_match_code;
    RETURN NEW;
END;
$$;
