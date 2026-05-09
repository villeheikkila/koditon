CREATE TABLE IF NOT EXISTS public.property_dimension_dirty_targets (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dirty_reasons text[] NOT NULL DEFAULT '{}',
    dirty_at timestamptz NOT NULL DEFAULT now(),
    queued_at timestamptz,
    resolved_at timestamptz,
    PRIMARY KEY (target_type, target_id),
    CONSTRAINT property_dimension_dirty_targets_target_type_check CHECK (target_type = ANY (ARRAY['listing'::text, 'document'::text, 'transaction'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text]))
);
CREATE INDEX IF NOT EXISTS idx_property_dimension_dirty_targets_queue
ON public.property_dimension_dirty_targets (dirty_at)
WHERE resolved_at IS NULL OR resolved_at < dirty_at;
CREATE OR REPLACE FUNCTION public.fnc__mark_property_dimension_target_dirty(p_target_type text, p_target_id uuid, p_reason text DEFAULT 'changed')
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    IF p_target_type IS NULL OR p_target_id IS NULL THEN
        RETURN 0;
    END IF;
    INSERT INTO public.property_dimension_dirty_targets (
        target_type,
        target_id,
        dirty_reasons,
        dirty_at,
        resolved_at
    )
    VALUES (
        p_target_type,
        p_target_id,
        ARRAY[COALESCE(NULLIF(p_reason, ''), 'changed')],
        now(),
        NULL
    )
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dirty_reasons = (
            SELECT array_agg(DISTINCT reason ORDER BY reason)
            FROM unnest(property_dimension_dirty_targets.dirty_reasons || EXCLUDED.dirty_reasons) AS reason
        ),
        dirty_at = now(),
        resolved_at = NULL;
    RETURN 1;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__mark_listing_dimension_targets_dirty(p_sale_listing_id uuid, p_reason text DEFAULT 'listing_changed')
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_count integer := 0;
    v_target record;
