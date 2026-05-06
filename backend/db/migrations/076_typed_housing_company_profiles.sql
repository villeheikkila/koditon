CREATE TABLE IF NOT EXISTS public.sale_listing_apartment_profiles (
    sale_listing_id uuid NOT NULL PRIMARY KEY REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE CASCADE,
    housing_company_id uuid REFERENCES public.housing_companies(housing_company_id) ON DELETE SET NULL,
    property_unit_id uuid REFERENCES public.property_units(property_unit_id) ON DELETE SET NULL,
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

CREATE INDEX IF NOT EXISTS idx_sale_listing_apartment_profiles_company
ON public.sale_listing_apartment_profiles (housing_company_id);

CREATE TABLE IF NOT EXISTS public.housing_company_systems (
    housing_company_system_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    housing_company_system_type text NOT NULL,
    housing_company_system_status text DEFAULT 'unknown' NOT NULL,
    housing_company_system_last_renovated_year integer,
    housing_company_system_next_expected_start_year integer,
    housing_company_system_next_expected_end_year integer,
    housing_company_system_confidence text DEFAULT 'low' NOT NULL,
    housing_company_system_evidence_level text DEFAULT 'none' NOT NULL,
    housing_company_system_summary text,
    housing_company_system_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (housing_company_system_type = ANY (ARRAY['pipes','water_supply','sewer','roof','facade','windows','balconies','elevator','heating','ventilation','drainage','electrical','yard','common_areas']::text[])),
    CHECK (housing_company_system_status = ANY (ARRAY['unknown','original','maintained','partly_renewed','renewed','under_study','planned','under_construction','risk']::text[])),
    CHECK (housing_company_system_confidence = ANY (ARRAY['low','medium','high']::text[])),
    CHECK (housing_company_system_evidence_level = ANY (ARRAY['none','ad_only','multiple_ads','announcement','manager_certificate','financial_statement','manual','forecast']::text[]))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_housing_company_systems_unique
ON public.housing_company_systems (housing_company_id, housing_company_system_type);

CREATE TABLE IF NOT EXISTS public.housing_company_renovations (
    housing_company_renovation_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    housing_company_id uuid NOT NULL REFERENCES public.housing_companies(housing_company_id) ON DELETE CASCADE,
    source_sale_listing_id uuid REFERENCES public.property_source_offerings(sale_listing_id) ON DELETE SET NULL,
    housing_company_renovation_category text NOT NULL,
    housing_company_renovation_status text NOT NULL,
    housing_company_renovation_stage text DEFAULT 'unknown' NOT NULL,
    housing_company_renovation_scope text DEFAULT 'unknown' NOT NULL,
    housing_company_renovation_responsibility text DEFAULT 'unknown' NOT NULL,
    housing_company_renovation_year integer,
    housing_company_renovation_window_start_year integer,
    housing_company_renovation_window_end_year integer,
    housing_company_renovation_cost_estimate_eur bigint,
    housing_company_renovation_confidence text DEFAULT 'low' NOT NULL,
    housing_company_renovation_evidence_level text DEFAULT 'none' NOT NULL,
    housing_company_renovation_summary text,
    housing_company_renovation_created_at timestamptz DEFAULT now() NOT NULL,
    housing_company_renovation_updated_at timestamptz DEFAULT now() NOT NULL,
    CHECK (housing_company_renovation_category = ANY (ARRAY['pipe','water_supply','sewer','roof','facade','window','balcony','elevator','heating','ventilation','drainage','electricity','yard','common_areas','other']::text[])),
    CHECK (housing_company_renovation_status = ANY (ARRAY['done','planned','suspected','forecast','cancelled','unknown']::text[])),
    CHECK (housing_company_renovation_stage = ANY (ARRAY['unknown','study','condition_assessment','planning','tendering','execution','completed']::text[])),
    CHECK (housing_company_renovation_scope = ANY (ARRAY['unknown','full','partial','maintenance']::text[])),
    CHECK (housing_company_renovation_responsibility = ANY (ARRAY['unknown','housing_company','shareholder','mixed']::text[])),
    CHECK (housing_company_renovation_confidence = ANY (ARRAY['low','medium','high']::text[])),
    CHECK (housing_company_renovation_evidence_level = ANY (ARRAY['none','ad_only','multiple_ads','announcement','manager_certificate','financial_statement','manual','forecast']::text[]))
);

CREATE INDEX IF NOT EXISTS idx_housing_company_renovations_company
ON public.housing_company_renovations (housing_company_id);

CREATE INDEX IF NOT EXISTS idx_housing_company_renovations_timing
ON public.housing_company_renovations (
    housing_company_renovation_status,
    housing_company_renovation_year,
    housing_company_renovation_window_start_year
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_housing_company_renovations_source_unique
ON public.housing_company_renovations (
    housing_company_id,
    source_sale_listing_id,
    housing_company_renovation_category,
    housing_company_renovation_status,
    housing_company_renovation_stage,
    housing_company_renovation_scope,
    COALESCE(housing_company_renovation_year, -1),
    md5(COALESCE(housing_company_renovation_summary, ''))
)
WHERE source_sale_listing_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS public.field_sources (
    field_source_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    field_source_target_table text NOT NULL,
    field_source_target_id uuid NOT NULL,
    field_source_target_field text NOT NULL,
    field_source_source_record_table text NOT NULL,
    field_source_source_record_id uuid NOT NULL,
    field_source_source_path text,
    field_source_evidence_text text,
    field_source_method text NOT NULL,
    field_source_confidence double precision DEFAULT 1 NOT NULL,
    field_source_observed_at timestamptz DEFAULT now() NOT NULL,
    field_source_valid_from date,
    field_source_valid_until date,
    CHECK (field_source_method = ANY (ARRAY['provider_field','llm','manual','parser','forecast']::text[])),
    CHECK (field_source_confidence >= 0 AND field_source_confidence <= 1)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_field_sources_unique
ON public.field_sources (
    field_source_target_table,
    field_source_target_id,
    field_source_target_field,
    field_source_source_record_table,
    field_source_source_record_id,
    COALESCE(field_source_source_path, ''),
    field_source_method
);

CREATE INDEX IF NOT EXISTS idx_field_sources_target
ON public.field_sources (
    field_source_target_table,
    field_source_target_id,
    field_source_target_field
);

CREATE INDEX IF NOT EXISTS idx_field_sources_source
ON public.field_sources (
    field_source_source_record_table,
    field_source_source_record_id
);
