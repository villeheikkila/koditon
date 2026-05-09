CREATE OR REPLACE FUNCTION public.fnc__legacy_property_dimension_claim_scope(p_target_type text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
SELECT CASE
    WHEN p_target_type = 'manual' THEN 'manual'
    ELSE 'source'
END
$$;

DELETE FROM public.property_dimension_claims
WHERE claim_scope = 'canonical'
    AND (
        source_claim_id IS NOT NULL
        OR projection_version = 'listing-canonical-v1'
    );

DELETE FROM public.property_dimension_claims c
WHERE c.claim_scope = 'canonical'
    AND c.source_table IN ('property_source_offerings','property_documents')
    AND EXISTS (
        SELECT 1
        FROM public.property_dimension_claims s
        WHERE s.claim_scope = 'source'
            AND s.target_type = c.target_type
            AND s.target_id = c.target_id
            AND s.dimension_key = c.dimension_key
            AND s.source_table = c.source_table
            AND s.source_id = c.source_id
            AND COALESCE(s.source_field, '') = COALESCE(c.source_field, '')
            AND s.projection_version = c.projection_version
    );

UPDATE public.property_dimension_claims
SET claim_scope = 'source',
    updated_at = now()
WHERE claim_scope = 'canonical'
    AND source_table IN ('property_source_offerings','property_documents');

CREATE OR REPLACE FUNCTION public.fnc__promote_listing_dimension_claims(p_sale_listing_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
BEGIN
    DELETE FROM public.property_dimension_claims
    WHERE claim_scope = 'canonical'
        AND source_table = 'property_source_offerings'
        AND source_id = p_sale_listing_id
        AND (
            source_claim_id IS NOT NULL
            OR projection_version = 'listing-canonical-v1'
        );
    RETURN 0;
END;
$$;

CREATE OR REPLACE FUNCTION public.fnc__resolve_dimension_values_for_target(p_target_type text, p_target_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_run_id uuid;
    v_count integer;
    v_projection_version text := 'dimension-resolver-v2';
BEGIN
    DELETE FROM public.property_dimension_values
    WHERE target_type = p_target_type
        AND target_id = p_target_id;
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    VALUES (
        'resolved_values',
        v_projection_version,
        'property_dimension_claims',
        p_target_id,
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id INTO v_run_id;
    WITH linked_listings AS (
        SELECT DISTINCT
            pos.sale_listing_id,
            po.property_offering_id,
            pu.property_unit_id,
            pu.physical_building_id,
            COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id
        FROM public.property_offering_sources pos
        JOIN public.property_offerings po ON po.property_offering_id = pos.property_offering_id
        JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
        LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
        WHERE pos.property_offering_source_link_status <> 'rejected'
            AND p_target_id = CASE p_target_type
                WHEN 'offering' THEN po.property_offering_id
                WHEN 'unit' THEN pu.property_unit_id
                WHEN 'building' THEN pu.physical_building_id
                WHEN 'housing_company' THEN COALESCE(pu.housing_company_id, pb.housing_company_id)
            END
    ),
    raw_candidates AS (
        SELECT c.*
        FROM public.property_dimension_claims c
        WHERE c.claim_scope = 'manual'
            AND c.target_type = p_target_type
            AND c.target_id = p_target_id
        UNION ALL
        SELECT c.*
        FROM public.property_dimension_claims c
        JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
        WHERE c.claim_scope = 'source'
            AND c.target_type = p_target_type
            AND c.target_id = p_target_id
            AND catalog.target_type = p_target_type
        UNION ALL
        SELECT c.*
        FROM linked_listings linked
        JOIN public.property_dimension_claims c
            ON c.claim_scope = 'source'
            AND c.target_type = 'listing'
            AND c.target_id = linked.sale_listing_id
        JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
        WHERE catalog.target_type = p_target_type
        UNION ALL
        SELECT c.*
        FROM public.property_documents d
        JOIN public.property_dimension_claims c
            ON c.claim_scope = 'source'
            AND c.source_table = 'property_documents'
            AND c.source_id = d.property_document_id
        JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
        WHERE catalog.target_type = p_target_type
            AND p_target_id = CASE p_target_type
                WHEN 'offering' THEN d.property_offering_id
                WHEN 'unit' THEN d.property_unit_id
                WHEN 'building' THEN d.physical_building_id
                WHEN 'housing_company' THEN d.housing_company_id
            END
    ),
    candidates AS (
        SELECT DISTINCT ON (property_dimension_claim_id)
            property_dimension_claim_id,
            p_target_type AS target_type,
            p_target_id AS target_id,
            dimension_key,
            value,
            value_kind,
            unit,
            confidence,
            source_reliability,
            claim_scope,
            source_table,
            source_field,
            source_observed_at,
            created_at
        FROM raw_candidates
        ORDER BY property_dimension_claim_id
    ),
    scored AS (
        SELECT
            c.*,
            COALESCE(sp.priority, CASE WHEN c.claim_scope = 'manual' THEN 1000 ELSE 50 END) AS source_priority,
            p.strategy,
            p.freshness_half_life_days,
            CASE
                WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
                ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
            END AS freshness_factor,
            CASE
                WHEN c.claim_scope = 'manual' THEN 1000000::double precision
                ELSE COALESCE(sp.priority, 50)::double precision * c.source_reliability * c.confidence *
                    CASE
                        WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
                        ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
                    END
            END AS score
        FROM candidates c
        JOIN public.property_dimension_resolution_policies p ON p.dimension_key = c.dimension_key
        LEFT JOIN public.property_dimension_source_priorities sp
            ON sp.dimension_key = c.dimension_key
            AND sp.source_table = c.source_table
            AND COALESCE(sp.source_field, '') = COALESCE(c.source_field, '')
    ),
    ranked AS (
        SELECT
            *,
            row_number() OVER (
                PARTITION BY dimension_key
                ORDER BY score DESC, created_at DESC, property_dimension_claim_id
            ) AS selected_rank
        FROM scored
    ),
    stats AS (
        SELECT
            dimension_key,
            count(*) AS claim_count,
            count(DISTINCT value::text) AS distinct_value_count
        FROM scored
        GROUP BY dimension_key
    ),
    selected AS (
        SELECT
            ranked.*,
            stats.claim_count,
            stats.distinct_value_count
        FROM ranked
        JOIN stats ON stats.dimension_key = ranked.dimension_key
        WHERE ranked.selected_rank = 1
    ),
    grouped AS (
        SELECT
            s.target_type,
            s.target_id,
            s.dimension_key,
            s.value,
            s.value_kind,
            s.unit,
            s.confidence,
            s.property_dimension_claim_id AS selected_claim_id,
            CASE
                WHEN s.claim_scope = 'manual' THEN 'manual override'
                ELSE concat(s.strategy, ' score=', round(s.score::numeric, 4)::text)
            END AS selected_reason,
            CASE
                WHEN s.claim_scope = 'manual' THEN 'manual_override'
                WHEN s.distinct_value_count > 1 THEN 'conflicting'
                WHEN s.claim_count > 1 THEN 'compatible'
                ELSE 'none'
            END AS conflict_status,
            array_remove(array_agg(r.property_dimension_claim_id) FILTER (WHERE r.value::text = s.value::text), NULL) AS supporting_claim_ids,
            array_remove(array_agg(r.property_dimension_claim_id) FILTER (WHERE r.value::text <> s.value::text), NULL) AS rejected_claim_ids
        FROM selected s
        JOIN ranked r ON r.dimension_key = s.dimension_key
        GROUP BY
            s.target_type,
            s.target_id,
            s.dimension_key,
            s.value,
            s.value_kind,
            s.unit,
            s.confidence,
            s.property_dimension_claim_id,
            s.claim_scope,
            s.strategy,
            s.score,
            s.distinct_value_count,
            s.claim_count
    )
    INSERT INTO public.property_dimension_values (
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        confidence,
        selected_claim_id,
        selected_reason,
        conflict_status,
        supporting_claim_ids,
        rejected_claim_ids,
        resolved_at
    )
    SELECT
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        confidence,
        selected_claim_id,
        selected_reason,
        conflict_status,
        COALESCE(supporting_claim_ids, ARRAY[]::uuid[]),
        COALESCE(rejected_claim_ids, ARRAY[]::uuid[]),
        now()
    FROM grouped;
    GET DIAGNOSTICS v_count = ROW_COUNT;
    UPDATE public.property_dimension_projection_runs
    SET result = jsonb_build_object('value_count', v_count)
    WHERE property_dimension_projection_run_id = v_run_id;
    RETURN v_count;
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
    RETURN jsonb_build_object(
        'source_claims', v_source_claims,
        'promoted_claims', v_promoted_claims,
        'values', v_values,
        'profiles', v_profiles
    );
END;
$$;
