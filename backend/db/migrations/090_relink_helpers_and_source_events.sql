DELETE FROM public.property_renovation_events
WHERE event_scope = 'canonical';
ALTER TABLE public.property_renovation_events
    DROP CONSTRAINT IF EXISTS property_renovation_events_event_scope_check;
ALTER TABLE public.property_renovation_events
    ADD CONSTRAINT property_renovation_events_event_scope_check
    CHECK (event_scope = ANY (ARRAY['source','manual']::text[]));
CREATE OR REPLACE FUNCTION public.fnc__mark_property_offering_dimension_targets_dirty(p_property_offering_id uuid, p_reason text DEFAULT 'offering_link_changed')
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_count integer := 0;
    v_target record;
BEGIN
    FOR v_target IN
        SELECT 'offering'::text AS target_type, po.property_offering_id AS target_id
        FROM public.property_offerings po
        WHERE po.property_offering_id = p_property_offering_id
        UNION
        SELECT 'unit', po.property_unit_id
        FROM public.property_offerings po
        WHERE po.property_offering_id = p_property_offering_id
        UNION
        SELECT 'building', pu.physical_building_id
        FROM public.property_offerings po
        JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
        WHERE po.property_offering_id = p_property_offering_id
            AND pu.physical_building_id IS NOT NULL
        UNION
        SELECT 'housing_company', COALESCE(pu.housing_company_id, pb.housing_company_id)
        FROM public.property_offerings po
        JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
        LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
        WHERE po.property_offering_id = p_property_offering_id
            AND COALESCE(pu.housing_company_id, pb.housing_company_id) IS NOT NULL
        UNION
        SELECT 'listing', pos.sale_listing_id
        FROM public.property_offering_sources pos
        WHERE pos.property_offering_id = p_property_offering_id
            AND pos.property_offering_source_link_status <> 'rejected'
    LOOP
        v_count := v_count + public.fnc__mark_property_dimension_target_dirty(v_target.target_type, v_target.target_id, p_reason);
    END LOOP;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__mark_property_unit_dimension_targets_dirty(p_property_unit_id uuid, p_reason text DEFAULT 'unit_link_changed')
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_count integer := 0;
    v_target record;
BEGIN
    FOR v_target IN
        SELECT 'unit'::text AS target_type, pu.property_unit_id AS target_id
        FROM public.property_units pu
        WHERE pu.property_unit_id = p_property_unit_id
        UNION
        SELECT 'building', pu.physical_building_id
        FROM public.property_units pu
        WHERE pu.property_unit_id = p_property_unit_id
            AND pu.physical_building_id IS NOT NULL
        UNION
        SELECT 'housing_company', COALESCE(pu.housing_company_id, pb.housing_company_id)
        FROM public.property_units pu
        LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
        WHERE pu.property_unit_id = p_property_unit_id
            AND COALESCE(pu.housing_company_id, pb.housing_company_id) IS NOT NULL
        UNION
        SELECT 'offering', po.property_offering_id
        FROM public.property_offerings po
        WHERE po.property_unit_id = p_property_unit_id
        UNION
        SELECT 'listing', pos.sale_listing_id
        FROM public.property_offerings po
        JOIN public.property_offering_sources pos ON pos.property_offering_id = po.property_offering_id
        WHERE po.property_unit_id = p_property_unit_id
            AND pos.property_offering_source_link_status <> 'rejected'
    LOOP
        v_count := v_count + public.fnc__mark_property_dimension_target_dirty(v_target.target_type, v_target.target_id, p_reason);
    END LOOP;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__relink_property_offering_source(
    p_sale_listing_id uuid,
    p_target_property_offering_id uuid,
    p_method text DEFAULT 'manual',
    p_score integer DEFAULT 100,
    p_reasons jsonb DEFAULT '{}'::jsonb
)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_offering_ids uuid[];
    v_dirty integer := 0;
    v_old_id uuid;
