CREATE TABLE IF NOT EXISTS public.physical_buildings (
    physical_building_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    housing_company_id uuid REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    physical_building_identity_key text NOT NULL UNIQUE,
    physical_building_address_norm text,
    physical_building_postal_norm text,
    physical_building_city_norm text,
    physical_building_build_year integer,
    physical_building_floor_count integer,
    physical_building_apartment_count integer,
    physical_building_elevator boolean,
    physical_building_latitude double precision,
    physical_building_longitude double precision,
    physical_building_created_at timestamptz DEFAULT now() NOT NULL,
    physical_building_updated_at timestamptz DEFAULT now() NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_physical_buildings_housing_company
ON public.physical_buildings (housing_company_id);
ALTER TABLE public.property_units
    ADD COLUMN IF NOT EXISTS physical_building_id uuid REFERENCES public.physical_buildings(physical_building_id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_property_units_physical_building
ON public.property_units (physical_building_id);
CREATE TABLE IF NOT EXISTS public.property_claims (
    property_claim_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_claim_target_type text NOT NULL,
    property_claim_target_id uuid NOT NULL,
    property_claim_namespace text NOT NULL,
    property_claim_key text NOT NULL,
    property_claim_value_kind text NOT NULL,
    property_claim_value_text text,
    property_claim_value_number double precision,
    property_claim_value_bool boolean,
    property_claim_value_json jsonb,
    property_claim_source_record_table text NOT NULL,
    property_claim_source_record_id uuid NOT NULL,
    property_claim_source_path text,
    property_claim_evidence_text text,
    property_claim_method text NOT NULL,
    property_claim_confidence double precision DEFAULT 0.5 NOT NULL,
    property_claim_source_reliability double precision DEFAULT 0.5 NOT NULL,
    property_claim_model text,
    property_claim_prompt_version text,
    property_claim_observed_at timestamptz DEFAULT now() NOT NULL,
    property_claim_valid_from date,
    property_claim_valid_until date,
    property_claim_created_at timestamptz DEFAULT now() NOT NULL,
    property_claim_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (property_claim_target_type = ANY (ARRAY['sale_listing','property_unit','physical_building','housing_company','transaction','document']::text[])),
    CHECK (property_claim_value_kind = ANY (ARRAY['text','number','bool','json']::text[])),
    CHECK (property_claim_method = ANY (ARRAY['provider_field','llm','manual','parser','forecast']::text[])),
    CHECK (property_claim_confidence >= 0 AND property_claim_confidence <= 1),
    CHECK (property_claim_source_reliability >= 0 AND property_claim_source_reliability <= 1)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_claims_unique_source
ON public.property_claims (
    property_claim_target_type,
    property_claim_target_id,
    property_claim_namespace,
    property_claim_key,
    property_claim_source_record_table,
    property_claim_source_record_id,
    COALESCE(property_claim_source_path, ''),
    property_claim_method
);
CREATE INDEX IF NOT EXISTS idx_property_claims_target
ON public.property_claims (
    property_claim_target_type,
    property_claim_target_id,
    property_claim_namespace,
    property_claim_key
);
CREATE TABLE IF NOT EXISTS public.apartment_profiles (
    apartment_profile_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_unit_id uuid NOT NULL UNIQUE REFERENCES public.property_units(property_unit_id) ON DELETE CASCADE,
    housing_company_id uuid REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    physical_building_id uuid REFERENCES public.physical_buildings(physical_building_id) ON DELETE SET NULL,
    source_sale_listing_id uuid REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE SET NULL,
    apartment_profile_area_m2 double precision,
    apartment_profile_living_area_m2 double precision,
    apartment_profile_room_layout text,
    apartment_profile_room_count integer,
    apartment_profile_bedroom_count integer,
    apartment_profile_floor_level integer,
    apartment_profile_total_floors integer,
    apartment_profile_kitchen_type text,
    apartment_profile_layout_quality text,
    apartment_profile_awkward_layout boolean,
    apartment_profile_condition text,
    apartment_profile_kitchen_condition text,
    apartment_profile_bathroom_condition text,
    apartment_profile_surface_renovation_need boolean,
    apartment_profile_modernization_need boolean,
    apartment_profile_sauna boolean,
    apartment_profile_balcony boolean,
    apartment_profile_balcony_glazing boolean,
    apartment_profile_parking_type text,
    apartment_profile_storage_quality text,
    apartment_profile_view_quality text,
    apartment_profile_noise_risk boolean,
    apartment_profile_accessibility text,
    apartment_profile_confidence text DEFAULT 'low' NOT NULL,
    apartment_profile_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (apartment_profile_kitchen_type IS NULL OR apartment_profile_kitchen_type = ANY (ARRAY['separate','open','kitchenette','unknown']::text[])),
    CHECK (apartment_profile_layout_quality IS NULL OR apartment_profile_layout_quality = ANY (ARRAY['weak','average','good','excellent','unknown']::text[])),
    CHECK (apartment_profile_condition IS NULL OR apartment_profile_condition = ANY (ARRAY['poor','fair','good','excellent','new','unknown']::text[])),
    CHECK (apartment_profile_kitchen_condition IS NULL OR apartment_profile_kitchen_condition = ANY (ARRAY['poor','fair','good','excellent','new','unknown']::text[])),
    CHECK (apartment_profile_bathroom_condition IS NULL OR apartment_profile_bathroom_condition = ANY (ARRAY['poor','fair','good','excellent','new','unknown']::text[])),
    CHECK (apartment_profile_parking_type IS NULL OR apartment_profile_parking_type = ANY (ARRAY['none','street','yard','garage','carport','separate_share','unknown']::text[])),
    CHECK (apartment_profile_storage_quality IS NULL OR apartment_profile_storage_quality = ANY (ARRAY['weak','normal','good','unknown']::text[])),
    CHECK (apartment_profile_view_quality IS NULL OR apartment_profile_view_quality = ANY (ARRAY['weak','normal','good','excellent','unknown']::text[])),
    CHECK (apartment_profile_accessibility IS NULL OR apartment_profile_accessibility = ANY (ARRAY['poor','average','good','unknown']::text[])),
    CHECK (apartment_profile_confidence = ANY (ARRAY['low','medium','high']::text[]))
);
CREATE INDEX IF NOT EXISTS idx_apartment_profiles_company
ON public.apartment_profiles (housing_company_id);
CREATE TABLE IF NOT EXISTS public.building_profiles (
    building_profile_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    physical_building_id uuid NOT NULL UNIQUE REFERENCES public.physical_buildings(physical_building_id) ON DELETE CASCADE,
    housing_company_id uuid REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    building_profile_build_year integer,
    building_profile_floor_count integer,
    building_profile_apartment_count integer,
    building_profile_energy_class text,
    building_profile_heating_method text,
    building_profile_material text,
    building_profile_roof_type text,
    building_profile_roof_material text,
    building_profile_elevator boolean,
    building_profile_confidence text DEFAULT 'low' NOT NULL,
    building_profile_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (building_profile_confidence = ANY (ARRAY['low','medium','high']::text[]))
);
CREATE TABLE IF NOT EXISTS public.housing_company_profiles (
    housing_company_profile_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    housing_company_id uuid NOT NULL UNIQUE REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    housing_company_profile_name text,
    housing_company_profile_business_id text,
    housing_company_profile_build_year integer,
    housing_company_profile_apartment_count integer,
    housing_company_profile_plot_ownership_type text,
    housing_company_profile_energy_class text,
    housing_company_profile_maintenance_risk text DEFAULT 'unknown' NOT NULL,
    housing_company_profile_financial_risk text DEFAULT 'unknown' NOT NULL,
    housing_company_profile_repair_backlog_risk text DEFAULT 'unknown' NOT NULL,
    housing_company_profile_confidence text DEFAULT 'low' NOT NULL,
    housing_company_profile_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (housing_company_profile_maintenance_risk = ANY (ARRAY['unknown','low','medium','high']::text[])),
    CHECK (housing_company_profile_financial_risk = ANY (ARRAY['unknown','low','medium','high']::text[])),
    CHECK (housing_company_profile_repair_backlog_risk = ANY (ARRAY['unknown','low','medium','high']::text[])),
    CHECK (housing_company_profile_confidence = ANY (ARRAY['low','medium','high']::text[]))
);
CREATE TABLE IF NOT EXISTS public.property_quality_scores (
    property_quality_score_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_quality_score_target_type text NOT NULL,
    property_quality_score_target_id uuid NOT NULL,
    property_quality_score_dimension text NOT NULL,
    property_quality_score_value integer NOT NULL,
    property_quality_score_confidence text DEFAULT 'low' NOT NULL,
    property_quality_score_reasons jsonb DEFAULT '[]'::jsonb NOT NULL,
    property_quality_score_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (property_quality_score_target_type = ANY (ARRAY['property_unit','physical_building','housing_company','property_offering']::text[])),
    CHECK (property_quality_score_value >= 0 AND property_quality_score_value <= 100),
    CHECK (property_quality_score_confidence = ANY (ARRAY['low','medium','high']::text[]))
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_quality_scores_unique
ON public.property_quality_scores (
    property_quality_score_target_type,
    property_quality_score_target_id,
    property_quality_score_dimension
);
CREATE TABLE IF NOT EXISTS public.property_valuation_runs (
    property_valuation_run_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_offering_id uuid REFERENCES public.property_offerings(property_offering_id) ON DELETE SET NULL,
    property_unit_id uuid REFERENCES public.property_units(property_unit_id) ON DELETE SET NULL,
    housing_company_id uuid REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    property_valuation_run_model_version text NOT NULL,
    property_valuation_run_market_value_low bigint,
    property_valuation_run_market_value_high bigint,
    property_valuation_run_risk_adjusted_value_low bigint,
    property_valuation_run_risk_adjusted_value_high bigint,
    property_valuation_run_recommended_offer_low bigint,
    property_valuation_run_recommended_offer_high bigint,
    property_valuation_run_verdict text NOT NULL,
    property_valuation_run_confidence text NOT NULL,
    property_valuation_run_reasons jsonb DEFAULT '[]'::jsonb NOT NULL,
    property_valuation_run_missing_evidence text[] DEFAULT ARRAY[]::text[] NOT NULL,
    property_valuation_run_created_at timestamptz DEFAULT now() NOT NULL
);
DROP TABLE IF EXISTS public.field_sources;
DROP TABLE IF EXISTS public.sale_listing_apartment_profiles;
DROP TABLE IF EXISTS public.property_valuation_facts;
DROP TABLE IF EXISTS public.property_source_offering_valuation_facts;
