CREATE SCHEMA IF NOT EXISTS runtime;

create table public.auth_signup_email_tokens (
  auth_signup_email_token_id bigint generated always as identity not null constraint auth_signup_email_tokens_pkey primary key,
  auth_signup_email_token_uuid uuid default gen_random_uuid() not null constraint auth_signup_email_tokens_uuid_key unique,
  auth_signup_email_target_email text not null,
  auth_signup_email_token_hash text not null constraint auth_signup_email_tokens_token_hash_key unique,
  auth_signup_email_expires_at timestamp with time zone not null,
  auth_signup_email_consumed_at timestamp with time zone,
  auth_signup_email_created_at timestamp with time zone default now() not null,
  constraint auth_signup_email_tokens_target_email_not_blank CHECK ((btrim(auth_signup_email_target_email) <> ''::text))
);

CREATE INDEX idx_auth_signup_email_tokens_active_expires_at ON public.auth_signup_email_tokens USING btree (auth_signup_email_expires_at) WHERE (auth_signup_email_consumed_at IS NULL);
CREATE INDEX idx_auth_signup_email_tokens_target_email ON public.auth_signup_email_tokens USING btree (lower(btrim(auth_signup_email_target_email)));

create table public.auth_signup_tickets (
  auth_signup_ticket_id bigint generated always as identity not null constraint auth_signup_tickets_pkey primary key,
  auth_signup_ticket_uuid uuid default gen_random_uuid() not null constraint auth_signup_tickets_uuid_key unique,
  auth_signup_ticket_target_email text not null,
  auth_signup_ticket_hash text not null constraint auth_signup_tickets_hash_key unique,
  auth_signup_ticket_expires_at timestamp with time zone not null,
  auth_signup_ticket_consumed_at timestamp with time zone,
  auth_signup_ticket_created_at timestamp with time zone default now() not null,
  constraint auth_signup_tickets_target_email_not_blank CHECK ((btrim(auth_signup_ticket_target_email) <> ''::text))
);

CREATE INDEX idx_auth_signup_tickets_active_expires_at ON public.auth_signup_tickets USING btree (auth_signup_ticket_expires_at) WHERE (auth_signup_ticket_consumed_at IS NULL);
CREATE INDEX idx_auth_signup_tickets_target_email ON public.auth_signup_tickets USING btree (lower(btrim(auth_signup_ticket_target_email)));

create table public.auth_webauthn_challenges (
  auth_webauthn_challenge_id bigint generated always as identity not null constraint auth_webauthn_challenges_pkey primary key,
  auth_webauthn_challenge_uuid uuid default gen_random_uuid() not null constraint auth_webauthn_challenges_auth_webauthn_challenge_uuid_key unique,
  auth_webauthn_challenge_flow text not null,
  auth_webauthn_challenge_session jsonb not null,
  auth_webauthn_challenge_expires_at timestamp with time zone not null,
  auth_webauthn_challenge_user_handle bytea,
  auth_webauthn_challenge_user_display_name text,
  auth_webauthn_challenge_device_id uuid,
  auth_webauthn_challenge_consumed_at timestamp with time zone,
  auth_webauthn_challenge_created_at timestamp with time zone default now() not null,
  auth_webauthn_challenge_verified_email text,
  user_id bigint constraint auth_webauthn_challenges_user_id_fkey references users(user_id) ON DELETE CASCADE,
  constraint auth_webauthn_challenge_flow_check CHECK ((auth_webauthn_challenge_flow = ANY (ARRAY['authenticate'::text, 'register'::text, 'signup'::text])))
);

CREATE INDEX idx_auth_webauthn_challenges_active ON public.auth_webauthn_challenges USING btree (auth_webauthn_challenge_uuid, auth_webauthn_challenge_flow) WHERE (auth_webauthn_challenge_consumed_at IS NULL);
CREATE INDEX idx_auth_webauthn_challenges_expires ON public.auth_webauthn_challenges USING btree (auth_webauthn_challenge_expires_at);
CREATE INDEX idx_auth_webauthn_challenges_uuid ON public.auth_webauthn_challenges USING btree (auth_webauthn_challenge_uuid);

create table public.device_sessions (
  device_session_uuid uuid default gen_random_uuid() not null constraint device_sessions_uuid_key unique,
  device_session_user_agent text,
  device_session_ip inet,
  device_session_created_at timestamp with time zone default now() not null,
  device_session_updated_at timestamp with time zone default now() not null,
  device_session_refreshed_at timestamp with time zone,
  device_session_not_after timestamp with time zone,
  device_session_revoked_at timestamp with time zone,
  device_session_provider text not null,
  device_session_id bigint generated always as identity not null constraint device_sessions_pkey primary key,
  device_session_user_device_id bigint not null constraint device_sessions_device_session_user_device_id_fkey references user_devices(user_device_id) ON DELETE CASCADE,
  user_id bigint not null constraint device_sessions_user_id_fkey references users(user_id) ON DELETE CASCADE,
  device_session_device_name text,
  device_session_device_os text,
  device_session_device_model text,
  device_session_app_version text,
  device_session_locale text,
  device_session_time_zone text,
  device_session_location_city text,
  device_session_location_region text,
  device_session_location_country_code text,
  device_session_location_source text,
  constraint device_session_provider_check CHECK ((device_session_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);

CREATE INDEX idx_device_sessions_not_after ON public.device_sessions USING btree (device_session_not_after DESC);
CREATE INDEX idx_device_sessions_user_created ON public.device_sessions USING btree (user_id, device_session_created_at);
CREATE INDEX idx_device_sessions_user_device_id ON public.device_sessions USING btree (device_session_user_device_id);
CREATE INDEX idx_device_sessions_user_id ON public.device_sessions USING btree (user_id);

create table public.energy_efficiency_aliases (
  energy_efficiency_alias text not null constraint energy_efficiency_aliases_pkey primary key,
  energy_efficiency_class_code text,
  energy_efficiency_standard_year integer,
  energy_efficiency_status text not null,
  energy_efficiency_match_code text,
  energy_efficiency_label text not null,
  constraint energy_efficiency_aliases_status_check CHECK ((energy_efficiency_status = ANY (ARRAY['known'::text, 'not_required'::text, 'not_available'::text, 'unknown'::text])))
);

create table public.feature_flags (
  flag_uuid uuid default gen_random_uuid() not null constraint feature_flags_uuid_key unique,
  flag_name text not null constraint feature_flags_flag_name_key unique,
  flag_description text,
  flag_default_enabled boolean default false not null,
  flag_created_at timestamp with time zone default now() not null,
  flag_id bigint generated always as identity not null constraint feature_flags_pkey primary key
);

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
  frontdoor_ad_data_hash text,
  frontdoor_ad_data_hash_algorithm text default 'sha256'::text not null,
  frontdoor_ad_data_changed_at timestamp with time zone,
  frontdoor_ad_data_normalized_at timestamp with time zone,
  frontdoor_ad_data_normalized_version integer default 0 not null
);

CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads USING btree (frontdoor_ad_page_not_found);
CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads USING btree (frontdoor_ad_processed_at);
CREATE INDEX idx_frontdoor_ads_data_hash ON public.frontdoor_ads USING btree (frontdoor_ad_data_hash);
CREATE INDEX idx_frontdoor_ads_data_normalized ON public.frontdoor_ads USING btree (frontdoor_ad_data_normalized_at) WHERE (frontdoor_ad_data_hash IS NOT NULL);
CREATE INDEX idx_frontdoor_ads_data_normalized_version ON public.frontdoor_ads USING btree (frontdoor_ad_data_normalized_version) WHERE (frontdoor_ad_data_hash IS NOT NULL);

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
  frontdoor_building_announcement_unpublishing_time_date date,
  frontdoor_building_announcement_data_normalized_at timestamp with time zone,
  frontdoor_building_announcement_data_normalized_version integer default 0 not null
);