BEGIN
    IF p_sale_listing_id IS NULL THEN
        RETURN 0;
    END IF;
    v_count := v_count + public.fnc__mark_property_dimension_target_dirty('listing', p_sale_listing_id, p_reason);
    FOR v_target IN
        WITH linked AS (
            SELECT
                po.property_offering_id,
                pu.property_unit_id,
                pu.physical_building_id,
                COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
            FROM public.property_offering_sources pos
            JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
            JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
            LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
            WHERE pos.sale_listing_id = p_sale_listing_id
                AND pos.property_offering_source_link_status <> 'rejected'
            ORDER BY pos.property_offering_source_link_score DESC NULLS LAST, pos.property_offering_source_created_at DESC
            LIMIT 1
        )
        SELECT 'offering'::text AS target_type, property_offering_id AS target_id FROM linked WHERE property_offering_id IS NOT NULL
        UNION ALL SELECT 'unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
        UNION ALL SELECT 'building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
        UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
    LOOP
        v_count := v_count + public.fnc__mark_property_dimension_target_dirty(v_target.target_type, v_target.target_id, p_reason);
    END LOOP;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__mark_dimension_target_queued(p_target_type text, p_target_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_count integer;
BEGIN
    UPDATE public.property_dimension_dirty_targets
    SET queued_at = now()
    WHERE target_type = p_target_type
        AND target_id = p_target_id
        AND (resolved_at IS NULL OR resolved_at < dirty_at);
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__clear_property_dimension_target_dirty(p_target_type text, p_target_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_count integer;
BEGIN
    UPDATE public.property_dimension_dirty_targets
    SET resolved_at = now(),
        dirty_reasons = '{}'
    WHERE target_type = p_target_type
        AND target_id = p_target_id;
    GET DIAGNOSTICS v_count = ROW_COUNT;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__clear_listing_dimension_targets_dirty(p_sale_listing_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_count integer := 0;
    v_target record;
BEGIN
    v_count := v_count + public.fnc__clear_property_dimension_target_dirty('listing', p_sale_listing_id);
    FOR v_target IN
        WITH linked AS (
            SELECT
                po.property_offering_id,
                pu.property_unit_id,
                pu.physical_building_id,
                COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
            FROM public.property_offering_sources pos
            JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
            JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
            LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
            WHERE pos.sale_listing_id = p_sale_listing_id
                AND pos.property_offering_source_link_status <> 'rejected'
            ORDER BY pos.property_offering_source_link_score DESC NULLS LAST, pos.property_offering_source_created_at DESC
            LIMIT 1
        )
        SELECT 'offering'::text AS target_type, property_offering_id AS target_id FROM linked WHERE property_offering_id IS NOT NULL
        UNION ALL SELECT 'unit', property_unit_id FROM linked WHERE property_unit_id IS NOT NULL
        UNION ALL SELECT 'building', physical_building_id FROM linked WHERE physical_building_id IS NOT NULL
        UNION ALL SELECT 'housing_company', housing_company_id FROM linked WHERE housing_company_id IS NOT NULL
    LOOP
        v_count := v_count + public.fnc__clear_property_dimension_target_dirty(v_target.target_type, v_target.target_id);
    END LOOP;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__resolve_dimension_target(p_target_type text, p_target_id uuid)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_values integer;
    v_profiles integer;
BEGIN
    v_values := public.fnc__resolve_dimension_values_for_target(p_target_type, p_target_id);
    v_profiles := public.fnc__project_dimension_profile_for_target(p_target_type, p_target_id);
    PERFORM public.fnc__clear_property_dimension_target_dirty(p_target_type, p_target_id);
    RETURN jsonb_build_object(
        'target_type', p_target_type,
        'target_id', p_target_id,
        'values', v_values,
        'profiles', v_profiles
    );
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__rebuild_listing_dimension_layer(p_sale_listing_id uuid)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_source_claims integer;
    v_promoted_claims integer;
    v_values integer := 0;
    v_profiles integer := 0;
    v_cleaned integer := 0;
    v_target record;
BEGIN
    v_source_claims := public.fnc__project_listing_provider_dimension_claims(p_sale_listing_id);
    v_promoted_claims := public.fnc__promote_listing_dimension_claims(p_sale_listing_id);
    FOR v_target IN
        WITH linked AS (
            SELECT
                po.property_offering_id,
                pu.property_unit_id,
                pu.physical_building_id,
                COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
            FROM public.property_offering_sources pos
            JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
            JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
            LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
            WHERE pos.sale_listing_id = p_sale_listing_id
                AND pos.property_offering_source_link_status <> 'rejected'
            ORDER BY pos.property_offering_source_link_score DESC NULLS LAST, pos.property_offering_source_created_at DESC
            LIMIT 1
        ),
        target_candidates AS (
            SELECT
                catalog.target_type,
                CASE catalog.target_type
                    WHEN 'offering' THEN linked.property_offering_id
                    WHEN 'unit' THEN linked.property_unit_id
                    WHEN 'building' THEN linked.physical_building_id
                    WHEN 'housing_company' THEN linked.housing_company_id
                END AS target_id
            FROM linked
            JOIN public.property_dimension_claims c
                ON c.claim_scope = 'source'
                AND c.target_type = 'listing'
                AND c.target_id = p_sale_listing_id
            JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
            UNION
            SELECT c.target_type, c.target_id
            FROM public.property_dimension_claims c
            JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
            WHERE c.claim_scope IN ('source','manual')
                AND c.source_table = 'property_source_offerings'
                AND c.source_id = p_sale_listing_id
                AND c.target_type = catalog.target_type
        )
        SELECT DISTINCT target_type, target_id
        FROM target_candidates
        WHERE target_id IS NOT NULL
    LOOP
        v_values := v_values + public.fnc__resolve_dimension_values_for_target(v_target.target_type, v_target.target_id);
        v_profiles := v_profiles + public.fnc__project_dimension_profile_for_target(v_target.target_type, v_target.target_id);
    END LOOP;
    v_cleaned := public.fnc__clear_listing_dimension_targets_dirty(p_sale_listing_id);
    RETURN jsonb_build_object(
        'source_claims', v_source_claims,
        'promoted_claims', v_promoted_claims,
        'values', v_values,
        'profiles', v_profiles,
        'cleaned_dirty_targets', v_cleaned
    );
END;
$$;
