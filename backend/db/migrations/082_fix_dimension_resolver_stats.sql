CREATE OR REPLACE FUNCTION public.fnc__resolve_dimension_values_for_target(p_target_type text, p_target_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_run_id uuid;
    v_count integer;
    v_projection_version text := 'dimension-resolver-v1';
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
    WITH candidates AS (
        SELECT
            c.property_dimension_claim_id,
            c.target_type,
            c.target_id,
            c.dimension_key,
            c.value,
            c.value_kind,
            c.unit,
            c.confidence,
            c.source_reliability,
            c.claim_scope,
            c.created_at,
            COALESCE(sp.priority, CASE WHEN c.claim_scope = 'manual' THEN 1000 ELSE 50 END) AS source_priority,
            p.strategy,
            p.freshness_half_life_days,
            CASE
                WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
                ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
            END AS freshness_factor
        FROM public.property_dimension_claims c
        JOIN public.property_dimension_catalog catalog ON catalog.dimension_key = c.dimension_key
        JOIN public.property_dimension_resolution_policies p ON p.dimension_key = c.dimension_key
        LEFT JOIN public.property_dimension_source_priorities sp
            ON sp.dimension_key = c.dimension_key
            AND sp.source_table = c.source_table
            AND COALESCE(sp.source_field, '') = COALESCE(c.source_field, '')
        WHERE c.target_type = p_target_type
            AND c.target_id = p_target_id
            AND c.claim_scope IN ('canonical','manual')
    ),
    scored AS (
        SELECT
            *,
            CASE
                WHEN claim_scope = 'manual' THEN 1000000::double precision
                ELSE source_priority::double precision * source_reliability * confidence * freshness_factor
            END AS score
        FROM candidates
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
