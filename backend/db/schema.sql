create table public.frontdoor_ads (
  frontdoor_ad_id uuid default gen_random_uuid() not null constraint frontdoor_ads_pkey primary key,
  frontdoor_ad_external_id text not null constraint frontdoor_ads_frontdoor_ads_external_id_key unique,
  frontdoor_ad_url text not null,
  frontdoor_ad_first_seen_at timestamp with time zone default now() not null,
  frontdoor_ad_last_seen_at timestamp with time zone default now() not null,
  frontdoor_ad_updated_at timestamp with time zone default now() not null,
  frontdoor_ad_data jsonb,
  frontdoor_ad_processed_at timestamp with time zone,
  frontdoor_ad_page_not_found boolean default false not null,
  frontdoor_ad_publishing_time timestamp with time zone,
  postal_postal_code_id uuid constraint frontdoor_ads_postal_postal_codes_id_fkey references postal_postal_codes(postal_postal_code_id),
  frontdoor_ad_address text generated always as ((frontdoor_ad_data #>> '{property,streetAddressFreeForm}'::text[])) stored,
  frontdoor_ad_area numeric generated always as (((frontdoor_ad_data #>> '{preparsed,area}'::text[]))::numeric) stored,
  frontdoor_ad_room_layout text generated always as ((frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}'::text[])) stored,
  frontdoor_ad_asking_price numeric generated always as (COALESCE(((frontdoor_ad_data #>> '{debfFreePrice}'::text[]))::numeric, ((frontdoor_ad_data #>> '{preparsed,price}'::text[]))::numeric)) stored,
  frontdoor_ad_street_address text,
  frontdoor_ad_city text,
  frontdoor_ad_postal text,
  frontdoor_ad_price bigint,
  frontdoor_ad_area_value float8,
  frontdoor_ad_address_key text,
  frontdoor_ad_search_text text
);

CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads USING btree (frontdoor_ad_page_not_found);
CREATE INDEX idx_frontdoor_ad_postal_postal_code_id ON public.frontdoor_ads USING btree (postal_postal_code_id);
CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads USING btree (frontdoor_ad_processed_at);

create table public.frontdoor_building_announcements (
  frontdoor_building_announcement_id uuid default gen_random_uuid() not null constraint frontdoor_building_announcements_pkey primary key,
  frontdoor_building_announcement_external_id integer,
  frontdoor_building_announcement_friendly_id text,
  frontdoor_building_announcement_unpublishing_time double precision,
  frontdoor_building_announcement_address_line1 text,
  frontdoor_building_announcement_address_line2 text,
  frontdoor_building_announcement_location text,
  frontdoor_building_announcement_search_price double precision,
  frontdoor_building_announcement_notify_price_changed boolean,
  frontdoor_building_announcement_property_type text,
  frontdoor_building_announcement_property_subtype text,
  frontdoor_building_announcement_construction_finished_year integer,
  frontdoor_building_announcement_main_image_uri text,
  frontdoor_building_announcement_has_open_bidding boolean,
  frontdoor_building_announcement_room_structure text,
  frontdoor_building_announcement_area double precision,
  frontdoor_building_announcement_total_area double precision,
  frontdoor_building_announcement_price_per_square double precision,
  frontdoor_building_announcement_days_on_market integer,
  frontdoor_building_announcement_new_building boolean,
  frontdoor_building_announcement_main_image_hidden boolean,
  frontdoor_building_announcement_is_company_announcement boolean,
  frontdoor_building_announcement_show_bidding_indicators boolean,
  frontdoor_building_announcement_published boolean,
  frontdoor_building_announcement_rent_period text,
  frontdoor_building_announcement_rental_unique_no integer,
  frontdoor_building_id uuid not null constraint frontdoor_building_announceme_frontdoor_building_announcem_fkey references frontdoor_buildings(frontdoor_building_id) ON DELETE CASCADE,
  frontdoor_building_announcement_first_seen_at timestamp with time zone default now() not null,
  frontdoor_building_announcement_last_seen_at timestamp with time zone default now() not null,
  frontdoor_building_announcement_unpublishing_time_date date
);

CREATE UNIQUE INDEX frontdoor_building_announcements_ext_id_unpub_time_price_key ON public.frontdoor_building_announcements USING btree (frontdoor_building_announcement_external_id, frontdoor_building_announcement_unpublishing_time, frontdoor_building_announcement_search_price);
CREATE INDEX idx_frontdoor_building_announcement_building_id ON public.frontdoor_building_announcements USING btree (frontdoor_building_id);

create table public.frontdoor_buildings (
  frontdoor_building_id uuid default gen_random_uuid() not null constraint frontdoor_buildings_pkey primary key,
  frontdoor_building_url text,
  frontdoor_building_first_seen_at timestamp with time zone default now() not null,
  frontdoor_building_last_seen_at timestamp with time zone default now() not null,
  frontdoor_building_updated_at timestamp with time zone default now() not null,
  frontdoor_building_company_name text,
  frontdoor_building_business_id text,
  frontdoor_building_apartment_count integer,
  frontdoor_building_floor_count integer,
  frontdoor_building_construction_end_year integer,
  frontdoor_building_build_year integer,
  frontdoor_building_has_elevator boolean,
  frontdoor_building_has_sauna boolean,
  frontdoor_building_energy_certificate_code text,
  frontdoor_building_plot_holding_type text,
  frontdoor_building_outer_roof_material text,
  frontdoor_building_outer_roof_type text,
  frontdoor_building_heating text,
  frontdoor_building_heating_fuel text[],
  frontdoor_building_street_address text,
  frontdoor_building_house_number text,
  frontdoor_building_postcode text,
  frontdoor_building_post_area text,
  frontdoor_building_municipality text,
  frontdoor_building_district text,
  frontdoor_building_latitude double precision,
  frontdoor_building_longitude double precision,
  frontdoor_building_elevator_renovated boolean,
  frontdoor_building_elevator_renovated_year integer,
  frontdoor_building_facade_renovated boolean,
  frontdoor_building_facade_renovated_year integer,
  frontdoor_building_window_renovated boolean,
  frontdoor_building_window_renovated_year integer,
  frontdoor_building_roof_renovated boolean,
  frontdoor_building_roof_renovated_year integer,
  frontdoor_building_pipe_renovated boolean,
  frontdoor_building_pipe_renovated_year integer,
  frontdoor_building_balcony_renovated boolean,
  frontdoor_building_balcony_renovated_year integer,
  frontdoor_building_electricity_renovated boolean,
  frontdoor_building_electricity_renovated_year integer,
  frontdoor_building_contact_phone text,
  frontdoor_building_contact_office_name text,
  frontdoor_building_contact_office_id integer,
  frontdoor_building_description text,
  frontdoor_building_car_storage_description text,
  frontdoor_building_other_info text,
  frontdoor_building_additional_addresses jsonb[],
  frontdoor_building_links jsonb[],
  frontdoor_building_data jsonb,
  frontdoor_building_processed_at timestamp with time zone,
  frontdoor_building_housing_company_id bigint constraint frontdoor_buildings_frontdoor_buildings_housing_company_id_key unique,
  frontdoor_building_housing_company_friendly_id text,
  frontdoor_building_geom postgis.geometry(Point,4326)
);

CREATE UNIQUE INDEX frontdoor_buildings_housing_company_friendly_id_unique ON public.frontdoor_buildings USING btree (frontdoor_building_housing_company_friendly_id) WHERE (frontdoor_building_housing_company_friendly_id IS NOT NULL);
CREATE UNIQUE INDEX frontdoor_buildings_url_unique ON public.frontdoor_buildings USING btree (frontdoor_building_url);
CREATE INDEX idx_frontdoor_building_business_id ON public.frontdoor_buildings USING btree (frontdoor_building_business_id);
CREATE INDEX idx_frontdoor_building_processed_at ON public.frontdoor_buildings USING btree (frontdoor_building_processed_at);

create table public.postal_ad_areas (
  postal_ad_area_id uuid default uuid_generate_v4() not null constraint postal_ad_areas_pkey primary key,
  postal_ad_area_code text not null constraint postal_ad_areas_postal_ad_areas_code_key unique,
  postal_ad_area_name_fi text not null,
  postal_ad_area_name_sv text,
  postal_ad_area_created_at timestamp with time zone default now() not null,
  postal_ad_area_updated_at timestamp with time zone default now() not null
);

create table public.postal_municipalities (
  postal_municipality_id uuid default uuid_generate_v4() not null constraint postal_municipalities_pkey primary key,
  postal_municipality_code text not null constraint postal_municipalities_postal_municipalities_code_key unique,
  postal_municipality_name_fi text not null,
  postal_municipality_name_sv text,
  postal_municipality_language_ratio_code text,
  postal_municipality_created_at timestamp with time zone default now() not null,
  postal_municipality_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_postal_municipality_name_fi ON public.postal_municipalities USING btree (postal_municipality_name_fi);

create table public.postal_postal_codes (
  postal_postal_code_id uuid default uuid_generate_v4() not null constraint postal_postal_codes_pkey primary key,
  postal_postal_code_date date not null,
  postal_postal_code_code text not null constraint postal_postal_codes_postal_postal_codes_code_key unique,
  postal_postal_code_name_fi text not null,
  postal_postal_code_name_sv text,
  postal_postal_code_abbr_fi text,
  postal_postal_code_abbr_sv text,
  postal_postal_code_valid_from date,
  postal_postal_code_type_code text,
  postal_ad_area_id uuid constraint postal_postal_codes_postal_postal_codes_ad_area_id_fkey references postal_ad_areas(postal_ad_area_id),
  postal_municipality_id uuid constraint postal_postal_codes_postal_postal_codes_municipality_id_fkey references postal_municipalities(postal_municipality_id),
  postal_postal_code_created_at timestamp with time zone default now() not null,
  postal_postal_code_updated_at timestamp with time zone default now() not null,
  postal_postal_code_neighborhood_fi text
);

CREATE INDEX idx_postal_postal_code_ad_area_id ON public.postal_postal_codes USING btree (postal_ad_area_id);
CREATE INDEX idx_postal_postal_code_municipality_id ON public.postal_postal_codes USING btree (postal_municipality_id);
CREATE INDEX idx_postal_postal_code_name_fi ON public.postal_postal_codes USING btree (postal_postal_code_name_fi);
CREATE INDEX idx_postal_postal_code_neighborhood_fi ON public.postal_postal_codes USING btree (postal_postal_code_neighborhood_fi);

create table public.prices_cities (
  prices_city_id uuid default uuid_generate_v4() not null constraint prices_cities_pkey primary key,
  prices_city_name text not null constraint prices_cities_prices_cities_name_key unique,
  prices_city_created_at timestamp with time zone default now() not null,
  prices_city_updated_at timestamp with time zone default now() not null
);

create table public.prices_neighborhoods (
  prices_neighborhood_id uuid default uuid_generate_v4() not null constraint prices_neighborhoods_pkey primary key,
  prices_neighborhood_name text not null,
  prices_city_id uuid not null constraint prices_neighborhoods_prices_neighborhoods_city_id_fkey references prices_cities(prices_city_id),
  prices_postal_code_id uuid constraint prices_neighborhoods_prices_neighborhoods_postal_code_id_fkey references prices_postal_codes(prices_postal_code_id),
  prices_neighborhood_created_at timestamp with time zone default now() not null,
  prices_neighborhood_updated_at timestamp with time zone default now() not null,
  prices_neighborhood_postal_postal_code_id uuid constraint prices_neighborhoods_prices_neighborhoods_posti_postal_cod_fkey references postal_postal_codes(postal_postal_code_id),
  constraint prices_neighborhoods_name_city_unique UNIQUE (prices_neighborhood_name, prices_city_id)
);

CREATE INDEX idx_prices_neighborhood_postal_postal_code_id ON public.prices_neighborhoods USING btree (prices_neighborhood_postal_postal_code_id);

create table public.prices_postal_codes (
  prices_postal_code_id uuid default uuid_generate_v4() not null constraint prices_postal_codes_pkey primary key,
  prices_postal_code_code text not null constraint prices_postal_codes_prices_postal_codes_code_key unique,
  prices_city_id uuid not null constraint prices_postal_codes_prices_postal_codes_city_id_fkey references prices_cities(prices_city_id),
  prices_postal_code_created_at timestamp with time zone default now() not null,
  prices_postal_code_updated_at timestamp with time zone default now() not null
);

create table public.prices_transactions (
  prices_transaction_id uuid default uuid_generate_v4() not null constraint prices_transactions_pkey primary key,
  prices_transaction_description text not null,
  prices_transaction_type text not null,
  prices_transaction_area double precision not null,
  prices_transaction_price integer not null,
  prices_transaction_price_per_square_meter integer not null,
  prices_transaction_build_year integer not null,
  prices_transaction_floor text,
  prices_transaction_elevator boolean not null,
  prices_transaction_condition text,
  prices_transaction_plot text,
  prices_transaction_energy_class text,
  prices_transaction_period_identifier text not null,
  prices_transaction_created_at timestamp with time zone default now() not null,
  prices_transaction_updated_at timestamp with time zone default now() not null,
  prices_transaction_category text not null,
  prices_neighborhood_id uuid constraint prices_transactions_prices_neighborhoods_id_fkey references prices_neighborhoods(prices_neighborhood_id),
  constraint prices_transaction_unique_key UNIQUE NULLS NOT DISTINCT (prices_neighborhood_id, prices_transaction_description, prices_transaction_type, prices_transaction_area, prices_transaction_price, prices_transaction_price_per_square_meter, prices_transaction_build_year, prices_transaction_floor, prices_transaction_elevator, prices_transaction_condition, prices_transaction_plot, prices_transaction_energy_class, prices_transaction_category)
);

CREATE INDEX idx_prices_transaction_period_identifier ON public.prices_transactions USING btree (prices_transaction_period_identifier);
CREATE UNIQUE INDEX prices_transactions_unique_key ON public.prices_transactions USING btree (prices_neighborhood_id, prices_transaction_description, prices_transaction_type, prices_transaction_area, prices_transaction_price, prices_transaction_price_per_square_meter, prices_transaction_build_year, prices_transaction_floor, prices_transaction_elevator, prices_transaction_condition, prices_transaction_plot, prices_transaction_energy_class, prices_transaction_category) NULLS NOT DISTINCT;

create table public.schema_migrations (
  version integer not null
);

create table public.shortcut_ads (
  shortcut_ad_id bigint not null constraint shortcut_ads_pkey primary key,
  shortcut_ad_url text not null,
  shortcut_ad_type text not null,
  shortcut_ad_first_seen_at timestamp with time zone default now() not null,
  shortcut_ad_last_seen_at timestamp with time zone default now() not null,
  shortcut_ad_data jsonb,
  shortcut_ad_updated_at timestamp with time zone default CURRENT_TIMESTAMP,
  shortcut_building_id uuid constraint shortcut_ads_shortcut_ads_building_id_fkey references shortcut_buildings(shortcut_building_id) ON DELETE SET NULL,
  shortcut_ad_address text generated always as ((shortcut_ad_data #>> '{address,formattedAddress}'::text[])) stored,
  shortcut_ad_area numeric generated always as (((shortcut_ad_data #>> '{adData,size}'::text[]))::numeric) stored,
  shortcut_ad_room_layout text generated always as ((shortcut_ad_data #>> '{adData,roomConfiguration}'::text[])) stored,
  shortcut_ad_asking_price numeric generated always as (COALESCE(((shortcut_ad_data #>> '{priceData,priceSell}'::text[]))::numeric, ((shortcut_ad_data #>> '{priceData,price}'::text[]))::numeric)) stored,
  shortcut_ad_street_address text,
  shortcut_ad_city text,
  shortcut_ad_postal text,
  shortcut_ad_price bigint,
  shortcut_ad_area_value float8,
  shortcut_ad_address_key text,
  shortcut_ad_search_text text
);

CREATE INDEX idx_shortcut_ad_zipcode_name ON public.shortcut_ads USING btree (((((shortcut_ad_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));

create table public.shortcut_building_listings (
  shortcut_building_listing_id uuid default gen_random_uuid() not null constraint shortcut_building_listings_pkey primary key,
  shortcut_building_id uuid not null constraint shortcut_building_listings_shortcut_building_listings_buil_fkey references shortcut_buildings(shortcut_building_id) ON DELETE CASCADE,
  shortcut_building_listing_layout text,
  shortcut_building_listing_size double precision,
  shortcut_building_listing_price double precision,
  shortcut_building_listing_price_per_sqm double precision,
  shortcut_building_listing_deleted_at timestamp with time zone,
  shortcut_building_listing_created_at timestamp with time zone default CURRENT_TIMESTAMP not null,
  shortcut_building_listing_updated_at timestamp with time zone default CURRENT_TIMESTAMP not null,
  shortcut_building_listing_marketing_time text,
  shortcut_building_listing_idx integer
);

CREATE UNIQUE INDEX shortcut_building_listings_unique_constraint ON public.shortcut_building_listings USING btree (shortcut_building_id, shortcut_building_listing_layout, shortcut_building_listing_size, shortcut_building_listing_price, shortcut_building_listing_price_per_sqm, shortcut_building_listing_deleted_at, shortcut_building_listing_marketing_time, shortcut_building_listing_idx);

create table public.shortcut_building_rentals (
  shortcut_building_rental_id uuid default gen_random_uuid() not null constraint shortcut_building_rentals_pkey primary key,
  shortcut_building_id uuid not null constraint shortcut_building_rentals_shortcut_building_rentals_buildi_fkey references shortcut_buildings(shortcut_building_id) ON DELETE CASCADE,
  shortcut_building_rental_layout text,
  shortcut_building_rental_size double precision,
  shortcut_building_rental_price double precision,
  shortcut_building_rental_deleted_at timestamp with time zone,
  shortcut_building_rental_created_at timestamp with time zone default CURRENT_TIMESTAMP not null,
  shortcut_building_rental_updated_at timestamp with time zone default CURRENT_TIMESTAMP not null,
  shortcut_building_rental_marketing_time text,
  shortcut_building_rental_idx integer
);

CREATE UNIQUE INDEX shortcut_building_rentals_unique_constraint ON public.shortcut_building_rentals USING btree (shortcut_building_id, shortcut_building_rental_layout, shortcut_building_rental_size, shortcut_building_rental_price, shortcut_building_rental_deleted_at, shortcut_building_rental_marketing_time, shortcut_building_rental_idx);

create table public.shortcut_buildings (
  shortcut_building_id uuid default gen_random_uuid() not null constraint shortcut_buildings_pkey primary key,
  shortcut_building_external_id bigint not null constraint shortcut_buildings_shortcut_buildings_external_id_key unique,
  shortcut_building_building_id text,
  shortcut_building_building_type text,
  shortcut_building_building_subtype text,
  shortcut_building_construction_year integer,
  shortcut_building_floor_count integer,
  shortcut_building_apartment_count integer,
  shortcut_building_heating_system text,
  shortcut_building_building_material text,
  shortcut_building_plot_type text,
  shortcut_building_wall_structure text,
  shortcut_building_heat_source text,
  shortcut_building_has_elevator text,
  shortcut_building_has_sauna text,
  shortcut_building_latitude double precision,
  shortcut_building_longitude double precision,
  shortcut_building_additional_addresses text,
  shortcut_building_url text not null,
  shortcut_building_created_at timestamp with time zone default CURRENT_TIMESTAMP not null,
  shortcut_building_updated_at timestamp with time zone default CURRENT_TIMESTAMP not null,
  shortcut_building_address text,
  shortcut_building_processed_at timestamp with time zone,
  shortcut_building_page_not_found boolean default false,
  shortcut_building_frame_construction_method text,
  shortcut_building_housing_company text,
  shortcut_building_geom postgis.geometry(Point,4326)
);

CREATE INDEX shortcut_building_geom_idx ON public.shortcut_buildings USING gist (shortcut_building_geom);

create table public.shortcut_tokens (
  shortcut_token_id uuid default gen_random_uuid() not null constraint shortcut_tokens_pkey primary key,
  shortcut_token_cuid text not null constraint shortcut_token_cuid_key unique,
  shortcut_token_token text not null,
  shortcut_token_loaded text not null,
  shortcut_token_created_at timestamp with time zone default now() not null,
  shortcut_token_updated_at timestamp with time zone default now() not null,
  shortcut_token_expires_at timestamp with time zone not null
);

CREATE INDEX idx_shortcut_token_cuid ON public.shortcut_tokens USING btree (shortcut_token_cuid);
CREATE INDEX idx_shortcut_token_expires_at ON public.shortcut_tokens USING btree (shortcut_token_expires_at DESC);

-- ============================================================================
-- task_queue schema
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS task_queue;

CREATE TABLE task_queue.entity_registry (
    entity_id TEXT NOT NULL PRIMARY KEY,
    entity_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'stopped')),
    scheduling_strategy TEXT NOT NULL DEFAULT 'manual'
        CHECK (scheduling_strategy IN ('daily', 'manual', 'on_demand', 'cron')),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entity_registry_status ON task_queue.entity_registry(status);
CREATE INDEX idx_entity_registry_entity_type ON task_queue.entity_registry(entity_type);
CREATE INDEX idx_entity_registry_scheduling_strategy ON task_queue.entity_registry(scheduling_strategy);
CREATE INDEX idx_entity_registry_schedulable ON task_queue.entity_registry(scheduling_strategy, status)
    WHERE status = 'active' AND scheduling_strategy = 'daily';

CREATE TABLE task_queue.task (
    task_id BIGSERIAL PRIMARY KEY,
    entity_id TEXT NOT NULL
        REFERENCES task_queue.entity_registry(entity_id) ON DELETE CASCADE,
    task_type TEXT NOT NULL DEFAULT 'frontdoor_sync',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'stopped')),
    priority INT NOT NULL DEFAULT 0,
    attempt INT NOT NULL DEFAULT 0,
    max_attempts INT NOT NULL DEFAULT 3,
    last_error TEXT,
    worker_id TEXT,
    scheduled_for TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    run_on DATE,
    queue_message_id BIGINT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uniq_task_daily
    ON task_queue.task(entity_id, task_type, run_on)
    WHERE run_on IS NOT NULL;

CREATE INDEX idx_task_entity ON task_queue.task(entity_id);
CREATE INDEX idx_task_status ON task_queue.task(status);
CREATE INDEX idx_task_worker ON task_queue.task(worker_id) WHERE status = 'processing';
CREATE INDEX idx_task_scheduled ON task_queue.task(scheduled_for) WHERE status = 'pending';
CREATE INDEX idx_task_priority_scheduled ON task_queue.task(priority DESC, scheduled_for ASC) WHERE status = 'pending';
CREATE INDEX idx_task_updated ON task_queue.task(updated_at);
CREATE INDEX idx_task_run_on ON task_queue.task(run_on);

CREATE TABLE task_queue.dead_letter_queue (
    dlq_id BIGSERIAL PRIMARY KEY,
    original_task_id BIGINT NOT NULL,
    entity_id TEXT NOT NULL,
    task_type TEXT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    total_attempts INT NOT NULL,
    first_error TEXT,
    last_error TEXT NOT NULL,
    error_history JSONB NOT NULL DEFAULT '[]'::jsonb,
    task_metadata JSONB DEFAULT '{}'::jsonb,
    original_created_at TIMESTAMPTZ NOT NULL,
    first_attempted_at TIMESTAMPTZ,
    last_attempted_at TIMESTAMPTZ NOT NULL,
    moved_to_dlq_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    requeued_at TIMESTAMPTZ,
    requeue_count INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_dlq_entity_id ON task_queue.dead_letter_queue(entity_id);
CREATE INDEX idx_dlq_task_type ON task_queue.dead_letter_queue(task_type);
CREATE INDEX idx_dlq_moved_at ON task_queue.dead_letter_queue(moved_to_dlq_at DESC);
CREATE INDEX idx_dlq_not_requeued ON task_queue.dead_letter_queue(moved_to_dlq_at DESC) WHERE requeued_at IS NULL;

CREATE OR REPLACE FUNCTION task_queue.fnc__register_entity(
    p_entity_id TEXT,
    p_entity_type TEXT,
    p_status TEXT DEFAULT 'active',
    p_scheduling_strategy TEXT DEFAULT 'manual',
    p_metadata JSONB DEFAULT '{}'::jsonb
) RETURNS void AS $$ BEGIN END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__register_entities(
    p_entity_ids TEXT[],
    p_entity_type TEXT,
    p_scheduling_strategy TEXT DEFAULT 'daily'
) RETURNS INT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__enqueue_task(
    p_task_id BIGINT
) RETURNS BIGINT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_daily_syncs(
    p_task_type TEXT DEFAULT 'frontdoor_sync'
) RETURNS INT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__requeue_stuck_tasks()
RETURNS INT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__move_to_dlq(
    p_task_id BIGINT,
    p_error_history JSONB DEFAULT '[]'::jsonb
) RETURNS BIGINT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION task_queue.fnc__requeue_from_dlq(
    p_dlq_id BIGINT,
    p_priority INT DEFAULT NULL,
    p_max_attempts INT DEFAULT 3
) RETURNS BIGINT AS $$ BEGIN RETURN 0; END; $$ LANGUAGE plpgsql;

-- ============================================================================
-- auth schema
-- ============================================================================

CREATE SCHEMA IF NOT EXISTS auth;

CREATE TYPE auth.auth_provider AS ENUM ('apple', 'anonymous');

CREATE TYPE auth.push_token_type AS ENUM ('apns');

CREATE TABLE auth.users (
    user_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_created_at timestamptz NOT NULL DEFAULT now(),
    user_updated_at timestamptz NOT NULL DEFAULT now(),
    user_deleted_at timestamptz
);

CREATE TABLE auth.identities (
    identity_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    identity_provider auth.auth_provider NOT NULL,
    identity_external_id text NOT NULL,
    identity_email text,
    identity_email_verified bool DEFAULT false,
    identity_data jsonb DEFAULT '{}'::jsonb,
    identity_created_at timestamptz NOT NULL DEFAULT now(),
    identity_updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT identities_provider_external_id_unique UNIQUE (identity_provider, identity_external_id)
);

CREATE TABLE auth.devices (
    device_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    device_name text,
    device_os text,
    device_app_version text,
    device_push_token text,
    device_push_token_type auth.push_token_type,
    device_push_token_updated_at timestamptz,
    device_created_at timestamptz NOT NULL DEFAULT now(),
    device_updated_at timestamptz NOT NULL DEFAULT now(),
    device_last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.sessions (
    session_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    session_device_id uuid REFERENCES auth.devices(device_id) ON DELETE SET NULL,
    session_user_agent text,
    session_ip inet,
    session_provider auth.auth_provider NOT NULL,
    session_refresh_token_hmac_key text NOT NULL,
    session_refresh_token_counter int8 NOT NULL DEFAULT 0,
    session_created_at timestamptz NOT NULL DEFAULT now(),
    session_updated_at timestamptz NOT NULL DEFAULT now(),
    session_refreshed_at timestamptz,
    session_not_after timestamptz,
    session_revoked_at timestamptz
);

CREATE TABLE auth.refresh_tokens (
    refresh_token_id bigserial PRIMARY KEY,
    session_id uuid NOT NULL REFERENCES auth.sessions(session_id) ON DELETE CASCADE,
    refresh_token_token_hash text NOT NULL,
    refresh_token_counter int8 NOT NULL,
    refresh_token_revoked bool NOT NULL DEFAULT false,
    refresh_token_created_at timestamptz NOT NULL DEFAULT now(),
    refresh_token_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.roles (
    role_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    role_name text NOT NULL UNIQUE,
    role_description text,
    role_created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.feature_flags (
    flag_id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    flag_name text NOT NULL UNIQUE,
    flag_description text,
    flag_default_enabled bool NOT NULL DEFAULT false,
    flag_created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE auth.role_feature_flags (
    role_id uuid NOT NULL REFERENCES auth.roles(role_id) ON DELETE CASCADE,
    flag_id uuid NOT NULL REFERENCES auth.feature_flags(flag_id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, flag_id)
);

CREATE TABLE auth.user_roles (
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    role_id uuid NOT NULL REFERENCES auth.roles(role_id) ON DELETE CASCADE,
    user_role_created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE auth.user_feature_flags (
    user_id uuid NOT NULL REFERENCES auth.users(user_id) ON DELETE CASCADE,
    flag_id uuid NOT NULL REFERENCES auth.feature_flags(flag_id) ON DELETE CASCADE,
    user_flag_enabled bool NOT NULL,
    user_flag_created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, flag_id)
);
