CREATE TABLE IF NOT EXISTS public.sale_listing_property_type_aliases (
    sale_listing_property_type_alias text PRIMARY KEY,
    sale_listing_property_type_code text NOT NULL,
    sale_listing_property_type_label text NOT NULL
);
CREATE TABLE IF NOT EXISTS public.sale_listing_plot_type_aliases (
    sale_listing_plot_type_alias text PRIMARY KEY,
    sale_listing_plot_type_code text NOT NULL,
    sale_listing_plot_type_label text NOT NULL
);
CREATE TABLE IF NOT EXISTS public.sale_listing_room_category_aliases (
    sale_listing_room_category_alias text PRIMARY KEY,
    sale_listing_room_category_code text NOT NULL,
    sale_listing_room_category_label text NOT NULL
);
INSERT INTO public.sale_listing_property_type_aliases (
    sale_listing_property_type_alias,
    sale_listing_property_type_code,
    sale_listing_property_type_label
)
VALUES
    ('kt', 'apartment_block', 'Kerrostalo'),
    ('kerrostalo', 'apartment_block', 'Kerrostalo'),
    ('apartment', 'apartment_block', 'Kerrostalo'),
    ('apartment_house', 'apartment_block', 'Kerrostalo'),
    ('apartment_block', 'apartment_block', 'Kerrostalo'),
    ('balcony_access_block', 'apartment_block', 'Kerrostalo'),
    ('block_of_flats', 'apartment_block', 'Kerrostalo'),
    ('flat', 'apartment_block', 'Kerrostalo'),
    ('wooden_house_apartment', 'apartment_block', 'Kerrostalo'),
    ('1', 'apartment_block', 'Kerrostalo'),
    ('rt', 'row_house', 'Rivitalo'),
    ('rivitalo', 'row_house', 'Rivitalo'),
    ('row_house', 'row_house', 'Rivitalo'),
    ('semi_detached_house', 'row_house', 'Rivitalo'),
    ('terraced_house', 'row_house', 'Rivitalo'),
    ('terrace_house', 'row_house', 'Rivitalo'),
    ('2', 'row_house', 'Rivitalo'),
    ('ok', 'detached_house', 'Omakotitalo'),
    ('omakotitalo', 'detached_house', 'Omakotitalo'),
    ('detached_house', 'detached_house', 'Omakotitalo'),
    ('separate_house', 'detached_house', 'Omakotitalo'),
    ('single_family_house', 'detached_house', 'Omakotitalo'),
    ('3', 'detached_house', 'Omakotitalo')
ON CONFLICT (sale_listing_property_type_alias) DO UPDATE SET
    sale_listing_property_type_code = EXCLUDED.sale_listing_property_type_code,
    sale_listing_property_type_label = EXCLUDED.sale_listing_property_type_label;
INSERT INTO public.sale_listing_plot_type_aliases (
    sale_listing_plot_type_alias,
    sale_listing_plot_type_code,
    sale_listing_plot_type_label
)
VALUES
    ('oma', 'own', 'Oma'),
    ('own', 'own', 'Oma'),
    ('owned', 'own', 'Oma'),
    ('vuokra', 'rent', 'Vuokra'),
    ('rent', 'rent', 'Vuokra'),
    ('rental', 'rent', 'Vuokra'),
    ('lease', 'rent', 'Vuokra'),
    ('leased', 'rent', 'Vuokra'),
    ('optional_rental', 'rent', 'Vuokra'),
    ('valinnainen_vuokratontti', 'rent', 'Vuokra'),
    ('vuokralla', 'rent', 'Vuokra')
ON CONFLICT (sale_listing_plot_type_alias) DO UPDATE SET
    sale_listing_plot_type_code = EXCLUDED.sale_listing_plot_type_code,
    sale_listing_plot_type_label = EXCLUDED.sale_listing_plot_type_label;
INSERT INTO public.sale_listing_room_category_aliases (
    sale_listing_room_category_alias,
    sale_listing_room_category_code,
    sale_listing_room_category_label
)
VALUES
    ('yksiöt', 'one_room', 'Yksiöt'),
    ('yksiot', 'one_room', 'Yksiöt'),
    ('one_room', 'one_room', 'Yksiöt'),
    ('kaksiot', 'two_rooms', 'Kaksiot'),
    ('kaksiöt', 'two_rooms', 'Kaksiot'),
    ('two_rooms', 'two_rooms', 'Kaksiot'),
    ('kolmiot', 'three_rooms', 'Kolmiot'),
    ('three_rooms', 'three_rooms', 'Kolmiot'),
    ('neljä_huonetta_tai_enemmän', 'four_plus_rooms', 'Neljä huonetta tai enemmän'),
    ('nelja_huonetta_tai_enemman', 'four_plus_rooms', 'Neljä huonetta tai enemmän'),
    ('four_plus_rooms', 'four_plus_rooms', 'Neljä huonetta tai enemmän')