CREATE UNIQUE INDEX frontdoor_building_announcements_ext_id_unpub_time_price_key ON public.frontdoor_building_announcements USING btree (frontdoor_building_announcement_external_id, frontdoor_building_announcement_unpublishing_time, frontdoor_building_announcement_search_price);
CREATE INDEX idx_frontdoor_building_announcement_building_id ON public.frontdoor_building_announcements USING btree (frontdoor_building_id);
CREATE INDEX idx_frontdoor_building_announcements_normalized ON public.frontdoor_building_announcements USING btree (frontdoor_building_announcement_data_normalized_at, frontdoor_building_announcement_data_normalized_version);

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

create table public.housing_company_facts (
  housing_company_fact_id uuid default gen_random_uuid() not null constraint housing_company_facts_pkey primary key,
  housing_company_id uuid not null constraint housing_company_facts_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE CASCADE,
  housing_company_source_id uuid constraint housing_company_facts_housing_company_source_id_fkey references housing_company_sources(housing_company_source_id) ON DELETE SET NULL,
  housing_company_fact_key text not null,
  housing_company_fact_value_text text,
  housing_company_fact_value_number double precision,
  housing_company_fact_value_bool boolean,
  housing_company_fact_value_json jsonb,
  housing_company_fact_raw_value text,
  housing_company_fact_confidence integer default 100 not null,
  housing_company_fact_first_seen_at timestamp with time zone,
  housing_company_fact_last_seen_at timestamp with time zone,
  housing_company_fact_created_at timestamp with time zone default now() not null,
  housing_company_fact_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_housing_company_facts_company_key ON public.housing_company_facts USING btree (housing_company_id, housing_company_fact_key);
CREATE UNIQUE INDEX idx_housing_company_facts_unique_source_hash ON public.housing_company_facts USING btree (housing_company_id, housing_company_source_id, housing_company_fact_key, md5(COALESCE(housing_company_fact_raw_value, ''::text)));

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

create table public.housing_company_sources (
  housing_company_source_id uuid default gen_random_uuid() not null constraint housing_company_sources_pkey primary key,
  housing_company_id uuid not null constraint housing_company_sources_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE CASCADE,
  housing_company_source_provider text not null,
  housing_company_source_kind text not null,
  housing_company_source_table text not null,
  housing_company_source_id_value text not null,
  housing_company_source_external_id text,
  housing_company_source_url text,
  housing_company_source_link_status text default 'confirmed'::text not null,
  housing_company_source_link_method text not null,
  housing_company_source_link_score integer default 100 not null,
  housing_company_source_link_reasons jsonb default '{}'::jsonb not null,
  housing_company_source_first_seen_at timestamp with time zone,
  housing_company_source_last_seen_at timestamp with time zone,
  housing_company_source_created_at timestamp with time zone default now() not null,
  housing_company_source_updated_at timestamp with time zone default now() not null,
  constraint housing_company_sources_unique_source UNIQUE (housing_company_source_provider, housing_company_source_kind, housing_company_source_table, housing_company_source_id_value),
  constraint housing_company_sources_status_check CHECK ((housing_company_source_link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text])))
);

CREATE INDEX idx_housing_company_sources_company ON public.housing_company_sources USING btree (housing_company_id);

