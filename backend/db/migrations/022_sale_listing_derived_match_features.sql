CREATE OR REPLACE FUNCTION public.fnc__derived_price_per_m2(price bigint, area double precision, existing double precision)
RETURNS double precision
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT COALESCE(existing, CASE WHEN price IS NOT NULL AND area IS NOT NULL AND area > 0 THEN price::double precision / area ELSE NULL END)
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
        sale_listing_public_id,
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
        'l_' || substr(md5('frontdoor:ad:' || NEW.frontdoor_ad_external_id), 1, 16),
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
        sale_listing_public_id = EXCLUDED.sale_listing_public_id,
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
UPDATE public.shortcut_ads
SET shortcut_ad_data = shortcut_ad_data
WHERE shortcut_ad_type = 'listing'
  AND shortcut_ad_data IS NOT NULL;
UPDATE public.frontdoor_ads
SET frontdoor_ad_data = frontdoor_ad_data
WHERE frontdoor_ad_data IS NOT NULL;
