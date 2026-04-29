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
DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_shortcut_ad ON public.shortcut_ads;
DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_shortcut_ad ON public.shortcut_ads;
CREATE TRIGGER trg__sync_sale_listing_from_shortcut_ad
AFTER INSERT OR UPDATE ON public.shortcut_ads
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_sale_listing_from_shortcut_ad();
CREATE TRIGGER trg__delete_sale_listing_from_shortcut_ad
BEFORE DELETE ON public.shortcut_ads
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_sale_listing_from_shortcut_ad();
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
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_frontdoor_ad ON public.frontdoor_ads;
DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_frontdoor_ad ON public.frontdoor_ads;
CREATE TRIGGER trg__sync_sale_listing_from_frontdoor_ad
AFTER INSERT OR UPDATE ON public.frontdoor_ads
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_sale_listing_from_frontdoor_ad();
CREATE TRIGGER trg__delete_sale_listing_from_frontdoor_ad
BEFORE DELETE ON public.frontdoor_ads
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_sale_listing_from_frontdoor_ad();
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
        sale_listing_public_id,
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
        sale_listing_updated_at
    )
    SELECT
        'l_' || substr(md5('frontdoor:announcement:' || NEW.frontdoor_building_announcement_id::text), 1, 16),
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
        now()
    FROM public.frontdoor_buildings fb
    WHERE fb.frontdoor_building_id = NEW.frontdoor_building_id
    ON CONFLICT (sale_listing_canonical_id) DO UPDATE SET
        sale_listing_public_id = EXCLUDED.sale_listing_public_id,
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
        sale_listing_updated_at = now();
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_frontdoor_announcement ON public.frontdoor_building_announcements;
DROP TRIGGER IF EXISTS trg__delete_sale_listing_from_frontdoor_announcement ON public.frontdoor_building_announcements;
CREATE TRIGGER trg__sync_sale_listing_from_frontdoor_announcement
AFTER INSERT OR UPDATE ON public.frontdoor_building_announcements
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_sale_listing_from_frontdoor_announcement();
CREATE TRIGGER trg__delete_sale_listing_from_frontdoor_announcement
BEFORE DELETE ON public.frontdoor_building_announcements
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_sale_listing_from_frontdoor_announcement();
CREATE OR REPLACE FUNCTION public.fnc__refresh_sale_listings_from_shortcut_building()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE public.sale_listings sl
    SET
        sale_listing_headline = COALESCE(sa.shortcut_ad_street_address, NEW.shortcut_building_address, sa.shortcut_ad_id::text),
        sale_listing_street_address = COALESCE(sa.shortcut_ad_street_address, NEW.shortcut_building_address),
        sale_listing_search_text = concat_ws(' ', sa.shortcut_ad_search_text, NEW.shortcut_building_address, NEW.shortcut_building_housing_company),
        sale_listing_updated_at = now()
    FROM public.shortcut_ads sa
    WHERE sl.shortcut_ad_id = sa.shortcut_ad_id
      AND sa.shortcut_building_id = NEW.shortcut_building_id
      AND sa.shortcut_ad_type = 'listing';
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__refresh_sale_listings_from_shortcut_building ON public.shortcut_buildings;
CREATE TRIGGER trg__refresh_sale_listings_from_shortcut_building
AFTER UPDATE ON public.shortcut_buildings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__refresh_sale_listings_from_shortcut_building();
CREATE OR REPLACE FUNCTION public.fnc__refresh_sale_listings_from_frontdoor_building()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE public.sale_listings sl
    SET
        sale_listing_url = NEW.frontdoor_building_url,
        sale_listing_city = COALESCE(fba.frontdoor_building_announcement_location, NEW.frontdoor_building_municipality, NEW.frontdoor_building_post_area),
        sale_listing_postal = NEW.frontdoor_building_postcode,
        sale_listing_search_text = concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, NEW.frontdoor_building_postcode, NEW.frontdoor_building_municipality, NEW.frontdoor_building_post_area, NEW.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure),
        sale_listing_updated_at = now()
    FROM public.frontdoor_building_announcements fba
    WHERE sl.frontdoor_building_announcement_id = fba.frontdoor_building_announcement_id
      AND fba.frontdoor_building_id = NEW.frontdoor_building_id
      AND fba.frontdoor_building_announcement_rent_period IS NULL
      AND fba.frontdoor_building_announcement_rental_unique_no IS NULL;
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__refresh_sale_listings_from_frontdoor_building ON public.frontdoor_buildings;
CREATE TRIGGER trg__refresh_sale_listings_from_frontdoor_building
AFTER UPDATE ON public.frontdoor_buildings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__refresh_sale_listings_from_frontdoor_building();
