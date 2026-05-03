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
        public.fnc__frontdoor_published_at(NEW.frontdoor_ad_data),
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

CREATE OR REPLACE FUNCTION public.fnc__frontdoor_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS tg__frontdoor_ads_link_postal_code ON public.frontdoor_ads;
DROP FUNCTION IF EXISTS public.fnc__link_frontdoor_ads_postal_code();
DROP INDEX IF EXISTS public.idx_frontdoor_ad_postal_postal_code_id;

ALTER TABLE public.frontdoor_ads
    DROP COLUMN IF EXISTS frontdoor_ad_publishing_time,
    DROP COLUMN IF EXISTS postal_postal_code_id,
    DROP COLUMN IF EXISTS frontdoor_ad_sauna;
