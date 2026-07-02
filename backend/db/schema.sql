CREATE SCHEMA IF NOT EXISTS runtime;

create table public.dimension_claims (
  property_dimension_claim_id uuid default gen_random_uuid() not null constraint dimension_claims_pkey primary key,
  property_dimension_projection_run_id uuid not null,
  projection_version text not null,
  claim_scope text not null,
  target_type text not null,
  target_id uuid not null,
  dimension_key text not null,
  value jsonb not null,
  value_kind text not null,
  unit text,
  source_table text not null,
  source_id uuid not null,
  source_field text,
  source_claim_id uuid,
  source_observed_at timestamp with time zone,
  valid_from date,
  valid_until date,
  confidence double precision default 0.5 not null,
  source_reliability double precision default 0.5 not null,
  evidence jsonb default '{}'::jsonb not null,
  extraction_model text,
  extraction_prompt_version text,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null,
  constraint dimension_claims_claim_scope_check CHECK ((claim_scope = ANY (ARRAY['source'::text, 'manual'::text]))),
  constraint dimension_claims_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint dimension_claims_source_reliability_check CHECK (((source_reliability >= (0)::double precision) AND (source_reliability <= (1)::double precision))),
  constraint dimension_claims_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text]))),
  constraint dimension_claims_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

CREATE INDEX idx_dimension_claims_dimension ON public.dimension_claims USING btree (dimension_key);
CREATE INDEX idx_dimension_claims_source ON public.dimension_claims USING btree (source_table, source_id, projection_version);
CREATE INDEX idx_dimension_claims_source_claim ON public.dimension_claims USING btree (source_claim_id);
CREATE INDEX idx_dimension_claims_target ON public.dimension_claims USING btree (claim_scope, target_type, target_id, dimension_key);
CREATE UNIQUE INDEX idx_dimension_claims_unique_source ON public.dimension_claims USING btree (claim_scope, target_type, target_id, dimension_key, source_table, source_id, COALESCE(source_field, ''::text), projection_version);
CREATE INDEX idx_dimension_claims_value_gin ON public.dimension_claims USING gin (value jsonb_path_ops);

create table public.dimension_profiles (
  target_type text not null,
  target_id uuid not null,
  dimensions jsonb default '{}'::jsonb not null,
  metadata jsonb default '{}'::jsonb not null,
  conflicts jsonb default '{}'::jsonb not null,
  resolved_at timestamp with time zone default now() not null,
  constraint dimension_profiles_pkey PRIMARY KEY (target_type, target_id),
  constraint dimension_profiles_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);