create table public.oauth_authorization_codes (
  oauth_authorization_code_id uuid default gen_random_uuid() not null constraint oauth_authorization_codes_pkey primary key,
  oauth_authorization_code_code_hash text not null constraint oauth_authorization_codes_oauth_authorization_code_code_has_key unique,
  oauth_client_id text not null,
  user_uuid uuid not null constraint oauth_authorization_codes_user_uuid_fkey references users(user_uuid) ON DELETE CASCADE,
  oauth_authorization_code_redirect_uri text not null,
  oauth_authorization_code_scopes text[] default '{}'::text[] not null,
  oauth_authorization_code_code_challenge text not null,
  oauth_authorization_code_code_challenge_method text not null,
  oauth_authorization_code_audience text default ''::text not null,
  oauth_authorization_code_expires_at timestamp with time zone not null,
  oauth_authorization_code_consumed_at timestamp with time zone,
  oauth_authorization_code_created_at timestamp with time zone default now() not null,
  oauth_authorization_code_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_oauth_authorization_codes_audience ON public.oauth_authorization_codes USING btree (oauth_authorization_code_audience);
CREATE INDEX idx_oauth_authorization_codes_client_id ON public.oauth_authorization_codes USING btree (oauth_client_id);
CREATE INDEX idx_oauth_authorization_codes_expires_at ON public.oauth_authorization_codes USING btree (oauth_authorization_code_expires_at);
CREATE INDEX idx_oauth_authorization_codes_user_uuid ON public.oauth_authorization_codes USING btree (user_uuid);

create table public.oauth_authorization_handoffs (
  oauth_authorization_handoff_id uuid default gen_random_uuid() not null constraint oauth_authorization_handoffs_pkey primary key,
  oauth_authorization_handoff_token_hash text not null constraint oauth_authorization_handoffs_oauth_authorization_handoff_to_key unique,
  oauth_authorization_handoff_user_code text not null constraint oauth_authorization_handoffs_oauth_authorization_handoff_us_key unique,
  oauth_client_id text not null,
  oauth_authorization_handoff_redirect_uri text not null,
  oauth_authorization_handoff_scopes text[] default '{}'::text[] not null,
  oauth_authorization_handoff_audience text default ''::text not null,
  oauth_authorization_handoff_state text default ''::text not null,
  oauth_authorization_handoff_code_challenge text not null,
  oauth_authorization_handoff_code_challenge_method text not null,
  user_uuid uuid constraint oauth_authorization_handoffs_user_uuid_fkey references users(user_uuid) ON DELETE SET NULL,
  oauth_authorization_handoff_authorization_code text,
  oauth_authorization_handoff_redirect_url text,
  oauth_authorization_handoff_denied_at timestamp with time zone,
  oauth_authorization_handoff_completed_at timestamp with time zone,
  oauth_authorization_handoff_expires_at timestamp with time zone not null,
  oauth_authorization_handoff_created_at timestamp with time zone default now() not null,
  oauth_authorization_handoff_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_oauth_authorization_handoffs_client_id ON public.oauth_authorization_handoffs USING btree (oauth_client_id);
CREATE INDEX idx_oauth_authorization_handoffs_expires_at ON public.oauth_authorization_handoffs USING btree (oauth_authorization_handoff_expires_at);
CREATE INDEX idx_oauth_authorization_handoffs_user_code ON public.oauth_authorization_handoffs USING btree (oauth_authorization_handoff_user_code);

create table public.oauth_device_authorizations (
  oauth_device_authorization_id uuid default gen_random_uuid() not null constraint oauth_device_authorizations_pkey primary key,
  oauth_device_authorization_device_code_hash text not null constraint oauth_device_authorizations_device_code_hash_key unique,
  oauth_client_id text not null,
  oauth_device_authorization_user_code text not null constraint oauth_device_authorizations_user_code_key unique,
  oauth_device_authorization_scopes text[] default '{}'::text[] not null,
  oauth_device_authorization_audience text default ''::text not null,
  user_uuid uuid constraint oauth_device_authorizations_user_uuid_fkey references users(user_uuid) ON DELETE SET NULL,
  oauth_device_authorization_expires_at timestamp with time zone not null,
  oauth_device_authorization_approved_at timestamp with time zone,
  oauth_device_authorization_denied_at timestamp with time zone,
  oauth_device_authorization_consumed_at timestamp with time zone,
  oauth_device_authorization_created_at timestamp with time zone default now() not null,
  oauth_device_authorization_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_oauth_device_authorizations_audience ON public.oauth_device_authorizations USING btree (oauth_device_authorization_audience);
CREATE INDEX idx_oauth_device_authorizations_client_id ON public.oauth_device_authorizations USING btree (oauth_client_id);
CREATE INDEX idx_oauth_device_authorizations_expires_at ON public.oauth_device_authorizations USING btree (oauth_device_authorization_expires_at);
CREATE INDEX idx_oauth_device_authorizations_user_code ON public.oauth_device_authorizations USING btree (oauth_device_authorization_user_code);

create table public.oauth_dynamic_clients (
  oauth_dynamic_client_id text not null constraint oauth_dynamic_clients_pkey primary key,
  oauth_dynamic_client_type text default 'public'::text not null,
  oauth_dynamic_client_redirect_uris text[] default '{}'::text[] not null,
  oauth_dynamic_client_scopes text[] default '{}'::text[] not null,
  oauth_dynamic_client_token_endpoint_auth_method text default 'none'::text not null,
  oauth_dynamic_client_name text,
  oauth_dynamic_client_metadata jsonb default '{}'::jsonb not null,
  oauth_dynamic_client_issued_at timestamp with time zone default now() not null,
  oauth_dynamic_client_disabled_at timestamp with time zone,
  oauth_dynamic_client_created_at timestamp with time zone default now() not null,
  oauth_dynamic_client_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_oauth_dynamic_clients_disabled_at ON public.oauth_dynamic_clients USING btree (oauth_dynamic_client_disabled_at);

create table public.oauth_refresh_tokens (
  oauth_refresh_token_id uuid default gen_random_uuid() not null constraint oauth_refresh_tokens_pkey primary key,
  oauth_refresh_token_token_hash text not null constraint oauth_refresh_tokens_oauth_refresh_token_token_hash_key unique,
  oauth_client_id text not null,
  user_uuid uuid not null constraint oauth_refresh_tokens_user_uuid_fkey references users(user_uuid) ON DELETE CASCADE,
  oauth_refresh_token_scopes text[] default '{}'::text[] not null,
  oauth_refresh_token_audience text default ''::text not null,
  oauth_refresh_token_expires_at timestamp with time zone not null,
  oauth_refresh_token_revoked_at timestamp with time zone,
  oauth_refresh_token_rotated_from uuid constraint oauth_refresh_tokens_oauth_refresh_token_rotated_from_fkey references oauth_refresh_tokens(oauth_refresh_token_id) ON DELETE SET NULL,
  oauth_refresh_token_created_at timestamp with time zone default now() not null,
  oauth_refresh_token_updated_at timestamp with time zone default now() not null,
  device_session_uuid uuid constraint oauth_refresh_tokens_device_session_uuid_fkey references device_sessions(device_session_uuid) ON DELETE SET NULL
);

CREATE INDEX idx_oauth_refresh_tokens_audience ON public.oauth_refresh_tokens USING btree (oauth_refresh_token_audience);
CREATE INDEX idx_oauth_refresh_tokens_client_id ON public.oauth_refresh_tokens USING btree (oauth_client_id);
CREATE INDEX idx_oauth_refresh_tokens_device_session_uuid ON public.oauth_refresh_tokens USING btree (device_session_uuid);
CREATE INDEX idx_oauth_refresh_tokens_expires_at ON public.oauth_refresh_tokens USING btree (oauth_refresh_token_expires_at);
CREATE INDEX idx_oauth_refresh_tokens_user_uuid ON public.oauth_refresh_tokens USING btree (user_uuid);

create table public.personal_access_tokens (
  personal_access_token_id uuid default gen_random_uuid() not null constraint personal_access_tokens_pkey primary key,
  personal_access_token_name text not null,
  personal_access_token_prefix text not null,
  personal_access_token_token_hash text not null,
  personal_access_token_scopes text[],
  personal_access_token_created_at timestamp with time zone default now() not null,
  personal_access_token_last_used_at timestamp with time zone,
  personal_access_token_expires_at timestamp with time zone,
  personal_access_token_revoked_at timestamp with time zone,
  user_id bigint not null constraint personal_access_tokens_user_id_fkey references users(user_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX idx_personal_access_tokens_prefix ON public.personal_access_tokens USING btree (personal_access_token_prefix);
CREATE INDEX idx_personal_access_tokens_user_id ON public.personal_access_tokens USING btree (user_id);

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
  prices_transaction_plot_owned boolean,
  constraint prices_transaction_unique_key UNIQUE NULLS NOT DISTINCT (prices_neighborhood_id, prices_transaction_description, prices_transaction_type, prices_transaction_area, prices_transaction_price, prices_transaction_price_per_square_meter, prices_transaction_build_year, prices_transaction_floor, prices_transaction_elevator, prices_transaction_condition, prices_transaction_plot, prices_transaction_energy_class, prices_transaction_category)
);

CREATE INDEX idx_prices_transaction_period_identifier ON public.prices_transactions USING btree (prices_transaction_period_identifier);
CREATE INDEX idx_prices_transactions_plot_owned ON public.prices_transactions USING btree (prices_transaction_plot_owned);
CREATE UNIQUE INDEX prices_transactions_unique_key ON public.prices_transactions USING btree (prices_neighborhood_id, prices_transaction_description, prices_transaction_type, prices_transaction_area, prices_transaction_price, prices_transaction_price_per_square_meter, prices_transaction_build_year, prices_transaction_floor, prices_transaction_elevator, prices_transaction_condition, prices_transaction_plot, prices_transaction_energy_class, prices_transaction_category) NULLS NOT DISTINCT;

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
  constraint property_dimension_catalog_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text]))),
  constraint property_dimension_catalog_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

create table public.property_dimension_claims (
  property_dimension_claim_id uuid default gen_random_uuid() not null constraint property_dimension_claims_pkey primary key,
  property_dimension_projection_run_id uuid not null constraint property_dimension_claims_property_dimension_projection_ru_fkey references property_dimension_projection_runs(property_dimension_projection_run_id) ON DELETE CASCADE,
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
  source_claim_id uuid constraint property_dimension_claims_source_claim_id_fkey references property_dimension_claims(property_dimension_claim_id) ON DELETE CASCADE,
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
  constraint property_dimension_claims_claim_scope_check CHECK ((claim_scope = ANY (ARRAY['source'::text, 'manual'::text]))),
  constraint property_dimension_claims_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint property_dimension_claims_source_reliability_check CHECK (((source_reliability >= (0)::double precision) AND (source_reliability <= (1)::double precision))),
  constraint property_dimension_claims_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'transaction'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text]))),
  constraint property_dimension_claims_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

