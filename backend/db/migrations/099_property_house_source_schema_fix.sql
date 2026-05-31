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
        sl.sale_listing_address_norm,
        sl.sale_listing_postal_norm,
        sl.sale_listing_city_norm,
        sl.sale_listing_build_year,
        COALESCE(sl.sale_listing_living_area_value, sl.sale_listing_area_value),
        sl.sale_listing_plot_area_value,
        sl.sale_listing_rooms_count,
        COALESCE(sl.sale_listing_latitude, pb.physical_building_latitude, postgis.ST_Y(hc.housing_company_geom)),
        COALESCE(sl.sale_listing_longitude, pb.physical_building_longitude, postgis.ST_X(hc.housing_company_geom)),
        jsonb_build_object('source', sl.sale_listing_source_provider, 'method', link_method, 'source_listing_id', sl.sale_listing_id),
        sl.sale_listing_id,
        now()
    FROM public.property_source_offerings sl
    LEFT JOIN public.property_offerings current_po ON current_po.property_offering_id = offering_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = current_po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    LEFT JOIN public.housing_companies hc ON hc.housing_company_id = COALESCE(pu.housing_company_id, pb.housing_company_id)
    WHERE sl.sale_listing_id = listing_id
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
        jsonb_build_object('source', 'detached_house_listing', 'identity_key', house_key),
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