ON CONFLICT (sale_listing_room_category_alias) DO UPDATE SET
    sale_listing_room_category_code = EXCLUDED.sale_listing_room_category_code,
    sale_listing_room_category_label = EXCLUDED.sale_listing_room_category_label;
CREATE OR REPLACE FUNCTION public.fnc__match_alias_key(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF(regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]åäö]+', '_', 'g'), '')
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listing_property_type_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT a.sale_listing_property_type_code
    FROM public.sale_listing_property_type_aliases a
    WHERE a.sale_listing_property_type_alias = public.fnc__match_alias_key(value)
    LIMIT 1
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listing_plot_type_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT a.sale_listing_plot_type_code
    FROM public.sale_listing_plot_type_aliases a
    WHERE a.sale_listing_plot_type_alias = public.fnc__match_alias_key(value)
    LIMIT 1
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listing_room_category_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT a.sale_listing_room_category_code
    FROM public.sale_listing_room_category_aliases a
    WHERE a.sale_listing_room_category_alias = public.fnc__match_alias_key(value)
    LIMIT 1
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listing_room_category_code(rooms integer, room_layout text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT CASE
        WHEN rooms = 1 THEN 'one_room'
        WHEN rooms = 2 THEN 'two_rooms'
        WHEN rooms = 3 THEN 'three_rooms'
        WHEN rooms >= 4 THEN 'four_plus_rooms'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])1\s*h' THEN 'one_room'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])2\s*h' THEN 'two_rooms'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])3\s*h' THEN 'three_rooms'
        WHEN lower(COALESCE(room_layout, '')) ~ '(^|[^0-9])[4-9]\s*h' THEN 'four_plus_rooms'
        ELSE NULL
    END
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listing_floor_text(floor_level integer, total_floors integer)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN floor_level IS NULL THEN NULL
        WHEN total_floors IS NULL THEN floor_level::text
        ELSE floor_level::text || '/' || total_floors::text
    END
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_floor_level(value text)
RETURNS integer
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF((regexp_match(COALESCE(value, ''), '^\s*(-?[0-9]+)(?:\s*/\s*-?[0-9]+)?\s*$'))[1], '')::integer
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_total_floors(value text)
RETURNS integer
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF((regexp_match(COALESCE(value, ''), '^\s*-?[0-9]+\s*/\s*([0-9]+)\s*$'))[1], '')::integer
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_floor_text(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT public.fnc__sale_listing_floor_text(
        public.fnc__prices_transaction_floor_level(value),
        public.fnc__prices_transaction_total_floors(value)
    )
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_property_type_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT public.fnc__sale_listing_property_type_code(value)
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_plot_type_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT public.fnc__sale_listing_plot_type_code(value)
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_room_category_code(value text)
RETURNS text
LANGUAGE sql
STABLE
AS $$
    SELECT public.fnc__sale_listing_room_category_code(value)
$$;
ALTER TABLE public.sale_listings
ADD COLUMN IF NOT EXISTS sale_listing_property_type_raw text,
ADD COLUMN IF NOT EXISTS sale_listing_property_type_code text,
ADD COLUMN IF NOT EXISTS sale_listing_room_category_code text,
ADD COLUMN IF NOT EXISTS sale_listing_floor_text text,
ADD COLUMN IF NOT EXISTS sale_listing_elevator boolean,
ADD COLUMN IF NOT EXISTS sale_listing_plot_type_raw text,
ADD COLUMN IF NOT EXISTS sale_listing_plot_type_code text;
CREATE INDEX IF NOT EXISTS idx_sale_listings_property_type_code ON public.sale_listings (sale_listing_property_type_code);
CREATE INDEX IF NOT EXISTS idx_sale_listings_room_category_code ON public.sale_listings (sale_listing_room_category_code);
CREATE INDEX IF NOT EXISTS idx_sale_listings_elevator ON public.sale_listings (sale_listing_elevator);
CREATE INDEX IF NOT EXISTS idx_sale_listings_plot_type_code ON public.sale_listings (sale_listing_plot_type_code);
CREATE OR REPLACE FUNCTION public.fnc__sale_listings_set_transaction_match_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    property_raw text;
    plot_raw text;
    elevator_value boolean;
BEGIN
    IF NEW.shortcut_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), ''),
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sb.shortcut_building_plot_type)), ''),
            COALESCE(sa.shortcut_ad_elevator, public.fnc__try_parse_bool(sb.shortcut_building_has_elevator))
        INTO property_raw, plot_raw, elevator_value
        FROM public.shortcut_ads sa
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        WHERE sa.shortcut_ad_id = NEW.shortcut_ad_id;
    ELSIF NEW.frontdoor_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), ''),
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''),
            fa.frontdoor_ad_elevator
        INTO property_raw, plot_raw, elevator_value
        FROM public.frontdoor_ads fa
        WHERE fa.frontdoor_ad_id = NEW.frontdoor_ad_id;
    ELSIF NEW.frontdoor_building_announcement_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
            NULL::text,
            fb.frontdoor_building_has_elevator
        INTO property_raw, plot_raw, elevator_value
        FROM public.frontdoor_building_announcements fba
        JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE fba.frontdoor_building_announcement_id = NEW.frontdoor_building_announcement_id;
    END IF;
    NEW.sale_listing_property_type_raw := property_raw;
    NEW.sale_listing_property_type_code := public.fnc__sale_listing_property_type_code(property_raw);
    NEW.sale_listing_room_category_code := public.fnc__sale_listing_room_category_code(NEW.sale_listing_rooms_count, NEW.sale_listing_room_layout);
    NEW.sale_listing_floor_text := public.fnc__sale_listing_floor_text(NEW.sale_listing_floor_level, NEW.sale_listing_total_floors);
    NEW.sale_listing_elevator := elevator_value;
    NEW.sale_listing_plot_type_raw := plot_raw;
    NEW.sale_listing_plot_type_code := public.fnc__sale_listing_plot_type_code(plot_raw);
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__sale_listings_set_transaction_match_fields ON public.sale_listings;
UPDATE public.sale_listings sl
SET
    sale_listing_property_type_raw = NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), ''),
    sale_listing_property_type_code = public.fnc__sale_listing_property_type_code(NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), '')),
    sale_listing_room_category_code = public.fnc__sale_listing_room_category_code(sl.sale_listing_rooms_count, sl.sale_listing_room_layout),
    sale_listing_floor_text = public.fnc__sale_listing_floor_text(sl.sale_listing_floor_level, sl.sale_listing_total_floors),
    sale_listing_elevator = COALESCE(sa.shortcut_ad_elevator, public.fnc__try_parse_bool(sb.shortcut_building_has_elevator)),
    sale_listing_plot_type_raw = NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sb.shortcut_building_plot_type)), ''),
    sale_listing_plot_type_code = public.fnc__sale_listing_plot_type_code(NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sb.shortcut_building_plot_type)), ''))
