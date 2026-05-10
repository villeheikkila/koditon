UPDATE public.property_dimension_resolution_policies
SET strategy = CASE
        WHEN dimension_key IN ('unit.apartment_number','unit.shares','building.build_year','housing_company.name','housing_company.business_id') THEN 'stable_identity'
        WHEN dimension_key LIKE 'charges.%' THEN 'latest_reliable'
        WHEN dimension_key LIKE 'condition.%' THEN 'latest_reliable'
        WHEN dimension_key IN ('risk.financial_risk','risk.maintenance_risk','risk.repair_backlog_risk') THEN 'latest_reliable'
        WHEN dimension_key LIKE 'risk.%' OR dimension_key LIKE 'site.%' OR dimension_key IN ('building.energy_class','building.heating_method','building.material','building.roof_type','building.roof_material','building.floor_count','building.apartment_count','building.elevator') THEN 'document_preferred'
        ELSE strategy
    END,
    freshness_half_life_days = CASE
        WHEN dimension_key LIKE 'charges.%' THEN 180
        WHEN dimension_key LIKE 'condition.%' THEN 365
        WHEN dimension_key IN ('risk.financial_risk','risk.maintenance_risk','risk.repair_backlog_risk') THEN 365
        WHEN dimension_key = 'building.energy_class' THEN 730
        ELSE freshness_half_life_days
    END,
    conflict_tolerance = CASE
        WHEN dimension_key LIKE 'charges.%' THEN jsonb_build_object('newer_listing_can_override_days', 90, 'reason', 'charges can change after certificate issue date')
        WHEN dimension_key LIKE 'condition.%' THEN jsonb_build_object('newer_listing_can_override_days', 180, 'reason', 'listing condition can reflect work after certificate issue date')
        WHEN dimension_key IN ('risk.financial_risk','risk.maintenance_risk','risk.repair_backlog_risk') THEN jsonb_build_object('newer_listing_can_override_days', 180, 'reason', 'latest listing text may include newer future work or decisions')
        ELSE conflict_tolerance
    END,
    updated_at = now()
WHERE dimension_key IN (
    SELECT dimension_key
    FROM public.property_dimension_catalog
);

INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability)
VALUES
    ('unit.apartment_number','property_documents','unit.apartment_number',95,0.9),
    ('unit.shares','property_documents','unit.shares',95,0.9),
    ('unit.area_m2','property_documents','unit.area_m2',92,0.9),
    ('layout.room_layout','property_documents','unit.room_layout',88,0.86),
    ('unit.floor_level','property_documents','unit.floor_level',88,0.86),
    ('charges.maintenance_monthly_eur','property_documents','unit.maintenance_charge_monthly',90,0.9),
    ('charges.capital_monthly_eur','property_documents','unit.capital_charge_monthly',90,0.9),
    ('charges.total_monthly_eur','property_documents','unit.total_charge_monthly',90,0.9),
    ('charges.debt_share_eur','property_documents','unit.debt_share_eur',90,0.9),
    ('risk.shareholder_liability','property_documents','unit.shareholder_liability',92,0.9),
    ('housing_company.name','property_documents','housing_company.name',96,0.92),
    ('housing_company.business_id','property_documents','housing_company.business_id',98,0.95),
    ('housing_company.apartment_count','property_documents','housing_company.apartment_count',90,0.9),
    ('site.plot_ownership_type','property_documents','housing_company.plot_ownership_type',94,0.9),
    ('building.energy_class','property_documents','housing_company.energy_class',90,0.88),
    ('building.build_year','property_documents','building.build_year',94,0.9),
    ('building.floor_count','property_documents','building.floor_count',92,0.9),
    ('building.apartment_count','property_documents','building.apartment_count',92,0.9),
    ('building.energy_class','property_documents','building.energy_class',90,0.88),
    ('building.heating_method','property_documents','building.heating_method',90,0.88),
    ('building.material','property_documents','building.material',90,0.88),
    ('building.roof_type','property_documents','building.roof_type',90,0.88),
    ('building.roof_material','property_documents','building.roof_material',90,0.88),
    ('building.elevator','property_documents','building.elevator',90,0.88),
    ('risk.financial_risk','property_documents','finances.financial_risk',88,0.86),
    ('risk.maintenance_risk','property_documents','finances.maintenance_risk',88,0.86),
    ('risk.repair_backlog_risk','property_documents','finances.repair_backlog_risk',88,0.86),
    ('risk.administrative_legal_risk','property_documents','risks.administrative_legal_risk',92,0.9),
    ('risk.restrictions','property_documents','risks.restrictions',92,0.9)
