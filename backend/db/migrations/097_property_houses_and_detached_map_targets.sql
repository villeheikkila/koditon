CREATE TABLE IF NOT EXISTS public.property_houses (
    property_house_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_house_identity_key text NOT NULL UNIQUE,
    property_house_address_norm text,
    property_house_postal_norm text,
    property_house_city_norm text,
    property_house_build_year integer,
    property_house_area_value double precision,
    property_house_plot_area_value double precision,
    property_house_rooms_count integer,
    property_house_latitude double precision,
    property_house_longitude double precision,
    property_house_match_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    primary_sale_listing_id uuid REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE SET NULL,
    property_house_created_at timestamptz NOT NULL DEFAULT now(),
    property_house_updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_property_houses_lat_lng
ON public.property_houses (property_house_latitude, property_house_longitude)
WHERE property_house_latitude IS NOT NULL
  AND property_house_longitude IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_property_houses_address
ON public.property_houses (property_house_postal_norm, property_house_city_norm, property_house_address_norm);
ALTER TABLE public.property_offerings
    ADD COLUMN IF NOT EXISTS property_house_id uuid REFERENCES public.property_houses(property_house_id) ON DELETE CASCADE;
ALTER TABLE public.property_offerings
    ALTER COLUMN property_unit_id DROP NOT NULL;
ALTER TABLE public.property_offerings
    DROP CONSTRAINT IF EXISTS property_offerings_parent_check;
ALTER TABLE public.property_offerings
    ADD CONSTRAINT property_offerings_parent_check CHECK (
        ((property_unit_id IS NOT NULL)::integer + (property_house_id IS NOT NULL)::integer) = 1
    );
CREATE INDEX IF NOT EXISTS idx_property_offerings_house
ON public.property_offerings (property_house_id)
WHERE property_house_id IS NOT NULL;
ALTER TABLE public.property_target_sources
    DROP CONSTRAINT IF EXISTS property_target_sources_target_type_check;
ALTER TABLE public.property_target_sources
    ADD CONSTRAINT property_target_sources_target_type_check CHECK (
        target_type = ANY (ARRAY['offering','unit','building','housing_company','house','document','transaction']::text[])
    );
ALTER TABLE public.property_dimension_claims
    DROP CONSTRAINT IF EXISTS property_dimension_claims_target_type_check;
ALTER TABLE public.property_dimension_claims
    ADD CONSTRAINT property_dimension_claims_target_type_check CHECK (
        target_type = ANY (ARRAY['listing','document','offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_dimension_values
    DROP CONSTRAINT IF EXISTS property_dimension_values_target_type_check;
ALTER TABLE public.property_dimension_values
    ADD CONSTRAINT property_dimension_values_target_type_check CHECK (
        target_type = ANY (ARRAY['offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_dimension_dirty_targets
    DROP CONSTRAINT IF EXISTS property_dimension_dirty_targets_target_type_check;
ALTER TABLE public.property_dimension_dirty_targets
    ADD CONSTRAINT property_dimension_dirty_targets_target_type_check CHECK (
        target_type = ANY (ARRAY['listing','document','transaction','offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_dimension_catalog
    DROP CONSTRAINT IF EXISTS property_dimension_catalog_target_type_check;
ALTER TABLE public.property_dimension_catalog
    ADD CONSTRAINT property_dimension_catalog_target_type_check CHECK (
        target_type = ANY (ARRAY['offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_dimension_manual_overrides
    DROP CONSTRAINT IF EXISTS property_dimension_manual_overrides_target_type_check;
ALTER TABLE public.property_dimension_manual_overrides
    ADD CONSTRAINT property_dimension_manual_overrides_target_type_check CHECK (
        target_type = ANY (ARRAY['offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_dimension_profiles
    DROP CONSTRAINT IF EXISTS property_dimension_profiles_target_type_check;
ALTER TABLE public.property_dimension_profiles
    ADD CONSTRAINT property_dimension_profiles_target_type_check CHECK (
        target_type = ANY (ARRAY['offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_renovation_events
    DROP CONSTRAINT IF EXISTS property_renovation_events_target_type_check;
ALTER TABLE public.property_renovation_events
    ADD CONSTRAINT property_renovation_events_target_type_check CHECK (
        target_type = ANY (ARRAY['listing','document','offering','unit','building','housing_company','house']::text[])
    );
ALTER TABLE public.property_system_profiles
    DROP CONSTRAINT IF EXISTS property_system_profiles_target_type_check;
ALTER TABLE public.property_system_profiles
    ADD CONSTRAINT property_system_profiles_target_type_check CHECK (
        target_type = ANY (ARRAY['unit','building','housing_company','house']::text[])
    );
CREATE OR REPLACE FUNCTION public.fnc__sync_property_house_for_sale_listing(listing_id uuid, link_method text DEFAULT 'sync_auto')
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    house_id uuid;
    offering_id uuid;
    house_key text;
BEGIN
    SELECT po.property_offering_id
    INTO offering_id
    FROM public.property_offering_sources pos
    JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
    WHERE pos.sale_listing_id = listing_id
        AND pos.property_offering_source_link_status <> 'rejected'
    ORDER BY pos.property_offering_source_link_score DESC, pos.property_offering_source_updated_at DESC
    LIMIT 1;
    IF offering_id IS NULL THEN
        RETURN NULL;
    END IF;
    SELECT COALESCE(
        'detached_address:' || public.fnc__canonical_identity_part(concat_ws('|', sale_listing_postal_norm, sale_listing_city_norm, sale_listing_building_match_key, sale_listing_area_value::text)),
        'detached_source:' || sale_listing_source_provider || ':' || sale_listing_source_kind || ':' || sale_listing_native_id
    )
    INTO house_key
    FROM public.property_source_offerings
    WHERE sale_listing_id = listing_id
        AND sale_listing_property_type_code = 'detached_house';
    IF house_key IS NULL THEN
        RETURN NULL;
    END IF;
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
        house_key,
        sale_listing_address_norm,
        sale_listing_postal_norm,
        sale_listing_city_norm,
        sale_listing_build_year,
        COALESCE(sale_listing_living_area_value, sale_listing_area_value),
        sale_listing_plot_area_value,
        sale_listing_rooms_count,
        sale_listing_latitude,
        sale_listing_longitude,
        jsonb_build_object('source', sale_listing_source_provider, 'method', link_method, 'source_listing_id', sale_listing_id),
        sale_listing_id,
        now()
    FROM public.property_source_offerings
    WHERE sale_listing_id = listing_id
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
    RETURNING property_house_id INTO house_id;
    UPDATE public.property_offerings
    SET property_house_id = house_id,
        property_unit_id = NULL,
        property_offering_updated_at = now()
    WHERE property_offering_id = offering_id;
    INSERT INTO public.property_target_sources (
        target_type,
        target_id,
        source_provider,
        source_kind,
        source_table,
        source_id,
        source_id_value,
        source_external_id,
        source_url,
        link_status,
        link_method,
        link_score,
        link_reasons,
        first_seen_at,
        last_seen_at
    )
    SELECT
        'house',
        house_id,
        sale_listing_source_provider,
        sale_listing_source_kind,
        'property_source_offerings',
        sale_listing_id,
        sale_listing_id::text,
        sale_listing_native_id,
        sale_listing_url,
        'confirmed',
        link_method,
        100,
        jsonb_build_object('source', 'detached_house_listing'),
        sale_listing_first_seen_at,
        sale_listing_last_seen_at
    FROM public.property_source_offerings
    WHERE sale_listing_id = listing_id
    ON CONFLICT (target_type, target_id, source_provider, source_kind, source_table, source_id_value) DO UPDATE SET
        source_id = COALESCE(EXCLUDED.source_id, property_target_sources.source_id),
        source_external_id = COALESCE(EXCLUDED.source_external_id, property_target_sources.source_external_id),
        source_url = COALESCE(EXCLUDED.source_url, property_target_sources.source_url),
        link_status = EXCLUDED.link_status,
        link_method = EXCLUDED.link_method,
        link_score = EXCLUDED.link_score,
        link_reasons = property_target_sources.link_reasons || EXCLUDED.link_reasons,
        first_seen_at = LEAST(COALESCE(property_target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, property_target_sources.first_seen_at)),
        last_seen_at = GREATEST(COALESCE(property_target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, property_target_sources.last_seen_at)),
        updated_at = now();
    INSERT INTO public.property_dimension_dirty_targets (target_type, target_id, dirty_reasons, dirty_at)
    VALUES ('house', house_id, ARRAY['detached_house_regroup'], now())
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = ARRAY(SELECT DISTINCT unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons)),
        dirty_at = GREATEST(property_dimension_dirty_targets.dirty_at, EXCLUDED.dirty_at);
    RETURN house_id;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_property_house_for_sale_listing_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM public.fnc__sync_property_house_for_sale_listing(NEW.sale_listing_id, 'sync_auto');
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__sync_property_house_for_sale_listing ON public.property_source_offerings;
CREATE TRIGGER trg__sync_property_house_for_sale_listing
AFTER INSERT OR UPDATE OF sale_listing_property_type_code, sale_listing_address_norm, sale_listing_building_match_key, sale_listing_area_value, sale_listing_latitude, sale_listing_longitude, sale_listing_last_seen_at
ON public.property_source_offerings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_property_house_for_sale_listing_trigger();