CREATE INDEX idx_property_dimension_claims_dimension ON public.property_dimension_claims USING btree (dimension_key);
CREATE INDEX idx_property_dimension_claims_source ON public.property_dimension_claims USING btree (source_table, source_id, projection_version);
CREATE INDEX idx_property_dimension_claims_source_claim ON public.property_dimension_claims USING btree (source_claim_id);
CREATE INDEX idx_property_dimension_claims_target ON public.property_dimension_claims USING btree (claim_scope, target_type, target_id, dimension_key);
CREATE UNIQUE INDEX idx_property_dimension_claims_unique_source ON public.property_dimension_claims USING btree (claim_scope, target_type, target_id, dimension_key, source_table, source_id, COALESCE(source_field, ''::text), projection_version);
CREATE INDEX idx_property_dimension_claims_value_gin ON public.property_dimension_claims USING gin (value jsonb_path_ops);

create table public.property_dimension_dirty_targets (
  target_type text not null,
  target_id uuid not null,
  dirty_reasons text[] default '{}'::text[] not null,
  dirty_at timestamp with time zone default now() not null,
  queued_at timestamp with time zone,
  resolved_at timestamp with time zone,
  constraint property_dimension_dirty_targets_pkey PRIMARY KEY (target_type, target_id),
  constraint property_dimension_dirty_targets_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'transaction'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text])))
);

CREATE INDEX idx_property_dimension_dirty_targets_queue ON public.property_dimension_dirty_targets USING btree (dirty_at) WHERE ((resolved_at IS NULL) OR (resolved_at < dirty_at));

create table public.property_dimension_manual_overrides (
  property_dimension_manual_override_id uuid default gen_random_uuid() not null constraint property_dimension_manual_overrides_pkey primary key,
  target_type text not null,
  target_id uuid not null,
  dimension_key text not null,
  value jsonb not null,
  value_kind text not null,
  unit text,
  reason text not null,
  created_by text not null,
  valid_from date,
  valid_until date,
  created_at timestamp with time zone default now() not null,
  revoked_at timestamp with time zone,
  constraint property_dimension_manual_overrides_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text]))),
  constraint property_dimension_manual_overrides_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

CREATE UNIQUE INDEX idx_property_dimension_manual_overrides_active ON public.property_dimension_manual_overrides USING btree (target_type, target_id, dimension_key) WHERE (revoked_at IS NULL);

create table public.property_dimension_profiles (
  target_type text not null,
  target_id uuid not null,
  dimensions jsonb default '{}'::jsonb not null,
  metadata jsonb default '{}'::jsonb not null,
  conflicts jsonb default '{}'::jsonb not null,
  resolved_at timestamp with time zone default now() not null,
  constraint property_dimension_profiles_pkey PRIMARY KEY (target_type, target_id),
  constraint property_dimension_profiles_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text])))
);