ON CONFLICT (dimension_key, source_table, COALESCE(source_field, '')) DO UPDATE SET
    priority = EXCLUDED.priority,
    default_reliability = EXCLUDED.default_reliability;

CREATE OR REPLACE FUNCTION public.fnc__resolve_dimension_values_for_target(p_target_type text, p_target_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_run_id uuid;
    v_count integer;
    v_projection_version text := 'dimension-resolver-v3';
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
            COALESCE(sp.default_reliability, c.source_reliability) AS effective_reliability,
            p.strategy,
            p.freshness_half_life_days,
            CASE
                WHEN c.claim_scope = 'manual' THEN 1::double precision
                WHEN p.strategy IN ('stable_identity','document_preferred') AND p.freshness_half_life_days IS NULL THEN 1::double precision
                WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
                ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
            END AS freshness_factor,
            CASE
                WHEN c.claim_scope = 'manual' THEN 1::double precision
                WHEN p.strategy = 'document_preferred' AND c.source_table = 'property_documents' THEN 1.45::double precision
                WHEN p.strategy = 'stable_identity' AND c.source_table = 'property_documents' THEN 1.2::double precision
                WHEN p.strategy = 'latest_reliable' AND c.source_table = 'property_source_offerings' THEN 1.05::double precision
                ELSE 1::double precision
            END AS authority_factor,
            CASE
                WHEN c.claim_scope = 'manual' THEN 1000000::double precision
                ELSE COALESCE(sp.priority, 50)::double precision *
                    COALESCE(sp.default_reliability, c.source_reliability) *
                    c.confidence *
                    CASE
                        WHEN p.strategy IN ('stable_identity','document_preferred') AND p.freshness_half_life_days IS NULL THEN 1::double precision
                        WHEN p.freshness_half_life_days IS NULL OR c.source_observed_at IS NULL THEN 1::double precision
                        ELSE power(0.5::double precision, GREATEST(0::double precision, EXTRACT(EPOCH FROM (now() - c.source_observed_at)) / 86400::double precision) / p.freshness_half_life_days::double precision)
                    END *
                    CASE
                        WHEN p.strategy = 'document_preferred' AND c.source_table = 'property_documents' THEN 1.45::double precision
                        WHEN p.strategy = 'stable_identity' AND c.source_table = 'property_documents' THEN 1.2::double precision
                        WHEN p.strategy = 'latest_reliable' AND c.source_table = 'property_source_offerings' THEN 1.05::double precision
                        ELSE 1::double precision
                    END
            END AS score
        FROM candidates c
        JOIN public.property_dimension_resolution_policies p ON p.dimension_key = c.dimension_key
        LEFT JOIN LATERAL (
            SELECT priority, default_reliability
            FROM public.property_dimension_source_priorities candidate_priority
            WHERE candidate_priority.dimension_key = c.dimension_key
                AND candidate_priority.source_table = c.source_table
                AND (
                    candidate_priority.source_field IS NULL
                    OR COALESCE(candidate_priority.source_field, '') = COALESCE(c.source_field, '')
                )
            ORDER BY CASE WHEN COALESCE(candidate_priority.source_field, '') = COALESCE(c.source_field, '') THEN 0 ELSE 1 END
            LIMIT 1
        ) sp ON true
    ),
    ranked AS (
        SELECT
            *,
            row_number() OVER (
                PARTITION BY dimension_key
                ORDER BY score DESC, source_observed_at DESC NULLS LAST, created_at DESC, property_dimension_claim_id
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
                ELSE concat(s.strategy, ' score=', round(s.score::numeric, 4)::text, ' priority=', s.source_priority::text, ' reliability=', round(s.effective_reliability::numeric, 3)::text, ' freshness=', round(s.freshness_factor::numeric, 3)::text, ' authority=', round(s.authority_factor::numeric, 3)::text)
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
            s.source_priority,
            s.effective_reliability,
            s.freshness_factor,
            s.authority_factor,
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
    SET result = jsonb_build_object('resolved_values', v_count),
        finished_at = now()
    WHERE property_dimension_projection_run_id = v_run_id;
    RETURN v_count;
END;
$$;
