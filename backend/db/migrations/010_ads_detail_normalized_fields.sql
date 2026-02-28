ALTER TABLE public.shortcut_ads
ADD COLUMN IF NOT EXISTS shortcut_ad_description_text text,
ADD COLUMN IF NOT EXISTS shortcut_ad_availability_text text,
ADD COLUMN IF NOT EXISTS shortcut_ad_renovations_done_text text,
ADD COLUMN IF NOT EXISTS shortcut_ad_renovations_planned_text text,
ADD COLUMN IF NOT EXISTS shortcut_ad_additional_info_text text,
ADD COLUMN IF NOT EXISTS shortcut_ad_charges_text text,
ADD COLUMN IF NOT EXISTS shortcut_ad_maintenance_charge_monthly float8,
ADD COLUMN IF NOT EXISTS shortcut_ad_total_charge_monthly float8,
ADD COLUMN IF NOT EXISTS shortcut_ad_water_charge float8,
ADD COLUMN IF NOT EXISTS shortcut_ad_debt_free_price int8,
ADD COLUMN IF NOT EXISTS shortcut_ad_debt_share_amount int8,
ADD COLUMN IF NOT EXISTS shortcut_ad_price_per_m2 float8,
ADD COLUMN IF NOT EXISTS shortcut_ad_floor_level int4,
ADD COLUMN IF NOT EXISTS shortcut_ad_total_floors int4,
ADD COLUMN IF NOT EXISTS shortcut_ad_build_year int4,
ADD COLUMN IF NOT EXISTS shortcut_ad_condition text,
ADD COLUMN IF NOT EXISTS shortcut_ad_energy_class text,
ADD COLUMN IF NOT EXISTS shortcut_ad_plot_type text,
ADD COLUMN IF NOT EXISTS shortcut_ad_elevator bool,
ADD COLUMN IF NOT EXISTS shortcut_ad_sauna bool,
ADD COLUMN IF NOT EXISTS shortcut_ad_rooms_count int4;

ALTER TABLE public.frontdoor_ads
ADD COLUMN IF NOT EXISTS frontdoor_ad_description_text text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_availability_text text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_renovations_done_text text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_renovations_planned_text text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_additional_info_text text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_charges_text text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_maintenance_charge_monthly float8,
ADD COLUMN IF NOT EXISTS frontdoor_ad_total_charge_monthly float8,
ADD COLUMN IF NOT EXISTS frontdoor_ad_water_charge float8,
ADD COLUMN IF NOT EXISTS frontdoor_ad_debt_free_price int8,
ADD COLUMN IF NOT EXISTS frontdoor_ad_debt_share_amount int8,
ADD COLUMN IF NOT EXISTS frontdoor_ad_price_per_m2 float8,
ADD COLUMN IF NOT EXISTS frontdoor_ad_floor_level int4,
ADD COLUMN IF NOT EXISTS frontdoor_ad_total_floors int4,
ADD COLUMN IF NOT EXISTS frontdoor_ad_build_year int4,
ADD COLUMN IF NOT EXISTS frontdoor_ad_condition text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_energy_class text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_plot_type text,
ADD COLUMN IF NOT EXISTS frontdoor_ad_elevator bool,
ADD COLUMN IF NOT EXISTS frontdoor_ad_sauna bool,
ADD COLUMN IF NOT EXISTS frontdoor_ad_rooms_count int4;

CREATE OR REPLACE FUNCTION public.fnc__try_parse_int4(value text)
RETURNS int4
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE
    WHEN NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '') IS NULL THEN NULL
    ELSE (NULLIF(regexp_replace(replace(COALESCE(value, ''), ',', '.'), '[^0-9\.-]', '', 'g'), '')::numeric)::int4
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__try_parse_bool(value text)
RETURNS bool
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE
    WHEN value IS NULL THEN NULL
    WHEN lower(trim(value)) IN ('1', 'true', 'yes', 'on', 'kylla', 'kyllä', 'on') THEN true
    WHEN lower(trim(value)) IN ('0', 'false', 'no', 'off', 'ei') THEN false
    ELSE NULL
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__jsonb_periodic_charge_price(payload jsonb, charge_key text)
RETURNS float8
LANGUAGE sql
IMMUTABLE
AS $$
SELECT public.fnc__try_parse_float8(
    jsonb_path_query_first(
        COALESCE(payload, '{}'::jsonb),
        '$.property.periodicCharges[*] ? (@.periodicCharge == $charge).price',
        jsonb_build_object('charge', to_jsonb(charge_key))
    ) #>> '{}'
);
$$;

