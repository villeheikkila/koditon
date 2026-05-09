CREATE TABLE IF NOT EXISTS public.property_dimension_projection_runs (
    property_dimension_projection_run_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    projection_type text NOT NULL,
    projection_version text NOT NULL,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    status text NOT NULL,
    result jsonb NOT NULL DEFAULT '{}'::jsonb,
    error_text text,
    started_at timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    CHECK (projection_type = ANY (ARRAY['source_claims','canonical_claims','renovation_events','resolved_values','profiles','system_profiles']::text[])),
    CHECK (status = ANY (ARRAY['running','succeeded','failed']::text[]))
);
CREATE INDEX IF NOT EXISTS idx_property_dimension_projection_runs_source
ON public.property_dimension_projection_runs (projection_type, source_table, source_id, projection_version, started_at DESC);
CREATE TABLE IF NOT EXISTS public.property_dimension_catalog (
    dimension_key text PRIMARY KEY,
    target_type text NOT NULL,
    value_kind text NOT NULL,
    unit text,
    profile_section text NOT NULL,
    profile_key text NOT NULL,
    promoted_to_valuation boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[]))
);
CREATE TABLE IF NOT EXISTS public.property_dimension_resolution_policies (
    dimension_key text PRIMARY KEY REFERENCES public.property_dimension_catalog(dimension_key),
    strategy text NOT NULL,
    freshness_half_life_days integer,
    conflict_tolerance jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (strategy = ANY (ARRAY['manual_override','latest_reliable','highest_reliability','document_preferred','stable_identity','numeric_consensus']::text[]))
);
CREATE TABLE IF NOT EXISTS public.property_dimension_source_priorities (
    dimension_key text NOT NULL REFERENCES public.property_dimension_catalog(dimension_key),
    source_table text NOT NULL,
    source_field text,
    priority integer NOT NULL,
    default_reliability double precision NOT NULL,
    CHECK (default_reliability >= 0 AND default_reliability <= 1)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_dimension_source_priorities_unique
ON public.property_dimension_source_priorities (dimension_key, source_table, COALESCE(source_field, ''));
CREATE TABLE IF NOT EXISTS public.property_dimension_claims (
    property_dimension_claim_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_dimension_projection_run_id uuid NOT NULL REFERENCES public.property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE,
    projection_version text NOT NULL,
    claim_scope text NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimension_key text NOT NULL,
    value jsonb NOT NULL,
    value_kind text NOT NULL,
    unit text,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    source_field text,
    source_claim_id uuid REFERENCES public.property_dimension_claims(property_dimension_claim_id) ON DELETE CASCADE,
    source_observed_at timestamptz,
    valid_from date,
    valid_until date,
    confidence double precision NOT NULL DEFAULT 0.5,
    source_reliability double precision NOT NULL DEFAULT 0.5,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    extraction_model text,
    extraction_prompt_version text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (claim_scope = ANY (ARRAY['source','canonical','manual']::text[])),
    CHECK (target_type = ANY (ARRAY['listing','document','transaction','offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1),
    CHECK (source_reliability >= 0 AND source_reliability <= 1)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_dimension_claims_unique_source
ON public.property_dimension_claims (claim_scope, target_type, target_id, dimension_key, source_table, source_id, COALESCE(source_field, ''), projection_version);
CREATE INDEX IF NOT EXISTS idx_property_dimension_claims_target
ON public.property_dimension_claims (claim_scope, target_type, target_id, dimension_key);
CREATE INDEX IF NOT EXISTS idx_property_dimension_claims_source
ON public.property_dimension_claims (source_table, source_id, projection_version);
CREATE INDEX IF NOT EXISTS idx_property_dimension_claims_source_claim
ON public.property_dimension_claims (source_claim_id);
CREATE INDEX IF NOT EXISTS idx_property_dimension_claims_dimension
ON public.property_dimension_claims (dimension_key);
CREATE INDEX IF NOT EXISTS idx_property_dimension_claims_value_gin
ON public.property_dimension_claims USING gin (value jsonb_path_ops);
CREATE TABLE IF NOT EXISTS public.property_dimension_manual_overrides (
    property_dimension_manual_override_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimension_key text NOT NULL,
    value jsonb NOT NULL,
    value_kind text NOT NULL,
    unit text,
    reason text NOT NULL,
    created_by text NOT NULL,
    valid_from date,
    valid_until date,
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[]))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_dimension_manual_overrides_active
ON public.property_dimension_manual_overrides (target_type, target_id, dimension_key) WHERE revoked_at IS NULL;
CREATE TABLE IF NOT EXISTS public.property_dimension_values (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimension_key text NOT NULL,
    value jsonb NOT NULL,
    value_kind text NOT NULL,
    unit text,
    confidence double precision NOT NULL,
    selected_claim_id uuid REFERENCES public.property_dimension_claims(property_dimension_claim_id) ON DELETE CASCADE,
    selected_reason text NOT NULL,
    conflict_status text NOT NULL DEFAULT 'none',
    supporting_claim_ids uuid[] NOT NULL DEFAULT '{}',
    rejected_claim_ids uuid[] NOT NULL DEFAULT '{}',
    resolved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id, dimension_key),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[])),
    CHECK (value_kind = ANY (ARRAY['string','number','boolean','object','array','null']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1),
    CHECK (conflict_status = ANY (ARRAY['none','compatible','conflicting','manual_override']::text[]))
);
CREATE INDEX IF NOT EXISTS idx_property_dimension_values_dimension
ON public.property_dimension_values (dimension_key);
CREATE INDEX IF NOT EXISTS idx_property_dimension_values_selected_claim
ON public.property_dimension_values (selected_claim_id);
CREATE TABLE IF NOT EXISTS public.property_dimension_profiles (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    dimensions jsonb NOT NULL DEFAULT '{}'::jsonb,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    conflicts jsonb NOT NULL DEFAULT '{}'::jsonb,
    resolved_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id),
    CHECK (target_type = ANY (ARRAY['offering','unit','building','housing_company']::text[]))
);
CREATE INDEX IF NOT EXISTS idx_property_dimension_profiles_dimensions_gin
ON public.property_dimension_profiles USING gin (dimensions jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_unit_dimension_profiles_area
ON public.property_dimension_profiles (((dimensions #>> '{unit,area_m2}')::double precision)) WHERE target_type = 'unit';
CREATE INDEX IF NOT EXISTS idx_unit_dimension_profiles_total_charge
ON public.property_dimension_profiles (((dimensions #>> '{charges,total_monthly_eur}')::double precision)) WHERE target_type = 'unit';
CREATE INDEX IF NOT EXISTS idx_building_dimension_profiles_build_year
ON public.property_dimension_profiles (((dimensions #>> '{building,build_year}')::integer)) WHERE target_type = 'building';
CREATE TABLE IF NOT EXISTS public.property_renovation_events (
    property_renovation_event_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    property_dimension_projection_run_id uuid NOT NULL REFERENCES public.property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE,
    projection_version text NOT NULL,
    event_scope text NOT NULL,
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_table text NOT NULL,
    source_id uuid NOT NULL,
    source_field text,
    source_event_id uuid REFERENCES public.property_renovation_events(property_renovation_event_id),
    category text NOT NULL,
    component text,
    status text NOT NULL,
    stage text,
    scope text,
    responsibility text,
    year integer,
    start_year integer,
    end_year integer,
    cost_estimate_eur bigint,
    summary text,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    confidence double precision NOT NULL DEFAULT 0.5,
    source_reliability double precision NOT NULL DEFAULT 0.5,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (event_scope = ANY (ARRAY['source','canonical','manual']::text[])),
    CHECK (target_type = ANY (ARRAY['listing','document','offering','unit','building','housing_company']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1),
    CHECK (source_reliability >= 0 AND source_reliability <= 1)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_renovation_events_unique_source
ON public.property_renovation_events (event_scope, target_type, target_id, source_table, source_id, COALESCE(source_field, ''), category, status, COALESCE(stage, ''), COALESCE(scope, ''), COALESCE(year, -1), COALESCE(start_year, -1), COALESCE(end_year, -1), md5(COALESCE(summary, '')), projection_version);
CREATE INDEX IF NOT EXISTS idx_property_renovation_events_target
ON public.property_renovation_events (event_scope, target_type, target_id, category, status);
CREATE INDEX IF NOT EXISTS idx_property_renovation_events_source
ON public.property_renovation_events (source_table, source_id, projection_version);
CREATE INDEX IF NOT EXISTS idx_property_renovation_events_source_event
ON public.property_renovation_events (source_event_id);
CREATE TABLE IF NOT EXISTS public.property_system_profiles (
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    system_type text NOT NULL,
    status text NOT NULL,
    last_renovated_year integer,
    next_expected_start_year integer,
    next_expected_end_year integer,
    stage text,
    scope text,
    responsibility text,
    cost_estimate_eur bigint,
    confidence double precision NOT NULL DEFAULT 0.5,
    selected_renovation_event_ids uuid[] NOT NULL DEFAULT '{}',
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (target_type, target_id, system_type),
    CHECK (target_type = ANY (ARRAY['unit','building','housing_company']::text[])),
    CHECK (confidence >= 0 AND confidence <= 1)
);
CREATE INDEX IF NOT EXISTS idx_property_system_profiles_target
ON public.property_system_profiles (target_type, target_id);
INSERT INTO public.property_dimension_catalog (dimension_key, target_type, value_kind, unit, profile_section, profile_key, promoted_to_valuation)
VALUES
    ('unit.area_m2','unit','number','m2','unit','area_m2',true),
    ('unit.living_area_m2','unit','number','m2','unit','living_area_m2',true),
    ('unit.total_area_m2','unit','number','m2','unit','total_area_m2',true),
    ('unit.other_area_m2','unit','number','m2','unit','other_area_m2',true),
    ('unit.floor_level','unit','number',NULL,'unit','floor_level',true),
    ('unit.total_floors','unit','number',NULL,'unit','total_floors',true),
    ('unit.apartment_number','unit','string',NULL,'unit','apartment_number',false),
    ('unit.shares','unit','string',NULL,'unit','shares',false),
    ('layout.room_layout','unit','string',NULL,'layout','room_layout',true),
    ('layout.room_count','unit','number',NULL,'layout','room_count',true),
    ('layout.bedroom_count','unit','number',NULL,'layout','bedroom_count',true),
    ('layout.kitchen_type','unit','string',NULL,'layout','kitchen_type',true),
    ('layout.separate_wc_count','unit','number',NULL,'layout','separate_wc_count',true),
    ('layout.quality','unit','string',NULL,'layout','quality',true),
    ('layout.awkward','unit','boolean',NULL,'layout','awkward',true),
    ('condition.unit_condition','unit','string',NULL,'condition','unit_condition',true),
    ('condition.kitchen_condition','unit','string',NULL,'condition','kitchen_condition',true),
    ('condition.bathroom_condition','unit','string',NULL,'condition','bathroom_condition',true),
    ('condition.surface_renovation_need','unit','boolean',NULL,'condition','surface_renovation_need',true),
    ('condition.modernization_need','unit','boolean',NULL,'condition','modernization_need',true),
    ('features.sauna','unit','boolean',NULL,'features','sauna',true),
    ('features.private_sauna','unit','boolean',NULL,'features','private_sauna',true),
    ('features.balcony','unit','boolean',NULL,'features','balcony',true),
    ('features.balcony_glazing','unit','boolean',NULL,'features','balcony_glazing',true),
    ('features.parking_type','unit','string',NULL,'features','parking_type',true),
    ('features.storage_quality','unit','string',NULL,'features','storage_quality',true),
    ('features.view_quality','unit','string',NULL,'features','view_quality',true),
    ('features.noise_risk','unit','boolean',NULL,'features','noise_risk',true),
    ('features.accessibility','unit','string',NULL,'features','accessibility',true),
    ('charges.maintenance_monthly_eur','unit','number','eur/month','charges','maintenance_monthly_eur',true),
    ('charges.capital_monthly_eur','unit','number','eur/month','charges','capital_monthly_eur',true),
    ('charges.total_monthly_eur','unit','number','eur/month','charges','total_monthly_eur',true),
    ('charges.water_monthly_eur','unit','number','eur/month','charges','water_monthly_eur',true),
    ('charges.parking_monthly_eur','unit','number','eur/month','charges','parking_monthly_eur',true),
    ('charges.debt_share_eur','unit','number','eur','charges','debt_share_eur',true),
    ('charges.charge_risk','unit','string',NULL,'charges','charge_risk',true),
    ('risk.shareholder_liability','unit','string',NULL,'risk','shareholder_liability',true),
    ('building.build_year','building','number',NULL,'building','build_year',true),
    ('building.floor_count','building','number',NULL,'building','floor_count',true),
    ('building.apartment_count','building','number',NULL,'building','apartment_count',true),
    ('building.elevator','building','boolean',NULL,'building','elevator',true),
    ('building.energy_class','building','string',NULL,'building','energy_class',true),
    ('building.heating_method','building','string',NULL,'building','heating_method',true),
    ('building.material','building','string',NULL,'building','material',true),
    ('building.roof_type','building','string',NULL,'building','roof_type',true),
    ('building.roof_material','building','string',NULL,'building','roof_material',true),
    ('building.common_area_quality','building','string',NULL,'building','common_area_quality',true),
    ('building.accessibility','building','string',NULL,'building','accessibility',true),
    ('housing_company.name','housing_company','string',NULL,'housing_company','name',false),
    ('housing_company.business_id','housing_company','string',NULL,'housing_company','business_id',false),
    ('housing_company.apartment_count','housing_company','number',NULL,'housing_company','apartment_count',true),
    ('housing_company.building_count','housing_company','number',NULL,'housing_company','building_count',false),
    ('site.plot_ownership_type','housing_company','string',NULL,'site','plot_ownership_type',true),
    ('site.plot_lease_end_year','housing_company','number',NULL,'site','plot_lease_end_year',false),
    ('site.plot_redemption_possible','housing_company','boolean',NULL,'site','plot_redemption_possible',false),
    ('risk.financial_risk','housing_company','string',NULL,'risk','financial_risk',true),
    ('risk.maintenance_risk','housing_company','string',NULL,'risk','maintenance_risk',true),
    ('risk.repair_backlog_risk','housing_company','string',NULL,'risk','repair_backlog_risk',true),
    ('risk.administrative_legal_risk','housing_company','string',NULL,'risk','administrative_legal_risk',false),
    ('risk.restrictions','housing_company','array',NULL,'risk','restrictions',false)
ON CONFLICT (dimension_key) DO UPDATE SET
    target_type = EXCLUDED.target_type,
    value_kind = EXCLUDED.value_kind,
    unit = EXCLUDED.unit,
    profile_section = EXCLUDED.profile_section,
    profile_key = EXCLUDED.profile_key,
    promoted_to_valuation = EXCLUDED.promoted_to_valuation,
    updated_at = now();
INSERT INTO public.property_dimension_resolution_policies (dimension_key, strategy, freshness_half_life_days, conflict_tolerance)
SELECT
    dimension_key,
    CASE
        WHEN dimension_key IN ('unit.area_m2','unit.living_area_m2','unit.total_area_m2','unit.other_area_m2','unit.floor_level','unit.total_floors','layout.room_count','layout.bedroom_count','layout.separate_wc_count','building.floor_count','building.apartment_count','housing_company.apartment_count','housing_company.building_count') THEN 'numeric_consensus'
        WHEN dimension_key IN ('unit.apartment_number','unit.shares','building.build_year','housing_company.name','housing_company.business_id') THEN 'stable_identity'
        WHEN dimension_key LIKE 'risk.%' OR dimension_key LIKE 'site.%' OR dimension_key IN ('building.energy_class','building.heating_method','building.material','building.roof_type','building.roof_material') THEN 'document_preferred'
        WHEN dimension_key = 'building.elevator' THEN 'highest_reliability'
        ELSE 'latest_reliable'
    END,
    CASE
        WHEN dimension_key LIKE 'charges.%' THEN 365
        WHEN dimension_key LIKE 'condition.%' THEN 730
        ELSE NULL
    END,
    '{}'::jsonb
FROM public.property_dimension_catalog
ON CONFLICT (dimension_key) DO UPDATE SET
    strategy = EXCLUDED.strategy,
    freshness_half_life_days = EXCLUDED.freshness_half_life_days,
    conflict_tolerance = EXCLUDED.conflict_tolerance,
    updated_at = now();
INSERT INTO public.property_dimension_source_priorities (dimension_key, source_table, source_field, priority, default_reliability)
VALUES
    ('unit.area_m2','property_source_offerings','sale_listing_area_value',70,0.75),
    ('unit.living_area_m2','property_source_offerings','sale_listing_living_area_value',70,0.75),
    ('unit.total_area_m2','property_source_offerings','sale_listing_total_area_value',70,0.75),
    ('unit.other_area_m2','property_source_offerings','sale_listing_other_area_value',70,0.75),
    ('layout.room_layout','property_source_offerings','sale_listing_room_layout',70,0.7),
    ('layout.room_count','property_source_offerings','sale_listing_rooms_count',70,0.75),
    ('layout.bedroom_count','property_source_offerings','sale_listing_bedrooms_count',70,0.7),
    ('unit.floor_level','property_source_offerings','sale_listing_floor_level',70,0.7),
    ('building.floor_count','property_source_offerings','sale_listing_total_floors',65,0.65),
    ('condition.unit_condition','property_source_offerings','sale_listing_condition',60,0.6),
    ('features.sauna','property_source_offerings','sale_listing_sauna',70,0.75),
    ('features.balcony','property_source_offerings','sale_listing_balcony',70,0.75),
    ('features.parking_type','property_source_offerings','sale_listing_parking_text',55,0.55),
    ('charges.maintenance_monthly_eur','property_source_offerings','sale_listing_maintenance_charge_monthly',70,0.7),
    ('charges.total_monthly_eur','property_source_offerings','sale_listing_total_charge_monthly',70,0.7),
    ('charges.water_monthly_eur','property_source_offerings','sale_listing_water_charge',60,0.6),
    ('charges.debt_share_eur','property_source_offerings','sale_listing_debt_share_amount',70,0.7),
    ('building.build_year','property_source_offerings','sale_listing_build_year',65,0.65),
    ('building.elevator','property_source_offerings','sale_listing_elevator',65,0.65),
    ('building.heating_method','property_source_offerings','sale_listing_heating_system',60,0.6),
    ('building.energy_class','property_source_offerings','sale_listing_energy_efficiency_label',60,0.6),
    ('building.material','property_source_offerings','sale_listing_building_material',60,0.6),
    ('building.roof_type','property_source_offerings','sale_listing_roof_type',60,0.6),
    ('building.roof_material','property_source_offerings','sale_listing_roof_material',60,0.6),
    ('housing_company.apartment_count','property_source_offerings','sale_listing_apartment_count',60,0.6),
    ('housing_company.name','property_source_offerings','sale_listing_housing_company_name',60,0.6),
    ('housing_company.business_id','property_source_offerings','sale_listing_housing_company_business_id',65,0.65),
    ('site.plot_ownership_type','property_source_offerings','sale_listing_plot_type_code',60,0.6)
ON CONFLICT (dimension_key, source_table, COALESCE(source_field, '')) DO UPDATE SET
    priority = EXCLUDED.priority,
    default_reliability = EXCLUDED.default_reliability;
CREATE OR REPLACE FUNCTION public.fnc__project_listing_provider_dimension_claims(p_sale_listing_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_run_id uuid;
    v_count integer;
    v_projection_version text := 'listing-provider-v1';
BEGIN
    DELETE FROM public.property_dimension_claims
    WHERE claim_scope = 'source'
        AND source_table = 'property_source_offerings'
        AND source_id = p_sale_listing_id
        AND projection_version = v_projection_version;
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    VALUES (
        'source_claims',
        v_projection_version,
        'property_source_offerings',
        p_sale_listing_id,
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id INTO v_run_id;
    INSERT INTO public.property_dimension_claims (
        property_dimension_projection_run_id,
        projection_version,
        claim_scope,
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        source_table,
        source_id,
        source_field,
        source_observed_at,
        confidence,
        source_reliability,
        evidence
    )
    SELECT
        v_run_id,
        v_projection_version,
        'source',
        'listing',
        sl.sale_listing_id,
        v.dimension_key,
        v.value,
        v.value_kind,
        c.unit,
        'property_source_offerings',
        sl.sale_listing_id,
        v.source_field,
        COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_updated_at, sl.sale_listing_created_at, now()),
        v.confidence,
        COALESCE(sp.default_reliability, v.source_reliability),
        jsonb_build_object('provider', sl.sale_listing_source_provider, 'source_kind', sl.sale_listing_source_kind)
    FROM public.property_source_offerings sl
    CROSS JOIN LATERAL (
        VALUES
            ('sale_listing_area_value','unit.area_m2','number',to_jsonb(sl.sale_listing_area_value),0.95,0.75),
            ('sale_listing_living_area_value','unit.living_area_m2','number',to_jsonb(sl.sale_listing_living_area_value),0.95,0.75),
            ('sale_listing_total_area_value','unit.total_area_m2','number',to_jsonb(sl.sale_listing_total_area_value),0.9,0.75),
            ('sale_listing_other_area_value','unit.other_area_m2','number',to_jsonb(sl.sale_listing_other_area_value),0.9,0.75),
            ('sale_listing_room_layout','layout.room_layout','string',to_jsonb(NULLIF(sl.sale_listing_room_layout, '')),0.9,0.7),
            ('sale_listing_rooms_count','layout.room_count','number',to_jsonb(sl.sale_listing_rooms_count),0.95,0.75),
            ('sale_listing_bedrooms_count','layout.bedroom_count','number',to_jsonb(sl.sale_listing_bedrooms_count),0.85,0.7),
            ('sale_listing_floor_level','unit.floor_level','number',to_jsonb(sl.sale_listing_floor_level),0.9,0.7),
            ('sale_listing_total_floors','building.floor_count','number',to_jsonb(sl.sale_listing_total_floors),0.85,0.65),
            ('sale_listing_condition','condition.unit_condition','string',to_jsonb(NULLIF(sl.sale_listing_condition, '')),0.75,0.6),
            ('sale_listing_sauna','features.sauna','boolean',to_jsonb(sl.sale_listing_sauna),0.9,0.75),
            ('sale_listing_balcony','features.balcony','boolean',to_jsonb(sl.sale_listing_balcony),0.9,0.75),
            ('sale_listing_parking_text','features.parking_type','string',to_jsonb(NULLIF(sl.sale_listing_parking_text, '')),0.65,0.55),
            ('sale_listing_maintenance_charge_monthly','charges.maintenance_monthly_eur','number',to_jsonb(sl.sale_listing_maintenance_charge_monthly),0.9,0.7),
            ('sale_listing_total_charge_monthly','charges.total_monthly_eur','number',to_jsonb(sl.sale_listing_total_charge_monthly),0.9,0.7),
            ('sale_listing_water_charge','charges.water_monthly_eur','number',to_jsonb(sl.sale_listing_water_charge),0.8,0.6),
            ('sale_listing_debt_share_amount','charges.debt_share_eur','number',to_jsonb(sl.sale_listing_debt_share_amount),0.9,0.7),
            ('sale_listing_build_year','building.build_year','number',to_jsonb(sl.sale_listing_build_year),0.85,0.65),
            ('sale_listing_elevator','building.elevator','boolean',to_jsonb(sl.sale_listing_elevator),0.8,0.65),
            ('sale_listing_heating_system','building.heating_method','string',to_jsonb(NULLIF(sl.sale_listing_heating_system, '')),0.75,0.6),
            ('sale_listing_energy_efficiency_label','building.energy_class','string',to_jsonb(NULLIF(sl.sale_listing_energy_efficiency_label, '')),0.75,0.6),
            ('sale_listing_building_material','building.material','string',to_jsonb(NULLIF(sl.sale_listing_building_material, '')),0.75,0.6),
            ('sale_listing_roof_type','building.roof_type','string',to_jsonb(NULLIF(sl.sale_listing_roof_type, '')),0.75,0.6),
            ('sale_listing_roof_material','building.roof_material','string',to_jsonb(NULLIF(sl.sale_listing_roof_material, '')),0.75,0.6),
            ('sale_listing_apartment_count','housing_company.apartment_count','number',to_jsonb(sl.sale_listing_apartment_count),0.75,0.6),
            ('sale_listing_housing_company_name','housing_company.name','string',to_jsonb(NULLIF(sl.sale_listing_housing_company_name, '')),0.8,0.6),
            ('sale_listing_housing_company_business_id','housing_company.business_id','string',to_jsonb(NULLIF(sl.sale_listing_housing_company_business_id, '')),0.9,0.65),
            ('sale_listing_plot_type_code','site.plot_ownership_type','string',to_jsonb(NULLIF(sl.sale_listing_plot_type_code, '')),0.75,0.6)
    ) AS v(source_field, dimension_key, value_kind, value, confidence, source_reliability)
    JOIN public.property_dimension_catalog c ON c.dimension_key = v.dimension_key
    LEFT JOIN public.property_dimension_source_priorities sp
        ON sp.dimension_key = v.dimension_key
        AND sp.source_table = 'property_source_offerings'
        AND sp.source_field = v.source_field
    WHERE sl.sale_listing_id = p_sale_listing_id
        AND v.value IS NOT NULL;
    GET DIAGNOSTICS v_count = ROW_COUNT;
    UPDATE public.property_dimension_projection_runs
    SET result = jsonb_build_object('claim_count', v_count)
    WHERE property_dimension_projection_run_id = v_run_id;
    RETURN v_count;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__promote_listing_dimension_claims(p_sale_listing_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_run_id uuid;
    v_count integer;
    v_projection_version text := 'listing-canonical-v1';
BEGIN
    DELETE FROM public.property_dimension_claims
    WHERE claim_scope = 'canonical'
        AND source_table = 'property_source_offerings'
        AND source_id = p_sale_listing_id
        AND projection_version = v_projection_version;
    INSERT INTO public.property_dimension_projection_runs (
        projection_type,
        projection_version,
        source_table,
        source_id,
        status,
        finished_at
    )
    VALUES (
        'canonical_claims',
        v_projection_version,
        'property_source_offerings',
        p_sale_listing_id,
        'succeeded',
        now()
    )
    RETURNING property_dimension_projection_run_id INTO v_run_id;
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
    candidates AS (
        SELECT
            sc.property_dimension_claim_id AS source_claim_id,
            c.target_type,
            CASE c.target_type
                WHEN 'offering' THEN linked.property_offering_id
                WHEN 'unit' THEN linked.property_unit_id
                WHEN 'building' THEN linked.physical_building_id
                WHEN 'housing_company' THEN linked.housing_company_id
            END AS target_id,
            sc.dimension_key,
            sc.value,
            sc.value_kind,
            sc.unit,
            sc.source_field,
            sc.source_observed_at,
            sc.confidence,
            sc.source_reliability,
            sc.evidence
        FROM linked
        JOIN public.property_dimension_claims sc
            ON sc.claim_scope = 'source'
            AND sc.target_type = 'listing'
            AND sc.target_id = p_sale_listing_id
        JOIN public.property_dimension_catalog c ON c.dimension_key = sc.dimension_key
    )
    INSERT INTO public.property_dimension_claims (
        property_dimension_projection_run_id,
        projection_version,
        claim_scope,
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        source_table,
        source_id,
        source_field,
        source_claim_id,
        source_observed_at,
        confidence,
        source_reliability,
        evidence
    )
    SELECT
        v_run_id,
        v_projection_version,
        'canonical',
        target_type,
        target_id,
        dimension_key,
        value,
        value_kind,
        unit,
        'property_source_offerings',
        p_sale_listing_id,
        source_field,
        source_claim_id,
        source_observed_at,
        confidence,
        source_reliability,
        evidence || jsonb_build_object('source_claim_id', source_claim_id)
    FROM candidates
    WHERE target_id IS NOT NULL;
    GET DIAGNOSTICS v_count = ROW_COUNT;
    UPDATE public.property_dimension_projection_runs
    SET result = jsonb_build_object('claim_count', v_count)
    WHERE property_dimension_projection_run_id = v_run_id;
    RETURN v_count;
END;
$$;
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
CREATE OR REPLACE FUNCTION public.fnc__project_dimension_profile_for_target(p_target_type text, p_target_id uuid)
RETURNS integer
LANGUAGE plpgsql
AS $$
DECLARE
    v_dimensions jsonb;
    v_conflicts jsonb;
BEGIN
    WITH sections AS (
        SELECT
            c.profile_section,
            jsonb_object_agg(c.profile_key, v.value ORDER BY c.profile_key) AS section_json
        FROM public.property_dimension_values v
        JOIN public.property_dimension_catalog c ON c.dimension_key = v.dimension_key
        WHERE v.target_type = p_target_type
            AND v.target_id = p_target_id
        GROUP BY c.profile_section
    )
    SELECT COALESCE(jsonb_object_agg(profile_section, section_json ORDER BY profile_section), '{}'::jsonb)
    INTO v_dimensions
    FROM sections;
    SELECT COALESCE(jsonb_object_agg(dimension_key, conflict_status ORDER BY dimension_key), '{}'::jsonb)
    INTO v_conflicts
    FROM public.property_dimension_values
    WHERE target_type = p_target_type
        AND target_id = p_target_id
        AND conflict_status <> 'none';
    INSERT INTO public.property_dimension_profiles (
        target_type,
        target_id,
        dimensions,
        metadata,
        conflicts,
        resolved_at
    )
    VALUES (
        p_target_type,
        p_target_id,
        v_dimensions,
        jsonb_build_object('projection_version', 'dimension-profile-v1'),
        v_conflicts,
        now()
    )
    ON CONFLICT (target_type, target_id) DO UPDATE SET
        dimensions = EXCLUDED.dimensions,
        metadata = EXCLUDED.metadata,
        conflicts = EXCLUDED.conflicts,
        resolved_at = EXCLUDED.resolved_at;
    RETURN 1;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__rebuild_listing_dimension_layer(p_sale_listing_id uuid)
RETURNS jsonb
LANGUAGE plpgsql
AS $$
DECLARE
    v_source_claims integer;
    v_canonical_claims integer;
    v_values integer := 0;
    v_profiles integer := 0;
    v_target record;
BEGIN
    v_source_claims := public.fnc__project_listing_provider_dimension_claims(p_sale_listing_id);
    v_canonical_claims := public.fnc__promote_listing_dimension_claims(p_sale_listing_id);
    FOR v_target IN
        SELECT DISTINCT target_type, target_id
        FROM public.property_dimension_claims
        WHERE claim_scope IN ('canonical','manual')
            AND source_table = 'property_source_offerings'
            AND source_id = p_sale_listing_id
            AND projection_version = 'listing-canonical-v1'
    LOOP
        v_values := v_values + public.fnc__resolve_dimension_values_for_target(v_target.target_type, v_target.target_id);
        v_profiles := v_profiles + public.fnc__project_dimension_profile_for_target(v_target.target_type, v_target.target_id);
    END LOOP;
    RETURN jsonb_build_object(
        'source_claims', v_source_claims,
        'canonical_claims', v_canonical_claims,
        'values', v_values,
        'profiles', v_profiles
    );
END;
$$;