CREATE INDEX idx_dimension_profiles_building_build_year ON public.dimension_profiles USING btree ((((dimensions #>> '{building,build_year}'::text[]))::integer)) WHERE (target_type = 'building'::text);
CREATE INDEX idx_dimension_profiles_dimensions_gin ON public.dimension_profiles USING gin (dimensions jsonb_path_ops);
CREATE INDEX idx_dimension_profiles_unit_area ON public.dimension_profiles USING btree ((((dimensions #>> '{unit,area_m2}'::text[]))::double precision)) WHERE (target_type = 'unit'::text);
CREATE INDEX idx_dimension_profiles_unit_total_charge ON public.dimension_profiles USING btree ((((dimensions #>> '{charges,total_monthly_eur}'::text[]))::double precision)) WHERE (target_type = 'unit'::text);

create table public.dimension_values (
  target_type text not null,
  target_id uuid not null,
  dimension_key text not null,
  value jsonb not null,
  value_kind text not null,
  unit text,
  confidence double precision not null,
  selected_claim_id uuid,
  selected_reason text not null,
  conflict_status text default 'none'::text not null,
  supporting_claim_ids uuid[] default '{}'::uuid[] not null,
  rejected_claim_ids uuid[] default '{}'::uuid[] not null,
  resolved_at timestamp with time zone default now() not null,
  constraint dimension_values_pkey PRIMARY KEY (target_type, target_id, dimension_key),
  constraint dimension_values_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint dimension_values_conflict_status_check CHECK ((conflict_status = ANY (ARRAY['none'::text, 'compatible'::text, 'conflicting'::text, 'manual_override'::text]))),
  constraint dimension_values_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text]))),
  constraint dimension_values_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

CREATE INDEX idx_dimension_values_dimension ON public.dimension_values USING btree (dimension_key);
CREATE INDEX idx_dimension_values_selected_claim ON public.dimension_values USING btree (selected_claim_id);

create table public.houses (
  house_id uuid not null constraint houses_pkey primary key,
  identity_key text not null,
  address_norm text,
  postal_norm text,
  city_norm text,
  latitude double precision,
  longitude double precision,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null
);

CREATE UNIQUE INDEX houses_identity_key_key ON public.houses USING btree (identity_key);
CREATE INDEX idx_houses_address ON public.houses USING btree (postal_norm, city_norm, address_norm);
CREATE INDEX idx_houses_lat_lng ON public.houses USING btree (latitude, longitude) WHERE ((latitude IS NOT NULL) AND (longitude IS NOT NULL));

create table public.housing_companies (
  housing_company_id uuid default gen_random_uuid() not null constraint property_buildings_pkey primary key,
  housing_company_identity_key text not null constraint property_buildings_property_building_identity_key_key unique,
  housing_company_postal_norm text,
  housing_company_city_norm text,
  housing_company_address_norm text,
  housing_company_name text,
  housing_company_business_id text,
  housing_company_build_year integer,
  housing_company_floor_count integer,
  housing_company_apartment_count integer,
  housing_company_elevator boolean,
  housing_company_energy_efficiency_label text,
  housing_company_match_reasons jsonb default '{}'::jsonb not null,
  housing_company_created_at timestamp with time zone default now() not null,
  housing_company_updated_at timestamp with time zone default now() not null,
  housing_company_geom postgis.geometry(Point,4326)
);

CREATE INDEX idx_housing_companies_address ON public.housing_companies USING btree (housing_company_postal_norm, housing_company_city_norm, housing_company_address_norm);
CREATE INDEX idx_housing_companies_business_id ON public.housing_companies USING btree (housing_company_business_id) WHERE ((housing_company_business_id IS NOT NULL) AND (housing_company_business_id <> ''::text));
CREATE INDEX idx_housing_companies_geom ON public.housing_companies USING gist (housing_company_geom);

create table public.housing_company_merge_decisions (
  housing_company_merge_decision_id uuid default gen_random_uuid() not null constraint housing_company_merge_decisions_pkey primary key,
  source_housing_company_id uuid not null constraint housing_company_merge_decisions_source_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE CASCADE,
  target_housing_company_id uuid not null constraint housing_company_merge_decisions_target_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE CASCADE,
  housing_company_merge_decision_status text default 'accepted'::text not null,
  housing_company_merge_decision_method text not null,
  housing_company_merge_decision_score integer,
  housing_company_merge_decision_confidence text,
  housing_company_merge_decision_reasons jsonb default '{}'::jsonb not null,
  housing_company_merge_decision_created_at timestamp with time zone default now() not null,
  housing_company_merge_decision_decided_at timestamp with time zone default now() not null,
  constraint housing_company_merge_decision_distinct_check CHECK ((source_housing_company_id <> target_housing_company_id)),
  constraint housing_company_merge_decision_method_check CHECK ((housing_company_merge_decision_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
  constraint housing_company_merge_decision_status_check CHECK ((housing_company_merge_decision_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])))
);

CREATE UNIQUE INDEX idx_housing_company_merge_decisions_active_pair ON public.housing_company_merge_decisions USING btree (source_housing_company_id, target_housing_company_id) WHERE (housing_company_merge_decision_status <> 'rejected'::text);
CREATE INDEX idx_housing_company_merge_decisions_source ON public.housing_company_merge_decisions USING btree (source_housing_company_id, housing_company_merge_decision_status);
CREATE INDEX idx_housing_company_merge_decisions_target ON public.housing_company_merge_decisions USING btree (target_housing_company_id, housing_company_merge_decision_status);

create table public.listings (
  listing_id uuid not null constraint listings_pkey primary key,
  listing_type text not null,
  listing_status text,
  primary_source_listing_id uuid,
  unit_id uuid,
  house_id uuid,
  first_seen_at timestamp with time zone,
  last_seen_at timestamp with time zone,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null,
  constraint listings_listing_type_check CHECK ((listing_type = ANY (ARRAY['sale'::text])))
);

CREATE INDEX idx_listings_house ON public.listings USING btree (house_id) WHERE (house_id IS NOT NULL);
CREATE INDEX idx_listings_last_seen ON public.listings USING btree (last_seen_at DESC);
CREATE INDEX idx_listings_primary_source_listing ON public.listings USING btree (primary_source_listing_id);
CREATE INDEX idx_listings_unit ON public.listings USING btree (unit_id) WHERE (unit_id IS NOT NULL);

create table public.physical_buildings (
  physical_building_id uuid default gen_random_uuid() not null constraint physical_buildings_pkey primary key,
  housing_company_id uuid constraint physical_buildings_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE SET NULL,
  physical_building_identity_key text not null constraint physical_buildings_physical_building_identity_key_key unique,
  physical_building_address_norm text,
  physical_building_postal_norm text,
  physical_building_city_norm text,
  physical_building_build_year integer,
  physical_building_floor_count integer,
  physical_building_apartment_count integer,
  physical_building_elevator boolean,
  physical_building_latitude double precision,
  physical_building_longitude double precision,
  physical_building_created_at timestamp with time zone default now() not null,
  physical_building_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_physical_buildings_housing_company ON public.physical_buildings USING btree (housing_company_id);
CREATE INDEX idx_physical_buildings_lat_lng ON public.physical_buildings USING btree (physical_building_latitude, physical_building_longitude) WHERE ((physical_building_latitude IS NOT NULL) AND (physical_building_longitude IS NOT NULL));

create table public.price_links (
  price_link_id uuid default gen_random_uuid() not null constraint price_links_pkey primary key,
  target_type text not null,
  target_id uuid not null,
  prices_transaction_id uuid not null constraint price_links_prices_transaction_id_fkey references origin.prices_transactions(prices_transaction_id) ON DELETE CASCADE,
  link_status text not null,
  link_method text not null,
  link_score integer not null,
  link_reasons jsonb default '{}'::jsonb not null,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null,
  constraint price_links_link_method_check CHECK ((link_method = ANY (ARRAY['sync_auto'::text, 'source_match_auto'::text, 'document_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
  constraint price_links_link_status_check CHECK ((link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text, 'superseded'::text]))),
  constraint price_links_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'source_listing'::text, 'source_building_announcement'::text, 'building'::text, 'housing_company'::text])))
);

CREATE INDEX idx_price_links_target ON public.price_links USING btree (target_type, target_id, link_status);
CREATE INDEX idx_price_links_transaction ON public.price_links USING btree (prices_transaction_id, link_status);
CREATE UNIQUE INDEX price_links_one_confirmed_listing_per_transaction ON public.price_links USING btree (prices_transaction_id) WHERE ((target_type = 'listing'::text) AND (link_status = 'confirmed'::text));
CREATE UNIQUE INDEX price_links_unique_target_transaction ON public.price_links USING btree (target_type, target_id, prices_transaction_id);

create table public.property_dimension_catalog (
  dimension_key text not null constraint property_dimension_catalog_pkey primary key,
  target_type text not null,
  value_kind text not null,
  unit text,
  profile_section text not null,
  profile_key text not null,
  promoted_to_valuation boolean default false not null,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null,
  constraint property_dimension_catalog_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text]))),
  constraint property_dimension_catalog_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

create table public.property_dimension_dirty_targets (
  target_type text not null,
  target_id uuid not null,
  dirty_reasons text[] default '{}'::text[] not null,
  dirty_at timestamp with time zone default now() not null,
  queued_at timestamp with time zone,
  resolved_at timestamp with time zone,
  constraint property_dimension_dirty_targets_pkey PRIMARY KEY (target_type, target_id),
  constraint property_dimension_dirty_targets_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'transaction'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);

CREATE INDEX idx_property_dimension_dirty_targets_queue ON public.property_dimension_dirty_targets USING btree (dirty_at) WHERE ((resolved_at IS NULL) OR (resolved_at < dirty_at));

create table public.property_dimension_projection_runs (
  property_dimension_projection_run_id uuid default gen_random_uuid() not null constraint property_dimension_projection_runs_pkey primary key,
  projection_type text not null,
  projection_version text not null,
  source_table text not null,
  source_id uuid not null,
  status text not null,
  result jsonb default '{}'::jsonb not null,
  error_text text,
  started_at timestamp with time zone default now() not null,
  finished_at timestamp with time zone,
  constraint property_dimension_projection_runs_projection_type_check CHECK ((projection_type = ANY (ARRAY['source_claims'::text, 'renovation_events'::text, 'resolved_values'::text, 'profiles'::text, 'system_profiles'::text]))),
  constraint property_dimension_projection_runs_status_check CHECK ((status = ANY (ARRAY['running'::text, 'succeeded'::text, 'failed'::text])))
);

CREATE INDEX idx_property_dimension_projection_runs_source ON public.property_dimension_projection_runs USING btree (projection_type, source_table, source_id, projection_version, started_at DESC);

create table public.property_dimension_resolution_policies (
  dimension_key text not null constraint property_dimension_resolution_policies_pkey primary key constraint property_dimension_resolution_policies_dimension_key_fkey references property_dimension_catalog(dimension_key),
  strategy text not null,
  freshness_half_life_days integer,
  conflict_tolerance jsonb default '{}'::jsonb not null,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null,
  constraint property_dimension_resolution_policies_strategy_check CHECK ((strategy = ANY (ARRAY['manual_override'::text, 'latest_reliable'::text, 'highest_reliability'::text, 'document_preferred'::text, 'stable_identity'::text, 'numeric_consensus'::text])))
);

create table public.property_dimension_source_priorities (
  dimension_key text not null constraint property_dimension_source_priorities_dimension_key_fkey references property_dimension_catalog(dimension_key),
  source_table text not null,
  source_field text,
  priority integer not null,
  default_reliability double precision not null,
  constraint property_dimension_source_priorities_default_reliability_check CHECK (((default_reliability >= (0)::double precision) AND (default_reliability <= (1)::double precision)))
);

CREATE UNIQUE INDEX idx_property_dimension_source_priorities_unique ON public.property_dimension_source_priorities USING btree (dimension_key, source_table, COALESCE(source_field, ''::text));

create table public.property_document_extraction_runs (
  property_document_extraction_run_id uuid default gen_random_uuid() not null constraint property_document_extraction_runs_pkey primary key,
  property_document_id uuid not null constraint property_document_extraction_runs_property_document_id_fkey references property_documents(property_document_id) ON DELETE CASCADE,
  property_document_extraction_run_model text not null,
  property_document_extraction_run_prompt_version text not null,
  property_document_extraction_run_status text not null,
  property_document_extraction_run_raw_json jsonb,
  property_document_extraction_run_error text,
  property_document_extraction_run_started_at timestamp with time zone default now() not null,
  property_document_extraction_run_finished_at timestamp with time zone,
  constraint property_document_extraction_runs_status_check CHECK ((property_document_extraction_run_status = ANY (ARRAY['running'::text, 'succeeded'::text, 'failed'::text])))
);

CREATE INDEX idx_property_document_extraction_runs_document ON public.property_document_extraction_runs USING btree (property_document_id, property_document_extraction_run_started_at DESC);

create table public.property_document_extractions (
  property_document_extraction_id uuid default gen_random_uuid() not null constraint property_document_extractions_pkey primary key,
  property_document_id uuid not null constraint property_document_extractions_property_document_id_fkey references property_documents(property_document_id) ON DELETE CASCADE,
  property_document_extraction_kind text not null,
  property_document_extraction_schema_version text not null,
  property_document_extraction_model text not null,
  property_document_extraction_prompt_version text not null,
  property_document_extraction_source_json jsonb not null,
  property_document_extraction_status text default 'succeeded'::text not null,
  property_document_extraction_error text,
  property_document_extraction_created_at timestamp with time zone default now() not null,
  property_document_extraction_extracted_at timestamp with time zone default now() not null,
  property_document_extraction_superseded_at timestamp with time zone,
  constraint property_document_extractions_kind_check CHECK ((property_document_extraction_kind = ANY (ARRAY['manager_certificate'::text]))),
  constraint property_document_extractions_schema_version_check CHECK ((property_document_extraction_schema_version <> ''::text)),
  constraint property_document_extractions_status_check CHECK ((property_document_extraction_status = ANY (ARRAY['succeeded'::text, 'failed'::text, 'superseded'::text])))
);

CREATE INDEX idx_property_document_extractions_document ON public.property_document_extractions USING btree (property_document_id, property_document_extraction_created_at DESC);
CREATE UNIQUE INDEX idx_property_document_extractions_latest ON public.property_document_extractions USING btree (property_document_id, property_document_extraction_kind) WHERE (property_document_extraction_superseded_at IS NULL);

create table public.property_documents (
  property_document_id uuid default gen_random_uuid() not null constraint property_documents_pkey primary key,
  property_offering_id uuid constraint property_documents_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  property_unit_id uuid constraint property_documents_property_unit_id_fkey references property_units(property_unit_id) ON DELETE SET NULL,
  physical_building_id uuid constraint property_documents_physical_building_id_fkey references physical_buildings(physical_building_id) ON DELETE SET NULL,
  housing_company_id uuid constraint property_documents_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE SET NULL,
  property_document_type text not null,
  property_document_filename text not null,
  property_document_mime_type text not null,
  property_document_size_bytes bigint not null,
  property_document_sha256 text not null,
  property_document_bytes bytea not null,
  property_document_extraction_status text default 'uploaded'::text not null,
  property_document_extraction_error text,
  property_document_uploaded_at timestamp with time zone default now() not null,
  property_document_extracted_at timestamp with time zone,
  property_document_created_at timestamp with time zone default now() not null,
  property_document_updated_at timestamp with time zone default now() not null,
  constraint property_documents_document_type_check CHECK ((property_document_type = ANY (ARRAY['manager_certificate'::text]))),
  constraint property_documents_extraction_status_check CHECK ((property_document_extraction_status = ANY (ARRAY['uploaded'::text, 'extracting'::text, 'extracted'::text, 'failed'::text]))),
  constraint property_documents_mime_type_check CHECK ((property_document_mime_type = 'application/pdf'::text)),
  constraint property_documents_sha256_check CHECK ((property_document_sha256 ~ '^[0-9a-f]{64}$'::text)),
  constraint property_documents_size_bytes_check CHECK (((property_document_size_bytes > 0) AND (property_document_size_bytes <= 26214400)))
);

CREATE UNIQUE INDEX idx_property_documents_detached_type_hash ON public.property_documents USING btree (property_document_type, property_document_sha256) WHERE (property_offering_id IS NULL);
CREATE INDEX idx_property_documents_housing_company ON public.property_documents USING btree (housing_company_id, property_document_type) WHERE (housing_company_id IS NOT NULL);
CREATE INDEX idx_property_documents_offering ON public.property_documents USING btree (property_offering_id, property_document_type, property_document_uploaded_at DESC);
CREATE UNIQUE INDEX idx_property_documents_offering_type_hash ON public.property_documents USING btree (property_offering_id, property_document_type, property_document_sha256);

create table public.property_houses (
  property_house_id uuid default gen_random_uuid() not null constraint property_houses_pkey primary key,
  property_house_identity_key text not null constraint property_houses_property_house_identity_key_key unique,
  property_house_address_norm text,
  property_house_postal_norm text,
  property_house_city_norm text,
  property_house_build_year integer,
  property_house_area_value double precision,
  property_house_plot_area_value double precision,
  property_house_rooms_count integer,
  property_house_latitude double precision,
  property_house_longitude double precision,
  property_house_match_reasons jsonb default '{}'::jsonb not null,
  primary_sale_listing_id uuid constraint property_houses_primary_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE SET NULL,
  property_house_created_at timestamp with time zone default now() not null,
  property_house_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_property_houses_address ON public.property_houses USING btree (property_house_postal_norm, property_house_city_norm, property_house_address_norm);
CREATE INDEX idx_property_houses_lat_lng ON public.property_houses USING btree (property_house_latitude, property_house_longitude) WHERE ((property_house_latitude IS NOT NULL) AND (property_house_longitude IS NOT NULL));

create table public.property_offering_merge_decisions (
  property_offering_merge_decision_id uuid default gen_random_uuid() not null constraint property_offering_merge_decisions_pkey primary key,
  source_property_offering_id uuid not null constraint property_offering_merge_decisi_source_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  target_property_offering_id uuid not null constraint property_offering_merge_decisi_target_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  property_offering_merge_decision_status text default 'accepted'::text not null,
  property_offering_merge_decision_method text not null,
  property_offering_merge_decision_score integer,
  property_offering_merge_decision_confidence text,
  property_offering_merge_decision_reasons jsonb default '{}'::jsonb not null,
  property_offering_merge_decision_created_at timestamp with time zone default now() not null,
  property_offering_merge_decision_decided_at timestamp with time zone default now() not null,
  constraint property_offering_merge_decision_distinct_check CHECK ((source_property_offering_id <> target_property_offering_id)),
  constraint property_offering_merge_decision_method_check CHECK ((property_offering_merge_decision_method = ANY (ARRAY['source_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
  constraint property_offering_merge_decision_status_check CHECK ((property_offering_merge_decision_status = ANY (ARRAY['proposed'::text, 'accepted'::text, 'rejected'::text, 'superseded'::text])))
);

CREATE UNIQUE INDEX idx_property_offering_merge_decisions_active_pair ON public.property_offering_merge_decisions USING btree (source_property_offering_id, target_property_offering_id) WHERE (property_offering_merge_decision_status <> 'rejected'::text);
CREATE INDEX idx_property_offering_merge_decisions_source ON public.property_offering_merge_decisions USING btree (source_property_offering_id, property_offering_merge_decision_status);
CREATE INDEX idx_property_offering_merge_decisions_target ON public.property_offering_merge_decisions USING btree (target_property_offering_id, property_offering_merge_decision_status);

create table public.property_offerings (
  property_offering_id uuid default gen_random_uuid() not null constraint property_offerings_pkey primary key,
  property_unit_id uuid constraint property_offerings_property_unit_id_fkey references property_units(property_unit_id) ON DELETE CASCADE,
  property_offering_identity_key text not null constraint property_offerings_property_offering_identity_key_key unique,
  property_offering_type text not null,
  property_offering_headline text not null,
  property_offering_asking_price bigint,
  property_offering_debt_free_price bigint,
  property_offering_price_per_m2 double precision,
  property_offering_first_seen_at timestamp with time zone,
  property_offering_last_seen_at timestamp with time zone,
  property_offering_status text,
  primary_sale_listing_id uuid constraint property_offerings_primary_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE SET NULL,
  property_offering_match_reasons jsonb default '{}'::jsonb not null,
  property_offering_created_at timestamp with time zone default now() not null,
  property_offering_updated_at timestamp with time zone default now() not null,
  property_house_id uuid constraint property_offerings_property_house_id_fkey references property_houses(property_house_id) ON DELETE CASCADE,
  constraint property_offerings_parent_check CHECK (((((property_unit_id IS NOT NULL))::integer + ((property_house_id IS NOT NULL))::integer) = 1)),
  constraint property_offerings_type_check CHECK ((property_offering_type = ANY (ARRAY['sale'::text])))
);

CREATE INDEX idx_property_offerings_house ON public.property_offerings USING btree (property_house_id) WHERE (property_house_id IS NOT NULL);
CREATE INDEX idx_property_offerings_primary_sale_listing ON public.property_offerings USING btree (primary_sale_listing_id);
CREATE INDEX idx_property_offerings_unit ON public.property_offerings USING btree (property_unit_id);

create table public.property_renovation_events (
  property_renovation_event_id uuid default gen_random_uuid() not null constraint property_renovation_events_pkey primary key,
  property_dimension_projection_run_id uuid not null constraint property_renovation_events_property_dimension_projection_r_fkey references property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE,
  projection_version text not null,
  event_scope text not null,
  target_type text not null,
  target_id uuid not null,
  source_table text not null,
  source_id uuid not null,
  source_field text,
  source_event_id uuid constraint property_renovation_events_source_event_id_fkey references property_renovation_events(property_renovation_event_id),
  category text not null,
  component text,
  status text not null,
  stage text,
  scope text,
  responsibility text,
  year integer,
  start_year integer,
  end_year integer,
  cost_estimate_eur bigint,
  summary text,
  evidence jsonb default '{}'::jsonb not null,
  confidence double precision default 0.5 not null,
  source_reliability double precision default 0.5 not null,
  created_at timestamp with time zone default now() not null,
  source_observed_at timestamp with time zone,
  constraint property_renovation_events_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint property_renovation_events_event_scope_check CHECK ((event_scope = ANY (ARRAY['source'::text, 'manual'::text]))),
  constraint property_renovation_events_source_reliability_check CHECK (((source_reliability >= (0)::double precision) AND (source_reliability <= (1)::double precision))),
  constraint property_renovation_events_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);

CREATE INDEX idx_property_renovation_events_source ON public.property_renovation_events USING btree (source_table, source_id, projection_version);
CREATE INDEX idx_property_renovation_events_source_event ON public.property_renovation_events USING btree (source_event_id);
CREATE INDEX idx_property_renovation_events_target ON public.property_renovation_events USING btree (event_scope, target_type, target_id, category, status);
CREATE INDEX idx_property_renovation_events_target_observed ON public.property_renovation_events USING btree (event_scope, target_type, target_id, category, status, source_observed_at DESC);
CREATE UNIQUE INDEX idx_property_renovation_events_unique_source ON public.property_renovation_events USING btree (event_scope, target_type, target_id, source_table, source_id, COALESCE(source_field, ''::text), category, status, COALESCE(stage, ''::text), COALESCE(scope, ''::text), COALESCE(year, '-1'::integer), COALESCE(start_year, '-1'::integer), COALESCE(end_year, '-1'::integer), md5(COALESCE(summary, ''::text)), projection_version);

create table public.property_source_offering_renovations (
  property_source_offering_renovation_id uuid default gen_random_uuid() not null constraint property_source_offering_renovations_pkey primary key,
  sale_listing_id uuid not null constraint property_source_offering_renovations_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  property_source_offering_renovation_source_field text not null,
  property_source_offering_renovation_category text not null,
  property_source_offering_renovation_status text not null,
  property_source_offering_renovation_year integer,
  property_source_offering_renovation_text text,
  property_source_offering_renovation_confidence integer default 100 not null,
  property_source_offering_renovation_created_at timestamp with time zone default now() not null,
  property_source_offering_renovation_updated_at timestamp with time zone default now() not null,
  property_source_offering_renovation_component text,
  property_source_offering_renovation_scope text,
  property_source_offering_renovation_stage text,
  property_source_offering_renovation_responsibility text,
  property_source_offering_renovation_cost_estimate_eur bigint,
  constraint property_source_offering_renovation_status_check CHECK ((property_source_offering_renovation_status = ANY (ARRAY['done'::text, 'planned'::text, 'unknown'::text])))
);

CREATE INDEX idx_property_source_offering_renovations_listing ON public.property_source_offering_renovations USING btree (sale_listing_id);
CREATE UNIQUE INDEX idx_property_source_offering_renovations_unique ON public.property_source_offering_renovations USING btree (sale_listing_id, property_source_offering_renovation_source_field, property_source_offering_renovation_category, property_source_offering_renovation_status, COALESCE(property_source_offering_renovation_year, 0), COALESCE(property_source_offering_renovation_component, ''::text), COALESCE(property_source_offering_renovation_stage, ''::text));

create table public.property_source_offerings (
  sale_listing_id uuid default gen_random_uuid() not null constraint sale_listings_pkey primary key,
  shortcut_ad_id bigint constraint sale_listings_shortcut_ad_id_fkey references origin.shortcut_ads(shortcut_ad_id) ON DELETE SET NULL,
  frontdoor_ad_id uuid constraint sale_listings_frontdoor_ad_id_fkey references origin.frontdoor_ads(frontdoor_ad_id) ON DELETE SET NULL,
  frontdoor_building_announcement_id uuid constraint sale_listings_frontdoor_building_announcement_id_fkey references origin.frontdoor_building_announcements(frontdoor_building_announcement_id) ON DELETE SET NULL,
  prices_transaction_id uuid constraint sale_listings_prices_transaction_id_fkey references origin.prices_transactions(prices_transaction_id) ON DELETE SET NULL,
  sale_listing_source_provider text not null,
  sale_listing_source_kind text not null,
  sale_listing_native_id text not null,
  sale_listing_canonical_id text not null constraint sale_listings_canonical_id_key unique,
  sale_listing_url text,
  sale_listing_headline text not null,
  sale_listing_street_address text,
  sale_listing_city text,
  sale_listing_postal text,
  sale_listing_asking_price bigint,
  sale_listing_area_value double precision,
  sale_listing_room_layout text,
  sale_listing_last_seen_at timestamp with time zone,
  sale_listing_published_at timestamp with time zone,
  sale_listing_search_text text,
  sale_listing_created_at timestamp with time zone default now() not null,
  sale_listing_updated_at timestamp with time zone default now() not null,
  sale_listing_street_name text,
  sale_listing_street_number text,
  sale_listing_building_letter text,
  sale_listing_apartment text,
  sale_listing_street_name_norm text,
  sale_listing_street_number_norm text,
  sale_listing_building_letter_norm text,
  sale_listing_city_norm text,
  sale_listing_postal_norm text,
  sale_listing_address_norm text,
  sale_listing_address_components jsonb,
  sale_listing_building_match_key text,
  sale_listing_street_match_key text,
  sale_listing_unit_match_key text,
  sale_listing_price_per_m2 double precision,
  sale_listing_debt_free_price bigint,
  sale_listing_debt_share_amount bigint,
  sale_listing_rooms_count integer,
  sale_listing_floor_level integer,
  sale_listing_total_floors integer,
  sale_listing_build_year integer,
  sale_listing_condition text,
  sale_listing_energy_class text,
  sale_listing_description_text text,
  sale_listing_property_type_raw text,
  sale_listing_property_type_code text,
  sale_listing_room_category_code text,
  sale_listing_floor_text text,
  sale_listing_elevator boolean,
  sale_listing_plot_type_raw text,
  sale_listing_plot_type_code text,
  sale_listing_energy_efficiency_label text,
  sale_listing_energy_efficiency_class_code text,
  sale_listing_energy_efficiency_standard_year integer,
  sale_listing_energy_efficiency_status text,
  sale_listing_energy_efficiency_match_code text,
  sale_listing_first_seen_at timestamp with time zone,
  sale_listing_prices_match_status text,
  sale_listing_prices_match_next_attempt_at timestamp with time zone,
  sale_listing_prices_match_last_attempted_at timestamp with time zone,
  sale_listing_prices_match_attempt_count integer default 0 not null,
  sale_listing_prices_match_expires_at timestamp with time zone,
  sale_listing_prices_match_run_id uuid constraint sale_listings_sale_listing_prices_match_run_id_fkey references sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE SET NULL,
  sale_listing_plot_owned boolean,
  sale_listing_source_match_status text,
  sale_listing_source_match_next_attempt_at timestamp with time zone,
  sale_listing_source_match_last_attempted_at timestamp with time zone,
  sale_listing_source_match_attempt_count integer default 0 not null,
  sale_listing_availability_text text,
  sale_listing_renovations_done_text text,
  sale_listing_renovations_planned_text text,
  sale_listing_additional_info_text text,
  sale_listing_charges_text text,
  sale_listing_maintenance_charge_monthly double precision,
  sale_listing_total_charge_monthly double precision,
  sale_listing_water_charge double precision,
  sale_listing_housing_company_name text,
  sale_listing_housing_company_business_id text,
  sale_listing_building_material text,
  sale_listing_heating_system text,
  sale_listing_roof_type text,
  sale_listing_roof_material text,
  sale_listing_apartment_count integer,
  sale_listing_car_storage_text text,
  sale_listing_building_description_text text,
  sale_listing_building_other_info_text text,
  sale_listing_latitude double precision,
  sale_listing_longitude double precision,
  sale_listing_living_area_value double precision,
  sale_listing_total_area_value double precision,
  sale_listing_other_area_value double precision,
  sale_listing_bedrooms_count integer,
  sale_listing_sauna boolean,
  sale_listing_balcony boolean,
  sale_listing_parking_text text,
  sale_listing_kitchen_description_text text,
  sale_listing_bathroom_description_text text,
  sale_listing_storage_description_text text,
  sale_listing_floor_materials_description_text text,
  sale_listing_wall_materials_description_text text,
  sale_listing_balcony_description_text text,
  sale_listing_sauna_description_text text,
  sale_listing_views_description_text text,
  sale_listing_appliances text[],
  sale_listing_features text[],
  sale_listing_plot_area_value double precision,
  sale_listing_services_text text,
  sale_listing_transport_text text,
  sale_listing_previous_asking_price bigint,
  sale_listing_previous_debt_free_price bigint,
  sale_listing_new_development boolean,
  constraint sale_listings_has_source_check CHECK (((shortcut_ad_id IS NOT NULL) OR (frontdoor_ad_id IS NOT NULL) OR (frontdoor_building_announcement_id IS NOT NULL))),
  constraint sale_listings_prices_match_status_check CHECK (((sale_listing_prices_match_status IS NULL) OR (sale_listing_prices_match_status = ANY (ARRAY['pending'::text, 'deferred'::text, 'auto_linked'::text, 'needs_review'::text, 'manual_linked'::text, 'rejected'::text, 'expired'::text, 'noop'::text])))),
  constraint sale_listings_source_kind_check CHECK ((sale_listing_source_kind = ANY (ARRAY['ad'::text, 'announcement'::text]))),
  constraint sale_listings_source_match_status_check CHECK (((sale_listing_source_match_status IS NULL) OR (sale_listing_source_match_status = ANY (ARRAY['pending'::text, 'deferred'::text, 'auto_linked'::text, 'needs_review'::text, 'manual_linked'::text, 'rejected'::text, 'noop'::text])))),
  constraint sale_listings_source_provider_check CHECK ((sale_listing_source_provider = ANY (ARRAY['shortcut'::text, 'frontdoor'::text])))
);

CREATE INDEX idx_property_source_offerings_street_name_number_ascii ON public.property_source_offerings USING btree (translate(sale_listing_street_name_norm, 'åäö'::text, 'aao'::text), sale_listing_street_number_norm, sale_listing_last_seen_at DESC);
CREATE INDEX idx_sale_listings_area ON public.property_source_offerings USING btree (sale_listing_area_value);
CREATE INDEX idx_sale_listings_build_year ON public.property_source_offerings USING btree (sale_listing_build_year);
CREATE INDEX idx_sale_listings_building_match_key ON public.property_source_offerings USING btree (sale_listing_building_match_key);
CREATE INDEX idx_sale_listings_city ON public.property_source_offerings USING btree (sale_listing_city);
CREATE INDEX idx_sale_listings_elevator ON public.property_source_offerings USING btree (sale_listing_elevator);
CREATE INDEX idx_sale_listings_energy_efficiency_class_year ON public.property_source_offerings USING btree (sale_listing_energy_efficiency_class_code, sale_listing_energy_efficiency_standard_year);
CREATE INDEX idx_sale_listings_energy_efficiency_match_code ON public.property_source_offerings USING btree (sale_listing_energy_efficiency_match_code);
CREATE INDEX idx_sale_listings_energy_efficiency_status ON public.property_source_offerings USING btree (sale_listing_energy_efficiency_status);
CREATE INDEX idx_sale_listings_first_seen ON public.property_source_offerings USING btree (sale_listing_first_seen_at);
CREATE INDEX idx_sale_listings_floor_level ON public.property_source_offerings USING btree (sale_listing_floor_level);
CREATE INDEX idx_sale_listings_last_seen ON public.property_source_offerings USING btree (sale_listing_last_seen_at DESC);
CREATE INDEX idx_sale_listings_plot_owned ON public.property_source_offerings USING btree (sale_listing_plot_owned);
CREATE INDEX idx_sale_listings_plot_type_code ON public.property_source_offerings USING btree (sale_listing_plot_type_code);
CREATE INDEX idx_sale_listings_postal ON public.property_source_offerings USING btree (sale_listing_postal);
CREATE INDEX idx_sale_listings_price ON public.property_source_offerings USING btree (sale_listing_asking_price);
CREATE INDEX idx_sale_listings_price_per_m2 ON public.property_source_offerings USING btree (sale_listing_price_per_m2);
CREATE INDEX idx_sale_listings_prices_match_last_seen ON public.property_source_offerings USING btree (sale_listing_last_seen_at) WHERE ((prices_transaction_id IS NULL) AND (sale_listing_source_kind = 'ad'::text));
CREATE INDEX idx_sale_listings_prices_match_queue ON public.property_source_offerings USING btree (sale_listing_prices_match_status, sale_listing_prices_match_next_attempt_at) WHERE (prices_transaction_id IS NULL);
CREATE INDEX idx_sale_listings_property_type_code ON public.property_source_offerings USING btree (sale_listing_property_type_code);
CREATE INDEX idx_sale_listings_room_category_code ON public.property_source_offerings USING btree (sale_listing_room_category_code);
CREATE INDEX idx_sale_listings_rooms_count ON public.property_source_offerings USING btree (sale_listing_rooms_count);
CREATE INDEX idx_sale_listings_search_trgm ON public.property_source_offerings USING gin (lower(sale_listing_search_text) gin_trgm_ops);
CREATE INDEX idx_sale_listings_source ON public.property_source_offerings USING btree (sale_listing_source_provider, sale_listing_source_kind);
CREATE INDEX idx_sale_listings_source_match_queue ON public.property_source_offerings USING btree (sale_listing_source_match_status, sale_listing_source_match_next_attempt_at) WHERE (sale_listing_source_kind = 'ad'::text);
CREATE INDEX idx_sale_listings_street_match_key ON public.property_source_offerings USING btree (sale_listing_street_match_key);
CREATE INDEX idx_sale_listings_unit_match_key ON public.property_source_offerings USING btree (sale_listing_unit_match_key);
CREATE UNIQUE INDEX sale_listings_frontdoor_ad_id_key ON public.property_source_offerings USING btree (frontdoor_ad_id) WHERE (frontdoor_ad_id IS NOT NULL);
CREATE UNIQUE INDEX sale_listings_frontdoor_building_announcement_id_key ON public.property_source_offerings USING btree (frontdoor_building_announcement_id) WHERE (frontdoor_building_announcement_id IS NOT NULL);
CREATE UNIQUE INDEX sale_listings_prices_transaction_id_key ON public.property_source_offerings USING btree (prices_transaction_id) WHERE (prices_transaction_id IS NOT NULL);
CREATE UNIQUE INDEX sale_listings_shortcut_ad_id_key ON public.property_source_offerings USING btree (shortcut_ad_id) WHERE (shortcut_ad_id IS NOT NULL);

create table public.property_units (
  property_unit_id uuid default gen_random_uuid() not null constraint property_units_pkey primary key,
  housing_company_id uuid not null constraint property_units_property_building_id_fkey references housing_companies(housing_company_id) ON DELETE CASCADE,
  property_unit_identity_key text not null constraint property_units_property_unit_identity_key_key unique,
  property_unit_address_norm text,
  property_unit_floor_level integer,
  property_unit_area_value double precision,
  property_unit_rooms_count integer,
  property_unit_room_layout text,
  property_unit_layout_match_key text,
  property_unit_match_reasons jsonb default '{}'::jsonb not null,
  property_unit_created_at timestamp with time zone default now() not null,
  property_unit_updated_at timestamp with time zone default now() not null,
  physical_building_id uuid constraint property_units_physical_building_id_fkey references physical_buildings(physical_building_id) ON DELETE SET NULL
);

CREATE INDEX idx_property_units_housing_company ON public.property_units USING btree (housing_company_id);
CREATE INDEX idx_property_units_physical_building ON public.property_units USING btree (physical_building_id);

create table public.sale_listing_prices_transaction_match_candidates (
  sale_listing_prices_transaction_match_candidate_id uuid default gen_random_uuid() not null constraint sale_listing_prices_transaction_match_candidates_pkey primary key,
  sale_listing_prices_transaction_match_run_id uuid not null constraint sale_listing_prices_transacti_sale_listing_prices_transact_fkey references sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE CASCADE,
  sale_listing_id uuid not null constraint sale_listing_prices_transaction_match_cand_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  prices_transaction_id uuid not null constraint sale_listing_prices_transaction_matc_prices_transaction_id_fkey references origin.prices_transactions(prices_transaction_id) ON DELETE CASCADE,
  sale_listing_prices_transaction_match_score integer not null,
  sale_listing_prices_transaction_match_confidence text not null,
  sale_listing_prices_transaction_match_status text default 'candidate'::text not null,
  sale_listing_prices_transaction_match_reasons jsonb default '{}'::jsonb not null,
  sale_listing_prices_transaction_match_price_delta_percent double precision,
  sale_listing_prices_transaction_match_created_at timestamp with time zone default now() not null,
  constraint sale_listing_prices_transaction_match_candidate_unique UNIQUE (sale_listing_prices_transaction_match_run_id, sale_listing_id, prices_transaction_id),
  constraint sale_listing_prices_transaction_match_confidence_check CHECK ((sale_listing_prices_transaction_match_confidence = ANY (ARRAY['high'::text, 'medium'::text, 'low'::text]))),
  constraint sale_listing_prices_transaction_match_status_check CHECK ((sale_listing_prices_transaction_match_status = ANY (ARRAY['candidate'::text, 'auto_linked'::text, 'ambiguous'::text, 'rejected'::text])))
);

CREATE INDEX idx_sale_listing_prices_transaction_match_candidates_listing_sc ON public.sale_listing_prices_transaction_match_candidates USING btree (sale_listing_id, sale_listing_prices_transaction_match_score DESC);
CREATE INDEX idx_sale_listing_prices_transaction_match_candidates_run_status ON public.sale_listing_prices_transaction_match_candidates USING btree (sale_listing_prices_transaction_match_run_id, sale_listing_prices_transaction_match_status);
CREATE INDEX idx_sale_listing_prices_transaction_match_candidates_transactio ON public.sale_listing_prices_transaction_match_candidates USING btree (prices_transaction_id, sale_listing_prices_transaction_match_score DESC);

create table public.sale_listing_prices_transaction_match_runs (
  sale_listing_prices_transaction_match_run_id uuid default gen_random_uuid() not null constraint sale_listing_prices_transaction_match_runs_pkey primary key,
  sale_listing_prices_transaction_match_run_mode text not null,
  sale_listing_prices_transaction_match_score_threshold integer default 90 not null,
  sale_listing_prices_transaction_match_competitor_margin integer default 15 not null,
  sale_listing_prices_transaction_match_candidates_count integer default 0 not null,
  sale_listing_prices_transaction_match_auto_linked_count integer default 0 not null,
  sale_listing_prices_transaction_match_ambiguous_count integer default 0 not null,
  sale_listing_prices_transaction_match_started_at timestamp with time zone default now() not null,
  sale_listing_prices_transaction_match_finished_at timestamp with time zone,
  constraint sale_listing_prices_transaction_match_margin_check CHECK ((sale_listing_prices_transaction_match_competitor_margin >= 0)),
  constraint sale_listing_prices_transaction_match_run_mode_check CHECK ((sale_listing_prices_transaction_match_run_mode = ANY (ARRAY['dry_run'::text, 'auto_link_safe'::text]))),
  constraint sale_listing_prices_transaction_match_threshold_check CHECK ((sale_listing_prices_transaction_match_score_threshold >= 0))
);

create table public.target_observations (
  target_observation_id uuid default gen_random_uuid() not null constraint target_observations_pkey primary key,
  target_type text not null,
  target_id uuid not null,
  observation_key text not null,
  observation_kind text not null,
  severity text not null,
  direction text not null,
  value jsonb,
  text text,
  confidence double precision not null,
  source_type text not null,
  source_id uuid not null,
  evidence jsonb default '{}'::jsonb not null,
  created_at timestamp with time zone default now() not null,
  superseded_at timestamp with time zone,
  constraint target_observations_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint target_observations_observation_kind_check CHECK ((observation_kind = ANY (ARRAY['risk'::text, 'opportunity'::text, 'inconsistency'::text, 'summary'::text, 'valuation_note'::text]))),
  constraint target_observations_source_type_check CHECK ((source_type = ANY (ARRAY['source_listing'::text, 'source_housing_company'::text, 'document'::text, 'price_transaction'::text, 'dimension_claim'::text, 'manual'::text]))),
  constraint target_observations_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);

CREATE INDEX idx_target_observations_source ON public.target_observations USING btree (source_type, source_id) WHERE (superseded_at IS NULL);
CREATE INDEX idx_target_observations_target ON public.target_observations USING btree (target_type, target_id, observation_kind, severity) WHERE (superseded_at IS NULL);
CREATE UNIQUE INDEX target_observations_active_unique ON public.target_observations USING btree (target_type, target_id, observation_key, source_type, source_id) WHERE (superseded_at IS NULL);

create table public.target_sources (
  target_source_id uuid default gen_random_uuid() not null constraint target_sources_pkey primary key,
  target_type text not null,
  target_id uuid not null,
  source_type text not null,
  source_id uuid not null,
  link_status text not null,
  link_method text not null,
  link_score integer default 0 not null,
  link_reasons jsonb default '{}'::jsonb not null,
  first_seen_at timestamp with time zone,
  last_seen_at timestamp with time zone,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null,
  constraint target_sources_link_method_check CHECK ((link_method = ANY (ARRAY['sync_auto'::text, 'source_match_auto'::text, 'document_match_auto'::text, 'manual'::text, 'backfill_auto'::text]))),
  constraint target_sources_link_status_check CHECK ((link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text, 'superseded'::text]))),
  constraint target_sources_source_type_check CHECK ((source_type = ANY (ARRAY['source_listing'::text, 'source_housing_company'::text, 'document'::text, 'price_transaction'::text, 'manual'::text]))),
  constraint target_sources_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'unit'::text, 'building'::text, 'housing_company'::text, 'house'::text])))
);

CREATE UNIQUE INDEX target_sources_active_source_listing ON public.target_sources USING btree (source_id) WHERE ((target_type = 'listing'::text) AND (source_type = 'source_listing'::text) AND (link_status <> 'rejected'::text));
CREATE INDEX target_sources_source ON public.target_sources USING btree (source_type, source_id, link_status);
CREATE INDEX target_sources_target ON public.target_sources USING btree (target_type, target_id, link_status);
CREATE UNIQUE INDEX target_sources_unique_target_source ON public.target_sources USING btree (target_type, target_id, source_type, source_id);

create table public.units (
  unit_id uuid not null constraint units_pkey primary key,
  housing_company_id uuid not null,
  physical_building_id uuid,
  identity_key text not null,
  address_norm text,
  apartment text,
  floor_level integer,
  area_m2 double precision,
  room_layout text,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_units_housing_company ON public.units USING btree (housing_company_id);
CREATE INDEX idx_units_physical_building ON public.units USING btree (physical_building_id) WHERE (physical_building_id IS NOT NULL);
CREATE UNIQUE INDEX units_identity_key_key ON public.units USING btree (identity_key);

create table runtime.kv_store (
  kv_key text not null constraint kv_store_pkey primary key,
  kv_value bytea not null,
  expires_at timestamp with time zone not null,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null
);

CREATE INDEX runtime_kv_store_expires_at_idx ON runtime.kv_store USING btree (expires_at);
