CREATE OR REPLACE FUNCTION public.fnc__shortcut_ad_street_address(data jsonb)
RETURNS text
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    street_name text;
    street_number text;
    building_letter text;
    formatted text;
BEGIN
    street_name := NULLIF(trim(COALESCE(data #>> '{address,street,name}', data #>> '{address,street}')), '');
    street_number := NULLIF(trim(data #>> '{address,streetNumber}'), '');
    building_letter := NULLIF(trim(data #>> '{address,buildingLetter}'), '');
    formatted := NULLIF(trim(data #>> '{address,formattedAddress}'), '');
    IF street_name IS NOT NULL AND street_number IS NOT NULL THEN
        RETURN concat_ws(' ', street_name, street_number, building_letter);
    END IF;
    RETURN COALESCE(formatted, street_name);
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__normalize_address_token(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF(trim(regexp_replace(lower(regexp_replace(trim(value), '[^[:alnum:]åäöÅÄÖ]+', ' ', 'g')), '\s+', ' ', 'g')), '')
$$;
CREATE OR REPLACE FUNCTION public.fnc__parse_finnish_address(value text)
RETURNS jsonb
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    line text;
    matches text[];
BEGIN
    line := NULLIF(trim(split_part(COALESCE(value, ''), ',', 1)), '');
    IF line IS NULL THEN
        RETURN '{}'::jsonb;
    END IF;
    line := regexp_replace(line, '\s+', ' ', 'g');
    matches := regexp_match(line, '^(.+?)\s+([0-9]+[[:alpha:]åäöÅÄÖ]?(?:[-/][0-9]+[[:alpha:]åäöÅÄÖ]?)?)(?:\s+([[:alpha:]åäöÅÄÖ]))?(?:\s+([0-9]+[[:alpha:]åäöÅÄÖ]?))?$');
    IF matches IS NULL THEN
        RETURN jsonb_build_object('street_name', line);
    END IF;
    RETURN jsonb_strip_nulls(jsonb_build_object(
        'street_name', NULLIF(trim(matches[1]), ''),
        'street_number', NULLIF(trim(matches[2]), ''),
        'building_letter', NULLIF(trim(matches[3]), ''),
        'apartment', NULLIF(trim(matches[4]), '')
    ));
END;
$$;
ALTER TABLE public.sale_listings
ADD COLUMN sale_listing_street_name text,
ADD COLUMN sale_listing_street_number text,
ADD COLUMN sale_listing_building_letter text,
ADD COLUMN sale_listing_apartment text,
ADD COLUMN sale_listing_street_name_norm text,
ADD COLUMN sale_listing_street_number_norm text,
ADD COLUMN sale_listing_building_letter_norm text,
ADD COLUMN sale_listing_city_norm text,
ADD COLUMN sale_listing_postal_norm text,
ADD COLUMN sale_listing_address_norm text,
ADD COLUMN sale_listing_address_components jsonb,
ADD COLUMN sale_listing_building_match_key text,
ADD COLUMN sale_listing_street_match_key text,
ADD COLUMN sale_listing_unit_match_key text;
CREATE OR REPLACE FUNCTION public.fnc__sale_listings_set_address_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    parsed jsonb;
BEGIN
    parsed := public.fnc__parse_finnish_address(NEW.sale_listing_street_address);
    NEW.sale_listing_street_name := COALESCE(NULLIF(trim(NEW.sale_listing_street_name), ''), parsed #>> '{street_name}');
    NEW.sale_listing_street_number := COALESCE(NULLIF(trim(NEW.sale_listing_street_number), ''), parsed #>> '{street_number}');
    NEW.sale_listing_building_letter := COALESCE(NULLIF(trim(NEW.sale_listing_building_letter), ''), parsed #>> '{building_letter}');
    NEW.sale_listing_apartment := COALESCE(NULLIF(trim(NEW.sale_listing_apartment), ''), parsed #>> '{apartment}');
    NEW.sale_listing_street_name_norm := public.fnc__normalize_address_token(NEW.sale_listing_street_name);
    NEW.sale_listing_street_number_norm := public.fnc__normalize_address_token(NEW.sale_listing_street_number);
    NEW.sale_listing_building_letter_norm := public.fnc__normalize_address_token(NEW.sale_listing_building_letter);
    NEW.sale_listing_city_norm := public.fnc__normalize_address_token(NEW.sale_listing_city);
    NEW.sale_listing_postal_norm := public.fnc__normalize_postal(NEW.sale_listing_postal);
    NEW.sale_listing_address_norm := public.fnc__normalize_address_token(concat_ws(' ', NEW.sale_listing_street_name, NEW.sale_listing_street_number, NEW.sale_listing_building_letter, NEW.sale_listing_apartment));
    NEW.sale_listing_address_components := jsonb_strip_nulls(jsonb_build_object(
        'street_name', NEW.sale_listing_street_name,
        'street_number', NEW.sale_listing_street_number,
        'building_letter', NEW.sale_listing_building_letter,
        'apartment', NEW.sale_listing_apartment,
        'city', NEW.sale_listing_city,
        'postal', NEW.sale_listing_postal
    ));
    IF NEW.sale_listing_postal_norm IS NOT NULL AND NEW.sale_listing_city_norm IS NOT NULL AND NEW.sale_listing_street_name_norm IS NOT NULL THEN
        NEW.sale_listing_street_match_key := NEW.sale_listing_postal_norm || '|' || NEW.sale_listing_city_norm || '|' || NEW.sale_listing_street_name_norm;
    ELSE
        NEW.sale_listing_street_match_key := NULL;
    END IF;
    IF NEW.sale_listing_street_match_key IS NOT NULL AND NEW.sale_listing_street_number_norm IS NOT NULL THEN
        NEW.sale_listing_building_match_key := NEW.sale_listing_street_match_key || '|' || NEW.sale_listing_street_number_norm || '|' || COALESCE(NEW.sale_listing_building_letter_norm, '');
    ELSE
        NEW.sale_listing_building_match_key := NULL;
    END IF;
    IF NEW.sale_listing_building_match_key IS NOT NULL AND NEW.sale_listing_apartment IS NOT NULL THEN
        NEW.sale_listing_unit_match_key := NEW.sale_listing_building_match_key || '|' || public.fnc__normalize_address_token(NEW.sale_listing_apartment);
    ELSE
        NEW.sale_listing_unit_match_key := NULL;
    END IF;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__sale_listings_set_address_fields ON public.sale_listings;
CREATE TRIGGER trg__sale_listings_set_address_fields
BEFORE INSERT OR UPDATE OF sale_listing_street_address, sale_listing_street_name, sale_listing_street_number, sale_listing_building_letter, sale_listing_apartment, sale_listing_city, sale_listing_postal ON public.sale_listings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sale_listings_set_address_fields();
CREATE INDEX idx_sale_listings_street_match_key ON public.sale_listings (sale_listing_street_match_key);
CREATE INDEX idx_sale_listings_building_match_key ON public.sale_listings (sale_listing_building_match_key);
CREATE INDEX idx_sale_listings_unit_match_key ON public.sale_listings (sale_listing_unit_match_key);
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
    street := public.fnc__shortcut_ad_street_address(NEW.shortcut_ad_data);
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
    area := COALESCE(
        public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,size}'),
        public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,sizeTotal}'),
        public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,sizeLiving}'),
        public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,sizeMin}'),
        public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,buildingOverrideTotalSize}'),
        public.fnc__try_parse_float8(NEW.shortcut_ad_data #>> '{adData,buildingOverrideSizeMin}')
    );
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
        sale_listing_public_id,
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
        sale_listing_updated_at
    )
    SELECT
        'l_' || substr(md5('shortcut:ad:' || NEW.shortcut_ad_id::text), 1, 16),
        NEW.shortcut_ad_id,
        'shortcut',
        'ad',
        NEW.shortcut_ad_id::text,
        'shortcut:ad:' || NEW.shortcut_ad_id::text,
        NEW.shortcut_ad_url,
        COALESCE(NEW.shortcut_ad_street_address, sb.shortcut_building_address, NEW.shortcut_ad_id::text),
        COALESCE(NEW.shortcut_ad_street_address, sb.shortcut_building_address),
        NULLIF(trim(NEW.shortcut_ad_data #>> '{address,street,name}'), ''),
        NULLIF(trim(NEW.shortcut_ad_data #>> '{address,streetNumber}'), ''),
        NULLIF(trim(NEW.shortcut_ad_data #>> '{address,buildingLetter}'), ''),
        NEW.shortcut_ad_city,
        NEW.shortcut_ad_postal,
        NEW.shortcut_ad_price,
        NEW.shortcut_ad_area_value,
        NEW.shortcut_ad_data #>> '{adData,roomConfiguration}',
        NEW.shortcut_ad_last_seen_at,
        (NEW.shortcut_ad_data #>> '{adData,published}')::timestamptz,
        concat_ws(' ', NEW.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company),
        now()
    FROM (SELECT 1) seed
    LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = NEW.shortcut_building_id
    ON CONFLICT (sale_listing_canonical_id) DO UPDATE SET
        sale_listing_public_id = EXCLUDED.sale_listing_public_id,
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
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;
UPDATE public.shortcut_ads
SET shortcut_ad_data = shortcut_ad_data
WHERE shortcut_ad_data IS NOT NULL;
UPDATE public.sale_listings
SET sale_listing_street_address = sale_listing_street_address;