CREATE INDEX idx_building_dimension_profiles_build_year ON public.property_dimension_profiles USING btree ((((dimensions #>> '{building,build_year}'::text[]))::integer)) WHERE (target_type = 'building'::text);
CREATE INDEX idx_property_dimension_profiles_dimensions_gin ON public.property_dimension_profiles USING gin (dimensions jsonb_path_ops);
CREATE INDEX idx_unit_dimension_profiles_area ON public.property_dimension_profiles USING btree ((((dimensions #>> '{unit,area_m2}'::text[]))::double precision)) WHERE (target_type = 'unit'::text);
CREATE INDEX idx_unit_dimension_profiles_total_charge ON public.property_dimension_profiles USING btree ((((dimensions #>> '{charges,total_monthly_eur}'::text[]))::double precision)) WHERE (target_type = 'unit'::text);

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

create table public.property_dimension_values (
  target_type text not null,
  target_id uuid not null,
  dimension_key text not null,
  value jsonb not null,
  value_kind text not null,
  unit text,
  confidence double precision not null,
  selected_claim_id uuid constraint property_dimension_values_selected_claim_id_fkey references property_dimension_claims(property_dimension_claim_id) ON DELETE CASCADE,
  selected_reason text not null,
  conflict_status text default 'none'::text not null,
  supporting_claim_ids uuid[] default '{}'::uuid[] not null,
  rejected_claim_ids uuid[] default '{}'::uuid[] not null,
  resolved_at timestamp with time zone default now() not null,
  constraint property_dimension_values_pkey PRIMARY KEY (target_type, target_id, dimension_key),
  constraint property_dimension_values_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint property_dimension_values_conflict_status_check CHECK ((conflict_status = ANY (ARRAY['none'::text, 'compatible'::text, 'conflicting'::text, 'manual_override'::text]))),
  constraint property_dimension_values_target_type_check CHECK ((target_type = ANY (ARRAY['offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text]))),
  constraint property_dimension_values_value_kind_check CHECK ((value_kind = ANY (ARRAY['string'::text, 'number'::text, 'boolean'::text, 'object'::text, 'array'::text, 'null'::text])))
);

CREATE INDEX idx_property_dimension_values_dimension ON public.property_dimension_values USING btree (dimension_key);
CREATE INDEX idx_property_dimension_values_selected_claim ON public.property_dimension_values USING btree (selected_claim_id);

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

create table public.property_offering_merge_decisions (
  property_offering_merge_decision_id uuid default gen_random_uuid() not null constraint property_offering_merge_decisions_pkey primary key,
  source_property_offering_id uuid not null constraint property_offering_merge_decisi_source_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  target_property_offering_id uuid not null constraint property_offering_merge_decisi_target_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  property_offering_source_match_candidate_id uuid constraint property_offering_merge_decis_property_offering_source_mat_fkey references property_offering_source_match_candidates(property_offering_source_match_candidate_id) ON DELETE SET NULL,
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

create table public.property_offering_source_match_candidates (
  property_offering_source_match_candidate_id uuid default gen_random_uuid() not null constraint property_offering_source_match_candidates_pkey primary key,
  property_offering_source_match_run_id uuid not null constraint property_offering_source_matc_property_offering_source_mat_fkey references property_offering_source_match_runs(property_offering_source_match_run_id) ON DELETE CASCADE,
  source_sale_listing_id uuid not null constraint property_offering_source_match_cand_source_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  source_property_offering_id uuid not null constraint property_offering_source_match_source_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  target_property_offering_id uuid not null constraint property_offering_source_match_target_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  target_sale_listing_id uuid not null constraint property_offering_source_match_cand_target_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  property_offering_source_match_score integer not null,
  property_offering_source_match_confidence text not null,
  property_offering_source_match_status text default 'candidate'::text not null,
  property_offering_source_match_reasons jsonb default '{}'::jsonb not null,
  property_offering_source_match_price_delta_percent double precision,
  property_offering_source_match_created_at timestamp with time zone default now() not null,
  constraint property_offering_source_match_candidate_unique UNIQUE (property_offering_source_match_run_id, source_sale_listing_id, target_property_offering_id),
  constraint property_offering_source_match_confidence_check CHECK ((property_offering_source_match_confidence = ANY (ARRAY['high'::text, 'medium'::text, 'low'::text]))),
  constraint property_offering_source_match_status_check CHECK ((property_offering_source_match_status = ANY (ARRAY['candidate'::text, 'auto_linked'::text, 'ambiguous'::text, 'rejected'::text])))
);

CREATE INDEX idx_property_offering_source_match_candidates_run_status ON public.property_offering_source_match_candidates USING btree (property_offering_source_match_run_id, property_offering_source_match_status);
CREATE INDEX idx_property_offering_source_match_candidates_source_score ON public.property_offering_source_match_candidates USING btree (source_sale_listing_id, property_offering_source_match_score DESC);
CREATE INDEX idx_property_offering_source_match_candidates_target_score ON public.property_offering_source_match_candidates USING btree (target_property_offering_id, property_offering_source_match_score DESC);

create table public.property_offering_source_match_runs (
  property_offering_source_match_run_id uuid default gen_random_uuid() not null constraint property_offering_source_match_runs_pkey primary key,
  property_offering_source_match_run_mode text not null,
  property_offering_source_match_score_threshold integer default 95 not null,
  property_offering_source_match_competitor_margin integer default 10 not null,
  property_offering_source_match_candidates_count integer default 0 not null,
  property_offering_source_match_auto_linked_count integer default 0 not null,
  property_offering_source_match_ambiguous_count integer default 0 not null,
  property_offering_source_match_started_at timestamp with time zone default now() not null,
  property_offering_source_match_finished_at timestamp with time zone,
  constraint property_offering_source_match_margin_check CHECK ((property_offering_source_match_competitor_margin >= 0)),
  constraint property_offering_source_match_run_mode_check CHECK ((property_offering_source_match_run_mode = ANY (ARRAY['dry_run'::text, 'auto_link_safe'::text]))),
  constraint property_offering_source_match_threshold_check CHECK ((property_offering_source_match_score_threshold >= 0))
);

create table public.property_offering_sources (
  property_offering_source_id uuid default gen_random_uuid() not null constraint property_offering_sources_pkey primary key,
  property_offering_id uuid not null constraint property_offering_sources_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  sale_listing_id uuid not null constraint property_offering_sources_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  property_offering_source_link_status text not null,
  property_offering_source_link_method text not null,
  property_offering_source_link_score integer not null,
  property_offering_source_link_reasons jsonb default '{}'::jsonb not null,
  property_offering_source_created_at timestamp with time zone default now() not null,
  property_offering_source_updated_at timestamp with time zone default now() not null,
  constraint property_offering_sources_method_check CHECK ((property_offering_source_link_method = ANY (ARRAY['backfill_auto'::text, 'sync_auto'::text, 'source_match_auto'::text, 'manual'::text]))),
  constraint property_offering_sources_status_check CHECK ((property_offering_source_link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text])))
);

CREATE UNIQUE INDEX idx_property_offering_sources_active_source ON public.property_offering_sources USING btree (sale_listing_id) WHERE (property_offering_source_link_status <> 'rejected'::text);
CREATE INDEX idx_property_offering_sources_offering ON public.property_offering_sources USING btree (property_offering_id);

create table public.property_offering_transactions (
  property_offering_transaction_id uuid default gen_random_uuid() not null constraint property_offering_transactions_pkey primary key,
  property_offering_id uuid not null constraint property_offering_transactions_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE CASCADE,
  prices_transaction_id uuid not null constraint property_offering_transactions_prices_transaction_id_fkey references prices_transactions(prices_transaction_id) ON DELETE CASCADE,
  property_offering_transaction_link_status text not null,
  property_offering_transaction_link_method text not null,
  property_offering_transaction_link_score integer not null,
  property_offering_transaction_link_reasons jsonb default '{}'::jsonb not null,
  property_offering_transaction_created_at timestamp with time zone default now() not null,
  property_offering_transaction_updated_at timestamp with time zone default now() not null,
  constraint property_offering_transactions_unique UNIQUE (property_offering_id, prices_transaction_id)
);

create table public.property_offerings (
  property_offering_id uuid default gen_random_uuid() not null constraint property_offerings_pkey primary key,
  property_unit_id uuid not null constraint property_offerings_property_unit_id_fkey references property_units(property_unit_id) ON DELETE CASCADE,
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
  constraint property_offerings_type_check CHECK ((property_offering_type = ANY (ARRAY['sale'::text])))
);

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
  constraint property_renovation_events_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint property_renovation_events_event_scope_check CHECK ((event_scope = ANY (ARRAY['source'::text, 'manual'::text]))),
  constraint property_renovation_events_source_reliability_check CHECK (((source_reliability >= (0)::double precision) AND (source_reliability <= (1)::double precision))),
  constraint property_renovation_events_target_type_check CHECK ((target_type = ANY (ARRAY['listing'::text, 'document'::text, 'offering'::text, 'unit'::text, 'building'::text, 'housing_company'::text])))
);

CREATE INDEX idx_property_renovation_events_source ON public.property_renovation_events USING btree (source_table, source_id, projection_version);
CREATE INDEX idx_property_renovation_events_source_event ON public.property_renovation_events USING btree (source_event_id);
CREATE INDEX idx_property_renovation_events_target ON public.property_renovation_events USING btree (event_scope, target_type, target_id, category, status);
CREATE UNIQUE INDEX idx_property_renovation_events_unique_source ON public.property_renovation_events USING btree (event_scope, target_type, target_id, source_table, source_id, COALESCE(source_field, ''::text), category, status, COALESCE(stage, ''::text), COALESCE(scope, ''::text), COALESCE(year, '-1'::integer), COALESCE(start_year, '-1'::integer), COALESCE(end_year, '-1'::integer), md5(COALESCE(summary, ''::text)), projection_version);

create table public.property_source_offering_insights (
  property_source_offering_insight_id uuid default gen_random_uuid() not null constraint property_source_offering_insights_pkey primary key,
  sale_listing_id uuid not null constraint property_source_offering_insights_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  property_source_offering_insight_source_field text not null,
  property_source_offering_insight_key text not null,
  property_source_offering_insight_value text not null,
  property_source_offering_insight_direction text not null,
  property_source_offering_insight_severity text not null,
  property_source_offering_insight_confidence integer default 50 not null,
  property_source_offering_insight_text text,
  property_source_offering_insight_created_at timestamp with time zone default now() not null,
  property_source_offering_insight_updated_at timestamp with time zone default now() not null
);

CREATE INDEX idx_property_source_offering_insights_listing ON public.property_source_offering_insights USING btree (sale_listing_id);
CREATE UNIQUE INDEX idx_property_source_offering_insights_unique ON public.property_source_offering_insights USING btree (sale_listing_id, property_source_offering_insight_source_field, property_source_offering_insight_key);

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
  shortcut_ad_id bigint constraint sale_listings_shortcut_ad_id_fkey references shortcut_ads(shortcut_ad_id) ON DELETE SET NULL,
  frontdoor_ad_id uuid constraint sale_listings_frontdoor_ad_id_fkey references frontdoor_ads(frontdoor_ad_id) ON DELETE SET NULL,
  frontdoor_building_announcement_id uuid constraint sale_listings_frontdoor_building_announcement_id_fkey references frontdoor_building_announcements(frontdoor_building_announcement_id) ON DELETE SET NULL,
  prices_transaction_id uuid constraint sale_listings_prices_transaction_id_fkey references prices_transactions(prices_transaction_id) ON DELETE SET NULL,
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
  sale_listing_source_match_run_id uuid constraint sale_listings_sale_listing_source_match_run_id_fkey references property_offering_source_match_runs(property_offering_source_match_run_id) ON DELETE SET NULL,
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

create table public.property_system_profiles (
  target_type text not null,
  target_id uuid not null,
  system_type text not null,
  status text not null,
  last_renovated_year integer,
  next_expected_start_year integer,
  next_expected_end_year integer,
  stage text,
  scope text,
  responsibility text,
  cost_estimate_eur bigint,
  confidence double precision default 0.5 not null,
  selected_renovation_event_ids uuid[] default '{}'::uuid[] not null,
  metadata jsonb default '{}'::jsonb not null,
  updated_at timestamp with time zone default now() not null,
  constraint property_system_profiles_pkey PRIMARY KEY (target_type, target_id, system_type),
  constraint property_system_profiles_confidence_check CHECK (((confidence >= (0)::double precision) AND (confidence <= (1)::double precision))),
  constraint property_system_profiles_target_type_check CHECK ((target_type = ANY (ARRAY['unit'::text, 'building'::text, 'housing_company'::text])))
);

CREATE INDEX idx_property_system_profiles_target ON public.property_system_profiles USING btree (target_type, target_id);

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

create table public.property_valuation_runs (
  property_valuation_run_id uuid default gen_random_uuid() not null constraint property_valuation_runs_pkey primary key,
  property_offering_id uuid constraint property_valuation_runs_property_offering_id_fkey references property_offerings(property_offering_id) ON DELETE SET NULL,
  property_unit_id uuid constraint property_valuation_runs_property_unit_id_fkey references property_units(property_unit_id) ON DELETE SET NULL,
  housing_company_id uuid constraint property_valuation_runs_housing_company_id_fkey references housing_companies(housing_company_id) ON DELETE SET NULL,
  property_valuation_run_model_version text not null,
  property_valuation_run_market_value_low bigint,
  property_valuation_run_market_value_high bigint,
  property_valuation_run_risk_adjusted_value_low bigint,
  property_valuation_run_risk_adjusted_value_high bigint,
  property_valuation_run_recommended_offer_low bigint,
  property_valuation_run_recommended_offer_high bigint,
  property_valuation_run_verdict text not null,
  property_valuation_run_confidence text not null,
  property_valuation_run_reasons jsonb default '[]'::jsonb not null,
  property_valuation_run_missing_evidence text[] default ARRAY[]::text[] not null,
  property_valuation_run_created_at timestamp with time zone default now() not null
);

create table public.role_feature_flags (
  flag_id bigint not null constraint role_feature_flags_flag_id_fkey references feature_flags(flag_id) ON DELETE CASCADE,
  role_id bigint not null constraint role_feature_flags_role_id_fkey references roles(role_id) ON DELETE CASCADE,
  constraint role_feature_flags_pkey PRIMARY KEY (role_id, flag_id)
);

CREATE INDEX idx_role_feature_flags_flag_id ON public.role_feature_flags USING btree (flag_id);
CREATE INDEX idx_role_feature_flags_role_id ON public.role_feature_flags USING btree (role_id);

create table public.roles (
  role_uuid uuid default gen_random_uuid() not null constraint roles_uuid_key unique,
  role_name text not null constraint roles_role_name_key unique,
  role_description text,
  role_created_at timestamp with time zone default now() not null,
  role_id bigint generated always as identity not null constraint roles_pkey primary key
);

create table public.sale_listing_plot_type_aliases (
  sale_listing_plot_type_alias text not null constraint sale_listing_plot_type_aliases_pkey primary key,
  sale_listing_plot_type_code text not null,
  sale_listing_plot_type_label text not null
);

create table public.sale_listing_prices_transaction_match_candidates (
  sale_listing_prices_transaction_match_candidate_id uuid default gen_random_uuid() not null constraint sale_listing_prices_transaction_match_candidates_pkey primary key,
  sale_listing_prices_transaction_match_run_id uuid not null constraint sale_listing_prices_transacti_sale_listing_prices_transact_fkey references sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE CASCADE,
  sale_listing_id uuid not null constraint sale_listing_prices_transaction_match_cand_sale_listing_id_fkey references property_source_offerings(sale_listing_id) ON DELETE CASCADE,
  prices_transaction_id uuid not null constraint sale_listing_prices_transaction_matc_prices_transaction_id_fkey references prices_transactions(prices_transaction_id) ON DELETE CASCADE,
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

create table public.sale_listing_property_type_aliases (
  sale_listing_property_type_alias text not null constraint sale_listing_property_type_aliases_pkey primary key,
  sale_listing_property_type_code text not null,
  sale_listing_property_type_label text not null
);

create table public.sale_listing_room_category_aliases (
  sale_listing_room_category_alias text not null constraint sale_listing_room_category_aliases_pkey primary key,
  sale_listing_room_category_code text not null,
  sale_listing_room_category_label text not null
);

create table public.schema_migrations (
  version integer not null constraint schema_migrations_pkey primary key
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
  shortcut_ad_data_schema_version smallint default 1 not null,
  shortcut_ad_data_hash text,
  shortcut_ad_data_hash_algorithm text default 'sha256'::text not null,
  shortcut_ad_data_changed_at timestamp with time zone,
  shortcut_ad_data_normalized_at timestamp with time zone,
  shortcut_ad_data_normalized_version integer default 0 not null
);

CREATE INDEX idx_shortcut_ads_data_hash ON public.shortcut_ads USING btree (shortcut_ad_data_hash);
CREATE INDEX idx_shortcut_ads_data_normalized ON public.shortcut_ads USING btree (shortcut_ad_data_normalized_at) WHERE (shortcut_ad_data_hash IS NOT NULL);
CREATE INDEX idx_shortcut_ads_data_normalized_version ON public.shortcut_ads USING btree (shortcut_ad_data_normalized_version) WHERE (shortcut_ad_data_hash IS NOT NULL);

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

create table public.sync_job_attempts (
  sync_job_attempt_id bigint generated always as identity not null constraint sync_job_attempts_pkey primary key,
  sync_job_id uuid not null constraint sync_job_attempts_sync_job_id_fkey references sync_jobs(sync_job_id) ON DELETE CASCADE,
  sync_job_attempt_queue_name text not null,
  sync_job_attempt_msg_id bigint,
  sync_job_attempt_no integer not null,
  sync_job_attempt_status text not null,
  sync_job_attempt_error_code text,
  sync_job_attempt_error_detail text,
  sync_job_attempt_payload_snapshot jsonb,
  sync_job_attempt_created_at timestamp with time zone default now() not null,
  sync_job_attempt_finished_at timestamp with time zone,
  constraint sync_job_attempts_payload_snapshot_object_check CHECK (((sync_job_attempt_payload_snapshot IS NULL) OR (jsonb_typeof(sync_job_attempt_payload_snapshot) = 'object'::text))),
  constraint sync_job_attempts_status_check CHECK ((sync_job_attempt_status = ANY (ARRAY['running'::text, 'succeeded'::text, 'failed'::text, 'retry'::text, 'not_found'::text, 'noop'::text, 'skipped_lock'::text])))
);

CREATE INDEX idx_sync_job_attempts_job_created_at_desc ON public.sync_job_attempts USING btree (sync_job_id, sync_job_attempt_created_at DESC);

create table public.sync_jobs (
  sync_job_id uuid default uuid_generate_v4() not null constraint sync_jobs_pkey primary key,
  sync_job_provider text not null,
  sync_job_kind text not null,
  sync_job_entity_id text not null,
  sync_job_dedup_key text not null constraint sync_jobs_sync_job_dedup_key_key unique,
  sync_job_status text default 'pending'::text not null,
  sync_job_priority integer default 0 not null,
  sync_job_attempt_count integer default 0 not null,
  sync_job_max_attempts integer default 3 not null,
  sync_job_run_after timestamp with time zone default now() not null,
  sync_job_capacity_class text default 'default'::text not null,
  sync_job_payload jsonb default '{}'::jsonb not null,
  sync_job_checkpoint jsonb,
  sync_job_result jsonb,
  sync_job_last_error text,
  sync_job_last_error_code text,
  sync_job_last_http_status integer,
  sync_job_last_pgmq_message_id bigint,
  sync_job_claim_token uuid,
  sync_job_created_at timestamp with time zone default now() not null,
  sync_job_updated_at timestamp with time zone default now() not null,
  sync_job_last_enqueued_at timestamp with time zone,
  sync_job_last_started_at timestamp with time zone,
  sync_job_last_finished_at timestamp with time zone,
  constraint sync_jobs_attempt_count_check CHECK ((sync_job_attempt_count >= 0)),
  constraint sync_jobs_checkpoint_object_check CHECK (((sync_job_checkpoint IS NULL) OR (jsonb_typeof(sync_job_checkpoint) = 'object'::text))),
  constraint sync_jobs_max_attempts_check CHECK ((sync_job_max_attempts >= 1)),
  constraint sync_jobs_payload_object_check CHECK ((jsonb_typeof(sync_job_payload) = 'object'::text)),
  constraint sync_jobs_result_object_check CHECK (((sync_job_result IS NULL) OR (jsonb_typeof(sync_job_result) = 'object'::text))),
  constraint sync_jobs_status_check CHECK ((sync_job_status = ANY (ARRAY['pending'::text, 'in_progress'::text, 'succeeded'::text, 'failed'::text, 'not_found'::text, 'noop'::text, 'skipped_lock'::text])))
);

CREATE INDEX idx_sync_jobs_capacity_status ON public.sync_jobs USING btree (sync_job_capacity_class, sync_job_status);
CREATE INDEX idx_sync_jobs_entity ON public.sync_jobs USING btree (sync_job_provider, sync_job_entity_id, sync_job_kind);
CREATE INDEX idx_sync_jobs_kind_status_run_after ON public.sync_jobs USING btree (sync_job_kind, sync_job_status, sync_job_run_after);
CREATE INDEX idx_sync_jobs_provider_status_run_after ON public.sync_jobs USING btree (sync_job_provider, sync_job_status, sync_job_run_after);

create table public.user_devices (
  user_device_uuid uuid default gen_random_uuid() not null constraint user_devices_uuid_key unique,
  user_device_name text,
  user_device_os text,
  user_device_app_version text,
  user_device_push_token text,
  user_device_push_token_updated_at timestamp with time zone,
  user_device_created_at timestamp with time zone default now() not null,
  user_device_updated_at timestamp with time zone default now() not null,
  user_device_last_seen_at timestamp with time zone default now() not null,
  user_device_push_token_type text,
  user_device_id bigint generated always as identity not null constraint user_devices_pkey primary key,
  user_id bigint not null constraint user_devices_user_id_fkey references users(user_id) ON DELETE CASCADE,
  user_device_push_is_development boolean default false not null,
  user_device_push_token_invalidated_at timestamp with time zone,
  user_device_push_token_invalidated_reason text,
  user_device_model text,
  user_device_locale text,
  user_device_time_zone text,
  constraint user_device_push_token_type_check CHECK (((user_device_push_token_type IS NULL) OR (user_device_push_token_type = 'apns'::text)))
);

CREATE INDEX idx_user_devices_push_token ON public.user_devices USING btree (user_device_push_token) WHERE (user_device_push_token IS NOT NULL);
CREATE INDEX idx_user_devices_user_id ON public.user_devices USING btree (user_id);

create table public.user_email_change_tokens (
  user_email_change_token_id bigint generated always as identity not null constraint user_email_change_tokens_pkey primary key,
  user_email_change_token_uuid uuid default gen_random_uuid() not null constraint user_email_change_tokens_uuid_key unique,
  user_id bigint not null constraint user_email_change_tokens_user_id_fkey references users(user_id) ON DELETE CASCADE,
  user_email_change_target_email text not null,
  user_email_change_token_hash text not null constraint user_email_change_tokens_token_hash_key unique,
  user_email_change_expires_at timestamp with time zone not null,
  user_email_change_consumed_at timestamp with time zone,
  user_email_change_created_at timestamp with time zone default now() not null,
  constraint user_email_change_tokens_target_email_not_blank CHECK ((btrim(user_email_change_target_email) <> ''::text))
);

CREATE INDEX idx_user_email_change_tokens_active_expires_at ON public.user_email_change_tokens USING btree (user_email_change_expires_at) WHERE (user_email_change_consumed_at IS NULL);
CREATE INDEX idx_user_email_change_tokens_user_id ON public.user_email_change_tokens USING btree (user_id);

create table public.user_feature_flags (
  user_flag_enabled boolean not null,
  user_flag_created_at timestamp with time zone default now() not null,
  flag_id bigint not null constraint user_feature_flags_flag_id_fkey references feature_flags(flag_id) ON DELETE CASCADE,
  user_id bigint not null constraint user_feature_flags_user_id_fkey references users(user_id) ON DELETE CASCADE,
  constraint user_feature_flags_pkey PRIMARY KEY (user_id, flag_id)
);

CREATE INDEX idx_user_feature_flags_flag_id ON public.user_feature_flags USING btree (flag_id);
CREATE INDEX idx_user_feature_flags_user_id ON public.user_feature_flags USING btree (user_id);

create table public.user_identities (
  user_identity_uuid uuid default gen_random_uuid() not null constraint user_identities_uuid_key unique,
  user_identity_external_id text not null,
  user_identity_email text,
  user_identity_email_verified boolean default false not null,
  user_identity_data jsonb default '{}'::jsonb,
  user_identity_created_at timestamp with time zone default now() not null,
  user_identity_updated_at timestamp with time zone default now() not null,
  user_identity_provider text not null,
  user_identity_id bigint generated always as identity not null constraint user_identities_pkey primary key,
  user_id bigint not null constraint user_identities_user_id_fkey references users(user_id) ON DELETE CASCADE,
  constraint user_identity_provider_check CHECK ((user_identity_provider = ANY (ARRAY['apple'::text, 'anonymous'::text, 'email'::text, 'google'::text, 'passkey'::text])))
);

CREATE UNIQUE INDEX idx_user_identities_provider_external_id_unique ON public.user_identities USING btree (user_identity_provider, user_identity_external_id);
CREATE INDEX idx_user_identities_user_id ON public.user_identities USING btree (user_id);

create table public.user_passkeys (
  user_passkey_id bigint generated always as identity not null constraint user_passkeys_pkey primary key,
  user_passkey_uuid uuid default gen_random_uuid() not null constraint user_passkeys_user_passkey_uuid_key unique,
  user_id bigint not null constraint user_passkeys_user_id_fkey references users(user_id) ON DELETE CASCADE,
  user_identity_id bigint not null constraint user_passkeys_user_identity_id_fkey references user_identities(user_identity_id) ON DELETE CASCADE,
  user_passkey_credential_id bytea not null constraint user_passkeys_credential_id_key unique,
  user_passkey_credential_id_b64url text not null constraint user_passkeys_credential_id_b64url_key unique,
  user_passkey_public_key bytea not null,
  user_passkey_attestation_type text not null,
  user_passkey_transports text[] default '{}'::text[] not null,
  user_passkey_user_handle bytea not null,
  user_passkey_sign_count bigint default 0 not null,
  user_passkey_flags integer,
  user_passkey_aaguid uuid,
  user_passkey_name text,
  user_passkey_backup_eligible boolean,
  user_passkey_backup_state boolean,
  user_passkey_last_used_at timestamp with time zone,
  user_passkey_created_at timestamp with time zone default now() not null,
  user_passkey_updated_at timestamp with time zone default now() not null,
  user_passkey_revoked_at timestamp with time zone
);

CREATE INDEX idx_user_passkeys_active_user_id ON public.user_passkeys USING btree (user_id) WHERE (user_passkey_revoked_at IS NULL);
CREATE INDEX idx_user_passkeys_user_handle ON public.user_passkeys USING btree (user_passkey_user_handle);
CREATE INDEX idx_user_passkeys_user_id ON public.user_passkeys USING btree (user_id);

create table public.user_roles (
  user_role_created_at timestamp with time zone default now() not null,
  role_id bigint not null constraint user_roles_role_id_fkey references roles(role_id) ON DELETE CASCADE,
  user_id bigint not null constraint user_roles_user_id_fkey references users(user_id) ON DELETE CASCADE,
  constraint user_roles_pkey PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_role_id ON public.user_roles USING btree (role_id);
CREATE INDEX idx_user_roles_user_id ON public.user_roles USING btree (user_id);

create table public.users (
  user_uuid uuid default gen_random_uuid() not null constraint users_uuid_key unique,
  user_first_name text,
  user_last_name text,
  user_username text constraint users_username_key unique,
  user_name_display enum__name_display default 'username'::enum__name_display,
  user_is_private boolean default false not null,
  user_is_onboarded boolean default false not null,
  user_joined_at timestamp with time zone default now() not null,
  user_search text generated always as (((user_username || COALESCE(user_first_name, ''::text)) || COALESCE(user_last_name, ''::text))) stored,
  user_preferred_name text generated always as ( CASE WHEN ((user_name_display = 'full_name'::enum__name_display) AND (user_first_name IS NOT NULL) AND (user_last_name IS NOT NULL)) THEN ((user_first_name || ' '::text) || user_last_name) ELSE user_username END) stored,
  user_id bigint generated always as identity not null constraint users_pkey primary key,
  user_email text,
  user_has_seen_passkey_onboarding boolean default false not null,
  constraint users_user_username_length CHECK (((user_username IS NULL) OR ((char_length(user_username) >= 2) AND (char_length(user_username) <= 16))))
);

CREATE INDEX idx_users_created_by_private ON public.users USING btree (user_uuid) WHERE (user_is_private = true);
CREATE UNIQUE INDEX idx_users_user_email_normalized_unique ON public.users USING btree (lower(btrim(user_email))) WHERE (user_email IS NOT NULL);
CREATE INDEX idx_users_username ON public.users USING btree (user_username);

create table runtime.kv_store (
  kv_key text not null constraint kv_store_pkey primary key,
  kv_value bytea not null,
  expires_at timestamp with time zone not null,
  created_at timestamp with time zone default now() not null,
  updated_at timestamp with time zone default now() not null
);

CREATE INDEX runtime_kv_store_expires_at_idx ON runtime.kv_store USING btree (expires_at);