BEGIN
    SELECT COALESCE(array_agg(property_offering_id), ARRAY[]::uuid[])
    INTO v_old_offering_ids
    FROM public.property_offering_sources
    WHERE sale_listing_id = p_sale_listing_id
        AND property_offering_source_link_status <> 'rejected';
    UPDATE public.property_offering_sources
    SET property_offering_source_link_status = 'rejected',
        property_offering_source_updated_at = now()
    WHERE sale_listing_id = p_sale_listing_id
        AND property_offering_id <> p_target_property_offering_id
        AND property_offering_source_link_status <> 'rejected';
    INSERT INTO public.property_offering_sources (
        property_offering_id,
        sale_listing_id,
        property_offering_source_link_status,
        property_offering_source_link_method,
        property_offering_source_link_score,
        property_offering_source_link_reasons,
        property_offering_source_updated_at
    )
    VALUES (
        p_target_property_offering_id,
        p_sale_listing_id,
        'confirmed',
        COALESCE(NULLIF(p_method, ''), 'manual'),
        COALESCE(p_score, 100),
        COALESCE(p_reasons, '{}'::jsonb),
        now()
    )
    ON CONFLICT (sale_listing_id) WHERE property_offering_source_link_status <> 'rejected'
    DO UPDATE SET
        property_offering_id = EXCLUDED.property_offering_id,
        property_offering_source_link_status = EXCLUDED.property_offering_source_link_status,
        property_offering_source_link_method = EXCLUDED.property_offering_source_link_method,
        property_offering_source_link_score = EXCLUDED.property_offering_source_link_score,
        property_offering_source_link_reasons = property_offering_sources.property_offering_source_link_reasons || EXCLUDED.property_offering_source_link_reasons,
        property_offering_source_updated_at = now();
    FOREACH v_old_id IN ARRAY v_old_offering_ids LOOP
        v_dirty := v_dirty + public.fnc__mark_property_offering_dimension_targets_dirty(v_old_id, 'offering_source_relinked_old');
    END LOOP;
    v_dirty := v_dirty + public.fnc__mark_property_offering_dimension_targets_dirty(p_target_property_offering_id, 'offering_source_relinked_new');
    v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('listing', p_sale_listing_id, 'offering_source_relinked');
    RETURN jsonb_build_object(
        'sale_listing_id', p_sale_listing_id,
        'old_property_offering_ids', v_old_offering_ids,
        'new_property_offering_id', p_target_property_offering_id,
        'dirty_targets', v_dirty
    );
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__relink_property_unit_building(
    p_property_unit_id uuid,
    p_target_physical_building_id uuid,
    p_reason text DEFAULT 'unit_building_relinked'
)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_building_id uuid;
    v_old_housing_company_id uuid;
    v_new_housing_company_id uuid;
    v_dirty integer := 0;
BEGIN
    SELECT pu.physical_building_id, pu.housing_company_id
    INTO v_old_building_id, v_old_housing_company_id
    FROM public.property_units pu
    WHERE pu.property_unit_id = p_property_unit_id
    FOR UPDATE;
    SELECT pb.housing_company_id
    INTO v_new_housing_company_id
    FROM public.physical_buildings pb
    WHERE pb.physical_building_id = p_target_physical_building_id;
    UPDATE public.property_units
    SET physical_building_id = p_target_physical_building_id,
        housing_company_id = COALESCE(v_new_housing_company_id, housing_company_id),
        property_unit_updated_at = now()
    WHERE property_unit_id = p_property_unit_id;
    v_dirty := v_dirty + public.fnc__mark_property_unit_dimension_targets_dirty(p_property_unit_id, p_reason);
    IF v_old_building_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('building', v_old_building_id, p_reason || '_old');
    END IF;
    IF v_old_housing_company_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('housing_company', v_old_housing_company_id, p_reason || '_old');
    END IF;
    RETURN jsonb_build_object(
        'property_unit_id', p_property_unit_id,
        'old_physical_building_id', v_old_building_id,
        'new_physical_building_id', p_target_physical_building_id,
        'old_housing_company_id', v_old_housing_company_id,
        'new_housing_company_id', v_new_housing_company_id,
        'dirty_targets', v_dirty
    );
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__relink_physical_building_housing_company(
    p_physical_building_id uuid,
    p_target_housing_company_id uuid,
    p_reason text DEFAULT 'building_housing_company_relinked'
)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_old_housing_company_id uuid;
    v_dirty integer := 0;
    v_unit record;
BEGIN
    SELECT housing_company_id
    INTO v_old_housing_company_id
    FROM public.physical_buildings
    WHERE physical_building_id = p_physical_building_id
    FOR UPDATE;
    UPDATE public.physical_buildings
    SET housing_company_id = p_target_housing_company_id,
        physical_building_updated_at = now()
    WHERE physical_building_id = p_physical_building_id;
    UPDATE public.property_units
    SET housing_company_id = p_target_housing_company_id,
        property_unit_updated_at = now()
    WHERE physical_building_id = p_physical_building_id;
    v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('building', p_physical_building_id, p_reason);
    IF v_old_housing_company_id IS NOT NULL THEN
        v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('housing_company', v_old_housing_company_id, p_reason || '_old');
    END IF;
    v_dirty := v_dirty + public.fnc__mark_property_dimension_target_dirty('housing_company', p_target_housing_company_id, p_reason || '_new');
    FOR v_unit IN
        SELECT pu.property_unit_id
        FROM public.property_units pu
        WHERE pu.physical_building_id = p_physical_building_id
    LOOP
        v_dirty := v_dirty + public.fnc__mark_property_unit_dimension_targets_dirty(v_unit.property_unit_id, p_reason);
    END LOOP;
    RETURN jsonb_build_object(
        'physical_building_id', p_physical_building_id,
        'old_housing_company_id', v_old_housing_company_id,
        'new_housing_company_id', p_target_housing_company_id,
        'dirty_targets', v_dirty
    );
END;
$$;
