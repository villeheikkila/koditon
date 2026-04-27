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
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,price}'),
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,rentPerMonth}'),
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,rentPerWeek}'),
        public.fnc__try_parse_bigint(NEW.shortcut_ad_data #>> '{priceData,rentPerDay}')
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

UPDATE public.shortcut_ads
SET shortcut_ad_data = shortcut_ad_data
WHERE shortcut_ad_data IS NOT NULL;
