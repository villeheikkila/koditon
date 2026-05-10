ALTER TABLE public.property_renovation_events
    ADD COLUMN IF NOT EXISTS source_observed_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_property_renovation_events_target_observed
ON public.property_renovation_events (event_scope, target_type, target_id, category, status, source_observed_at DESC);

WITH listings AS (
    SELECT DISTINCT sale_listing_id
    FROM public.property_source_offering_renovations
),
linked AS (
    SELECT DISTINCT ON (pos.sale_listing_id)
        pos.sale_listing_id,
        COALESCE(pu.housing_company_id, pb.housing_company_id) AS housing_company_id,
        pu.physical_building_id,
        pos.sale_listing_last_seen_at,
        pos.sale_listing_updated_at,
        pos.sale_listing_created_at
    FROM public.property_source_offerings pos
    JOIN listings ON listings.sale_listing_id = pos.sale_listing_id
    LEFT JOIN public.property_offering_sources link
        ON link.sale_listing_id = pos.sale_listing_id
        AND link.property_offering_source_link_status <> 'rejected'
    LEFT JOIN public.property_offerings po ON po.property_offering_id = link.property_offering_id
    LEFT JOIN public.property_units pu ON pu.property_unit_id = po.property_unit_id
    LEFT JOIN public.physical_buildings pb ON pb.physical_building_id = pu.physical_building_id
    WHERE COALESCE(pu.housing_company_id, pb.housing_company_id, pu.physical_building_id) IS NOT NULL
    ORDER BY pos.sale_listing_id, link.property_offering_source_link_score DESC NULLS LAST, link.property_offering_source_updated_at DESC NULLS LAST
),
runs AS (
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    SELECT
        'renovation_events',
        'listing-renovation-events-v1',
        'property_source_offerings',
        linked.sale_listing_id,
        'succeeded',
        now()
    FROM linked
    RETURNING property_dimension_projection_run_id, source_id
)
INSERT INTO public.property_renovation_events (
        property_dimension_projection_run_id,
        projection_version,
        event_scope,
        target_type,
        target_id,
        source_table,
        source_id,
        source_field,
        category,
        component,
        status,
        stage,
        scope,
        responsibility,
        year,
        start_year,
        end_year,
        cost_estimate_eur,
        summary,
        evidence,
        confidence,
        source_reliability,
        source_observed_at
    )
    SELECT
        runs.property_dimension_projection_run_id,
        'listing-renovation-events-v1',
        'source',
        CASE WHEN linked.housing_company_id IS NOT NULL THEN 'housing_company' ELSE 'building' END,
        COALESCE(linked.housing_company_id, linked.physical_building_id),
        'property_source_offerings',
        linked.sale_listing_id,
        renovation.property_source_offering_renovation_source_field,
        renovation.property_source_offering_renovation_category,
        NULLIF(renovation.property_source_offering_renovation_component, ''),
        renovation.property_source_offering_renovation_status,
        NULLIF(renovation.property_source_offering_renovation_stage, ''),
        NULLIF(renovation.property_source_offering_renovation_scope, ''),
        NULLIF(renovation.property_source_offering_renovation_responsibility, ''),
        renovation.property_source_offering_renovation_year,
        NULL,
        NULL,
        renovation.property_source_offering_renovation_cost_estimate_eur,
        NULLIF(renovation.property_source_offering_renovation_text, ''),
        jsonb_build_object('evidence_level', CASE WHEN renovation.property_source_offering_renovation_source_field LIKE 'llm_%' THEN 'listing_llm' ELSE 'listing_field' END),
        GREATEST(0, LEAST(1, COALESCE(renovation.property_source_offering_renovation_confidence, 50)::double precision / 100)),
        CASE WHEN renovation.property_source_offering_renovation_source_field LIKE 'llm_%' THEN 0.75 ELSE 0.65 END,
        COALESCE(linked.sale_listing_last_seen_at, linked.sale_listing_updated_at, linked.sale_listing_created_at, now())
    FROM linked
    JOIN runs ON runs.source_id = linked.sale_listing_id
    JOIN public.property_source_offering_renovations renovation ON renovation.sale_listing_id = linked.sale_listing_id
    ON CONFLICT (
        event_scope,
        target_type,
        target_id,
        source_table,
        source_id,
        COALESCE(source_field, ''),
        category,
        status,
        COALESCE(stage, ''),
        COALESCE(scope, ''),
        COALESCE(year, -1),
        COALESCE(start_year, -1),
        COALESCE(end_year, -1),
        md5(COALESCE(summary, '')),
        projection_version
    ) DO UPDATE SET
        component = EXCLUDED.component,
        responsibility = EXCLUDED.responsibility,
        cost_estimate_eur = EXCLUDED.cost_estimate_eur,
        confidence = EXCLUDED.confidence,
        source_reliability = EXCLUDED.source_reliability,
        source_observed_at = EXCLUDED.source_observed_at,
        evidence = EXCLUDED.evidence;
