CREATE OR REPLACE FUNCTION public.fnc__derived_price_per_m2(price bigint, area double precision, existing double precision)
RETURNS double precision
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT COALESCE(existing, CASE WHEN price IS NOT NULL AND area IS NOT NULL AND area > 0 THEN price::double precision / area ELSE NULL END)
$$;
DROP FUNCTION IF EXISTS public.fnc__link_sale_listing_prices_transaction(text, uuid);
DROP FUNCTION IF EXISTS public.fnc__unlink_sale_listing_prices_transaction(text);
ALTER TABLE public.sale_listings DROP CONSTRAINT IF EXISTS sale_listings_public_id_key;
ALTER TABLE public.sale_listings ALTER COLUMN sale_listing_public_id DROP NOT NULL;
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
        sale_listing_updated_at
    )
    SELECT
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
        public.fnc__derived_price_per_m2(NEW.shortcut_ad_price, NEW.shortcut_ad_area_value, NEW.shortcut_ad_price_per_m2),
        NEW.shortcut_ad_debt_free_price,
        NEW.shortcut_ad_debt_share_amount,
        NEW.shortcut_ad_rooms_count,
        NEW.shortcut_ad_floor_level,
        NEW.shortcut_ad_total_floors,
        COALESCE(NEW.shortcut_ad_build_year, sb.shortcut_building_construction_year),
        NEW.shortcut_ad_condition,
        NEW.shortcut_ad_energy_class,
        NEW.shortcut_ad_description_text,
        now()
    FROM (SELECT 1) seed
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
        sale_listing_updated_at
    )
    VALUES (
        NEW.frontdoor_ad_id,
        'frontdoor',
        'ad',
        NEW.frontdoor_ad_external_id,
        'frontdoor:ad:' || NEW.frontdoor_ad_external_id,
        NEW.frontdoor_ad_url,
        COALESCE(NEW.frontdoor_ad_street_address, NEW.frontdoor_ad_external_id),
        NEW.frontdoor_ad_street_address,
        NEW.frontdoor_ad_city,
        NEW.frontdoor_ad_postal,
        NEW.frontdoor_ad_price,
        NEW.frontdoor_ad_area_value,
        NEW.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}',
        NEW.frontdoor_ad_last_seen_at,
        NEW.frontdoor_ad_publishing_time,
        NEW.frontdoor_ad_search_text,
        public.fnc__derived_price_per_m2(NEW.frontdoor_ad_price, NEW.frontdoor_ad_area_value, NEW.frontdoor_ad_price_per_m2),
        NEW.frontdoor_ad_debt_free_price,
        NEW.frontdoor_ad_debt_share_amount,
        NEW.frontdoor_ad_rooms_count,
        NEW.frontdoor_ad_floor_level,
        NEW.frontdoor_ad_total_floors,
        NEW.frontdoor_ad_build_year,
        NEW.frontdoor_ad_condition,
        NEW.frontdoor_ad_energy_class,
        NEW.frontdoor_ad_description_text,
        now()
    )
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
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__refresh_sale_listings_from_shortcut_building()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE public.sale_listings sl
    SET
        sale_listing_headline = COALESCE(sa.shortcut_ad_street_address, NEW.shortcut_building_address, sa.shortcut_ad_id::text),
        sale_listing_street_address = COALESCE(sa.shortcut_ad_street_address, NEW.shortcut_building_address),
        sale_listing_build_year = COALESCE(sa.shortcut_ad_build_year, NEW.shortcut_building_construction_year),
        sale_listing_search_text = concat_ws(' ', sa.shortcut_ad_search_text, NEW.shortcut_building_address, NEW.shortcut_building_housing_company),
        sale_listing_updated_at = now()
    FROM public.shortcut_ads sa
    WHERE sl.shortcut_ad_id = sa.shortcut_ad_id
      AND sa.shortcut_building_id = NEW.shortcut_building_id
      AND sa.shortcut_ad_type = 'listing';
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_sale_listing_from_frontdoor_announcement()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.sale_listings WHERE frontdoor_building_announcement_id = OLD.frontdoor_building_announcement_id;
        RETURN OLD;
    END IF;
    IF NEW.frontdoor_building_announcement_rent_period IS NOT NULL OR NEW.frontdoor_building_announcement_rental_unique_no IS NOT NULL THEN
        DELETE FROM public.sale_listings WHERE frontdoor_building_announcement_id = NEW.frontdoor_building_announcement_id;
        RETURN NEW;
    END IF;
    INSERT INTO public.sale_listings (
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
        sale_listing_build_year,
        sale_listing_energy_class,
        sale_listing_updated_at
    )
    SELECT
        NEW.frontdoor_building_announcement_id,
        'frontdoor',
        'announcement',
        NEW.frontdoor_building_announcement_id::text,
        'frontdoor:announcement:' || NEW.frontdoor_building_announcement_id::text,
        fb.frontdoor_building_url,
        COALESCE(NEW.frontdoor_building_announcement_address_line1, NEW.frontdoor_building_announcement_friendly_id, NEW.frontdoor_building_announcement_external_id::text, NEW.frontdoor_building_announcement_id::text),
        concat_ws(' ', NEW.frontdoor_building_announcement_address_line1, NEW.frontdoor_building_announcement_address_line2),
        COALESCE(NEW.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area),
        fb.frontdoor_building_postcode,
        CASE WHEN NEW.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE NEW.frontdoor_building_announcement_search_price::bigint END,
        NEW.frontdoor_building_announcement_area,
        NEW.frontdoor_building_announcement_room_structure,
        NEW.frontdoor_building_announcement_last_seen_at,
        NULL::timestamptz,
        concat_ws(' ', NEW.frontdoor_building_announcement_id::text, NEW.frontdoor_building_announcement_external_id::text, NEW.frontdoor_building_announcement_friendly_id, NEW.frontdoor_building_announcement_address_line1, NEW.frontdoor_building_announcement_address_line2, NEW.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, NEW.frontdoor_building_announcement_room_structure),
        NEW.frontdoor_building_announcement_price_per_square,
        COALESCE(NEW.frontdoor_building_announcement_construction_finished_year, fb.frontdoor_building_build_year, fb.frontdoor_building_construction_end_year),
        fb.frontdoor_building_energy_certificate_code,
        now()
    FROM public.frontdoor_buildings fb
    WHERE fb.frontdoor_building_id = NEW.frontdoor_building_id
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
        sale_listing_published_at = EXCLUDED.sale_listing_published_at,
        sale_listing_search_text = EXCLUDED.sale_listing_search_text,
        sale_listing_price_per_m2 = EXCLUDED.sale_listing_price_per_m2,
        sale_listing_build_year = EXCLUDED.sale_listing_build_year,
        sale_listing_energy_class = EXCLUDED.sale_listing_energy_class,
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__link_sale_listing_prices_transaction(target_sale_listing_id uuid, transaction_id uuid)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    existing_listing_id uuid;
BEGIN
    IF target_sale_listing_id IS NULL THEN
        RAISE EXCEPTION 'target_sale_listing_id is required';
    END IF;
    IF transaction_id IS NULL THEN
        RAISE EXCEPTION 'transaction_id is required';
    END IF;
    PERFORM 1
    FROM public.sale_listings
    WHERE sale_listing_id = target_sale_listing_id
    FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'sale listing % not found', target_sale_listing_id;
    END IF;
    PERFORM 1
    FROM public.prices_transactions
    WHERE prices_transaction_id = transaction_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'prices transaction % not found', transaction_id;
    END IF;
    SELECT sale_listing_id INTO existing_listing_id
    FROM public.sale_listings
    WHERE prices_transaction_id = transaction_id
      AND sale_listing_id <> target_sale_listing_id;
    IF existing_listing_id IS NOT NULL THEN
        RAISE EXCEPTION 'prices transaction % is already linked to sale listing %', transaction_id, existing_listing_id;
    END IF;
    UPDATE public.sale_listings
    SET prices_transaction_id = transaction_id,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = target_sale_listing_id;
    RETURN target_sale_listing_id;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__unlink_sale_listing_prices_transaction(target_sale_listing_id uuid)
RETURNS uuid
LANGUAGE plpgsql
AS $$
BEGIN
    IF target_sale_listing_id IS NULL THEN
        RAISE EXCEPTION 'target_sale_listing_id is required';
    END IF;
    PERFORM 1
    FROM public.sale_listings
    WHERE sale_listing_id = target_sale_listing_id
    FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'sale listing % not found', target_sale_listing_id;
    END IF;
    UPDATE public.sale_listings
    SET prices_transaction_id = NULL,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = target_sale_listing_id;
    RETURN target_sale_listing_id;
END;
$$;
ALTER TABLE public.sale_listings DROP CONSTRAINT IF EXISTS sale_listings_public_id_key;
ALTER TABLE public.sale_listings DROP COLUMN IF EXISTS sale_listing_public_id;