FROM public.shortcut_ads sa
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
WHERE sl.shortcut_ad_id = sa.shortcut_ad_id;
UPDATE public.sale_listings sl
SET
    sale_listing_property_type_raw = NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), ''),
    sale_listing_property_type_code = public.fnc__sale_listing_property_type_code(NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), '')),
    sale_listing_room_category_code = public.fnc__sale_listing_room_category_code(sl.sale_listing_rooms_count, sl.sale_listing_room_layout),
    sale_listing_floor_text = public.fnc__sale_listing_floor_text(sl.sale_listing_floor_level, sl.sale_listing_total_floors),
    sale_listing_elevator = fa.frontdoor_ad_elevator,
    sale_listing_plot_type_raw = NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''),
    sale_listing_plot_type_code = public.fnc__sale_listing_plot_type_code(NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''))
FROM public.frontdoor_ads fa
WHERE sl.frontdoor_ad_id = fa.frontdoor_ad_id;
UPDATE public.sale_listings sl
SET
    sale_listing_property_type_raw = NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
    sale_listing_property_type_code = public.fnc__sale_listing_property_type_code(NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), '')),
    sale_listing_room_category_code = public.fnc__sale_listing_room_category_code(sl.sale_listing_rooms_count, sl.sale_listing_room_layout),
    sale_listing_floor_text = public.fnc__sale_listing_floor_text(sl.sale_listing_floor_level, sl.sale_listing_total_floors),
    sale_listing_elevator = fb.frontdoor_building_has_elevator,
    sale_listing_plot_type_raw = NULL,
    sale_listing_plot_type_code = NULL
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE sl.frontdoor_building_announcement_id = fba.frontdoor_building_announcement_id;
CREATE TRIGGER trg__sale_listings_set_transaction_match_fields
BEFORE INSERT OR UPDATE ON public.sale_listings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sale_listings_set_transaction_match_fields();