CREATE OR REPLACE FUNCTION public.fnc__shortcut_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    street text;
    city text;
    postal text;
    price int8;
    area float8;
BEGIN
    street := NULLIF(trim(COALESCE(
        NEW.shortcut_ad_data #>> '{address,street,name}',
        NEW.shortcut_ad_data #>> '{address,street}',
        NEW.shortcut_ad_data #>> '{address,formattedAddress}'
    )), '');
    city := NULLIF(trim(COALESCE(
        NEW.shortcut_ad_data #>> '{address,city,name}',
        NEW.shortcut_ad_data #>> '{address,city}'
    )), '');
    postal := NULLIF(trim(COALESCE(
        NEW.shortcut_ad_data #>> '{address,zipCode,value}',
        NEW.shortcut_ad_data #>> '{address,zipCode,name}',
        NEW.shortcut_ad_data #>> '{address,zipCode}'
    )), '');
    price := COALESCE(
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceSell}'),
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,price}')
    );
    area := public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,size}');
    NEW.shortcut_ad_street_address := street;
    NEW.shortcut_ad_city := city;
    NEW.shortcut_ad_postal := postal;
    NEW.shortcut_ad_price := price;
    NEW.shortcut_ad_area_value := area;
    NEW.shortcut_ad_address_key := concat_ws(
        '|',
        public.fnc__normalize_match_text(street),
        public.fnc__normalize_postal(postal),
        public.fnc__normalize_match_text(city)
    );
    NEW.shortcut_ad_search_text := trim(concat_ws(
        ' ',
        NEW.shortcut_ad_id::text,
        NEW.shortcut_ad_url,
        street,
        city,
        postal,
        NEW.shortcut_ad_data #>> '{adData,roomConfiguration}'
    ));
    NEW.shortcut_ad_description_text := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,description}', NEW.shortcut_ad_data #>> '{description}', NEW.shortcut_ad_data #>> '{text}')), '');
    NEW.shortcut_ad_availability_text := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,availabilityDescription}', NEW.shortcut_ad_data #>> '{availabilityDescription}', NEW.shortcut_ad_data #>> '{adData,availableFrom}')), '');
    NEW.shortcut_ad_renovations_done_text := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,renovationsDoneDescription}', NEW.shortcut_ad_data #>> '{property,renovationsDoneDescription}')), '');
    NEW.shortcut_ad_renovations_planned_text := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,renovationsPlannedDescription}', NEW.shortcut_ad_data #>> '{property,renovationsPlannedDescription}')), '');
    NEW.shortcut_ad_additional_info_text := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,additionalInfo}', NEW.shortcut_ad_data #>> '{moreInformationAvailableFrom}', NEW.shortcut_ad_data #>> '{property,otherInfo}')), '');
    NEW.shortcut_ad_charges_text := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{priceData,chargesText}', NEW.shortcut_ad_data #>> '{priceData,additionalInfo}', NEW.shortcut_ad_data #>> '{property,periodicChargesAdditionalInfo}', NEW.shortcut_ad_data #>> '{property,managementChargesAdditionalInfo}')), '');
    NEW.shortcut_ad_maintenance_charge_monthly := COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,maintenanceCharge}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,monthlyFee}'));
    NEW.shortcut_ad_total_charge_monthly := COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,totalCharge}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,monthlyFee}'));
    NEW.shortcut_ad_water_charge := public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,waterFee}');
    NEW.shortcut_ad_debt_free_price := COALESCE(public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceDebtFree}'), public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,priceSell}'));
    NEW.shortcut_ad_debt_share_amount := public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,debtShare}');
    NEW.shortcut_ad_price_per_m2 := COALESCE(public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,pricePerSqm}'), public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{priceData,pricePerSquareMeter}'));
    NEW.shortcut_ad_floor_level := COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,floor}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{floor}'));
    NEW.shortcut_ad_total_floors := COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,totalFloors}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{buildingData,floors}'));
    NEW.shortcut_ad_build_year := COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{buildingData,year}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,constructionYear}'));
    NEW.shortcut_ad_condition := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,condition}', NEW.shortcut_ad_data #>> '{property,condition}')), '');
    NEW.shortcut_ad_energy_class := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,energyClass}', NEW.shortcut_ad_data #>> '{property,energyClass}')), '');
    NEW.shortcut_ad_plot_type := NULLIF(trim(COALESCE(NEW.shortcut_ad_data #>> '{adData,plotType}', NEW.shortcut_ad_data #>> '{property,plotType}')), '');
    NEW.shortcut_ad_elevator := COALESCE(public.fnc__try_parse_bool(NEW.shortcut_ad_data #>> '{adData,elevator}'), public.fnc__try_parse_bool(NEW.shortcut_ad_data #>> '{adData,hasElevator}'));
    NEW.shortcut_ad_sauna := COALESCE(public.fnc__try_parse_bool(NEW.shortcut_ad_data #>> '{adData,sauna}'), public.fnc__try_parse_bool(NEW.shortcut_ad_data #>> '{adData,hasSauna}'));
    NEW.shortcut_ad_rooms_count := COALESCE(public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{adData,rooms}'), public.fnc__try_parse_int4(NEW.shortcut_ad_data #>> '{rooms}'));
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__frontdoor_ads_set_normalized_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    street text;
    city text;
    postal text;
    price int8;
    area float8;
BEGIN
    street := NULLIF(trim(COALESCE(
        NEW.frontdoor_ad_data #>> '{property,streetAddressFreeForm}',
        NEW.frontdoor_ad_data #>> '{property,address}',
        NEW.frontdoor_ad_data #>> '{property,streetNameFreeForm}'
    )), '');
    city := NULLIF(trim(COALESCE(
        NEW.frontdoor_ad_data #>> '{property,municipalityNameFreeForm}',
        NEW.frontdoor_ad_data #>> '{property,municipality}',
        NEW.frontdoor_ad_data #>> '{property,city}',
        NEW.frontdoor_ad_data #>> '{property,postCode,postArea}'
    )), '');
    postal := NULLIF(trim(COALESCE(
        NEW.frontdoor_ad_data #>> '{property,postalCode}',
        NEW.frontdoor_ad_data #>> '{property,addressPostalCode}',
        NEW.frontdoor_ad_data #>> '{property,postCode,postCode}'
    )), '');
    price := COALESCE(
        public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debfFreePrice}'),
        public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{preparsed,price}')
    );
    area := COALESCE(
        public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{preparsed,area}'),
        public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{property,livingArea}')
    );
    NEW.frontdoor_ad_street_address := street;
    NEW.frontdoor_ad_city := city;
    NEW.frontdoor_ad_postal := postal;
    NEW.frontdoor_ad_price := price;
    NEW.frontdoor_ad_area_value := area;
    NEW.frontdoor_ad_address_key := concat_ws(
        '|',
        public.fnc__normalize_match_text(street),
        public.fnc__normalize_postal(postal),
        public.fnc__normalize_match_text(city)
    );
    NEW.frontdoor_ad_search_text := trim(concat_ws(
        ' ',
        NEW.frontdoor_ad_external_id,
        NEW.frontdoor_ad_url,
        street,
        city,
        postal,
        NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}'
    ));
    NEW.frontdoor_ad_description_text := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{text}', NEW.frontdoor_ad_data #>> '{property,description}')), '');
    NEW.frontdoor_ad_availability_text := NULLIF(trim(NEW.frontdoor_ad_data #>> '{availabilityDescription}'), '');
    NEW.frontdoor_ad_renovations_done_text := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,renovationsDoneDescription}', NEW.frontdoor_ad_data #>> '{property,housingCompany,renovationsDoneDescription}')), '');
    NEW.frontdoor_ad_renovations_planned_text := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,renovationsPlannedDescription}', NEW.frontdoor_ad_data #>> '{property,housingCompany,renovationsPlannedDescription}')), '');
    NEW.frontdoor_ad_additional_info_text := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{moreInformationAvailableFrom}', NEW.frontdoor_ad_data #>> '{property,housingCompany,otherInfo}', NEW.frontdoor_ad_data #>> '{additionalItemsIncludedInSale}')), '');
    NEW.frontdoor_ad_charges_text := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,periodicChargesAdditionalInfo}', NEW.frontdoor_ad_data #>> '{property,managementChargesAdditionalInfo}')), '');
    NEW.frontdoor_ad_maintenance_charge_monthly := public.fnc__jsonb_periodic_charge_price(NEW.frontdoor_ad_data, 'HOUSING_COMPANY_MAINTENANCE_CHARGE');
    NEW.frontdoor_ad_total_charge_monthly := public.fnc__jsonb_periodic_charge_price(NEW.frontdoor_ad_data, 'HOUSING_COMPANY_TOTAL_CHARGE');
    NEW.frontdoor_ad_water_charge := public.fnc__jsonb_periodic_charge_price(NEW.frontdoor_ad_data, 'WATER');
    NEW.frontdoor_ad_debt_free_price := public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debfFreePrice}');
    NEW.frontdoor_ad_debt_share_amount := public.fnc__try_parse_bigint(NEW.frontdoor_ad_data #>> '{debtShareAmount}');
    NEW.frontdoor_ad_price_per_m2 := COALESCE(public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{pricePerSquareMeter}'), public.fnc__try_parse_float8(NEW.frontdoor_ad_data #>> '{preparsed,pricePerSquareMeter}'));
    NEW.frontdoor_ad_floor_level := COALESCE(public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,housingCompanyApartmentInformationDTO,floorLevel}'), public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{property,floorLevel}'));
    NEW.frontdoor_ad_total_floors := COALESCE(public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{property,housingCompany,floorCount}'), public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,floorCount}'));
    NEW.frontdoor_ad_build_year := COALESCE(public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,constructionFinishedYear}'), public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{property,housingCompany,usageStartYear}'));
    NEW.frontdoor_ad_condition := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,inspection,overallCondition}', NEW.frontdoor_ad_data #>> '{property,condition}')), '');
    NEW.frontdoor_ad_energy_class := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', NEW.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}')), '');
    NEW.frontdoor_ad_plot_type := NULLIF(trim(COALESCE(NEW.frontdoor_ad_data #>> '{property,plot,plotType}', NEW.frontdoor_ad_data #>> '{property,plot,holdingType}')), '');
    NEW.frontdoor_ad_elevator := CASE
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.property.housingCompany.housingCompanyFeatures[*] ? (@ == "ELEVATOR")') THEN true
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_ELEVATOR")') THEN false
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_ELEVATOR")') THEN true
        ELSE NULL
    END;
    NEW.frontdoor_ad_sauna := CASE
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_NO_SAUNA")') THEN false
        WHEN jsonb_path_exists(COALESCE(NEW.frontdoor_ad_data, '{}'::jsonb), '$.residenceDetailsDTO.generalDwellingFeatures[*] ? (@ == "HAS_SAUNA")') THEN true
        ELSE NULL
    END;
    NEW.frontdoor_ad_rooms_count := public.fnc__try_parse_int4(NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,totalRoomCount}');
    RETURN NEW;
END;
$$;

UPDATE public.shortcut_ads
SET shortcut_ad_data = shortcut_ad_data
WHERE shortcut_ad_data IS NOT NULL;

UPDATE public.frontdoor_ads
SET frontdoor_ad_data = frontdoor_ad_data
WHERE frontdoor_ad_data IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_shortcut_ads_build_year ON public.shortcut_ads(shortcut_ad_build_year);
CREATE INDEX IF NOT EXISTS idx_shortcut_ads_floor_level ON public.shortcut_ads(shortcut_ad_floor_level);
CREATE INDEX IF NOT EXISTS idx_shortcut_ads_maintenance_charge ON public.shortcut_ads(shortcut_ad_maintenance_charge_monthly);

CREATE INDEX IF NOT EXISTS idx_frontdoor_ads_build_year ON public.frontdoor_ads(frontdoor_ad_build_year);
CREATE INDEX IF NOT EXISTS idx_frontdoor_ads_floor_level ON public.frontdoor_ads(frontdoor_ad_floor_level);
CREATE INDEX IF NOT EXISTS idx_frontdoor_ads_maintenance_charge ON public.frontdoor_ads(frontdoor_ad_maintenance_charge_monthly);
