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
  frontdoor_ad_area_value double precision,
  frontdoor_ad_address_key text,
  frontdoor_ad_search_text text
);

CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads USING btree (frontdoor_ad_page_not_found);
CREATE INDEX idx_frontdoor_ad_postal_postal_code_id ON public.frontdoor_ads USING btree (postal_postal_code_id);
CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads USING btree (frontdoor_ad_processed_at);
CREATE INDEX idx_frontdoor_ads_address_key ON public.frontdoor_ads USING btree (frontdoor_ad_address_key);
CREATE INDEX idx_frontdoor_ads_area_value ON public.frontdoor_ads USING btree (frontdoor_ad_area_value);
CREATE INDEX idx_frontdoor_ads_postal ON public.frontdoor_ads USING btree (frontdoor_ad_postal);
CREATE INDEX idx_frontdoor_ads_price ON public.frontdoor_ads USING btree (frontdoor_ad_price);
CREATE INDEX idx_frontdoor_ads_search_trgm ON public.frontdoor_ads USING gin (lower(frontdoor_ad_search_text) gin_trgm_ops);
CREATE INDEX idx_frontdoor_ads_street_trgm ON public.frontdoor_ads USING gin (lower(frontdoor_ad_street_address) gin_trgm_ops);

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
  shortcut_ad_area_value double precision,
  shortcut_ad_address_key text,
  shortcut_ad_search_text text
);

CREATE INDEX idx_shortcut_ad_zipcode_name ON public.shortcut_ads USING btree (((((shortcut_ad_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));
CREATE INDEX idx_shortcut_ads_address_key ON public.shortcut_ads USING btree (shortcut_ad_address_key);
CREATE INDEX idx_shortcut_ads_area_value ON public.shortcut_ads USING btree (shortcut_ad_area_value);
CREATE INDEX idx_shortcut_ads_postal ON public.shortcut_ads USING btree (shortcut_ad_postal);
CREATE INDEX idx_shortcut_ads_price ON public.shortcut_ads USING btree (shortcut_ad_price);
CREATE INDEX idx_shortcut_ads_search_trgm ON public.shortcut_ads USING gin (lower(shortcut_ad_search_text) gin_trgm_ops);
CREATE INDEX idx_shortcut_ads_street_trgm ON public.shortcut_ads USING gin (lower(shortcut_ad_street_address) gin_trgm_ops);

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

create table public.frontdoor_pending_tasks (
  frontdoor_pending_task_id bigserial not null constraint frontdoor_pending_tasks_pkey primary key,
  frontdoor_pending_task_entity_id text not null,
  frontdoor_pending_task_type text not null,
  frontdoor_pending_task_status text not null default 'pending',
  frontdoor_pending_task_priority integer not null default 0,
  frontdoor_pending_task_max_attempts integer not null default 3,
  frontdoor_pending_task_attempts integer not null default 0,
  frontdoor_pending_task_last_error text,
  frontdoor_pending_task_created_at timestamp with time zone not null default now(),
  frontdoor_pending_task_started_at timestamp with time zone,
  frontdoor_pending_task_completed_at timestamp with time zone,
  constraint frontdoor_pending_tasks_entity_type_key unique (frontdoor_pending_task_entity_id, frontdoor_pending_task_type)
);

create table public.shortcut_pending_tasks (
  shortcut_pending_task_id bigserial not null constraint shortcut_pending_tasks_pkey primary key,
  shortcut_pending_task_entity_id text not null,
  shortcut_pending_task_type text not null,
  shortcut_pending_task_status text not null default 'pending',
  shortcut_pending_task_priority integer not null default 0,
  shortcut_pending_task_max_attempts integer not null default 3,
  shortcut_pending_task_attempts integer not null default 0,
  shortcut_pending_task_last_error text,
  shortcut_pending_task_created_at timestamp with time zone not null default now(),
  shortcut_pending_task_started_at timestamp with time zone,
  shortcut_pending_task_completed_at timestamp with time zone,
  constraint shortcut_pending_tasks_entity_type_key unique (shortcut_pending_task_entity_id, shortcut_pending_task_type)
);

create table public.prices_pending_tasks (
  prices_pending_task_id bigserial not null constraint prices_pending_tasks_pkey primary key,
  prices_pending_task_entity_id text not null,
  prices_pending_task_type text not null,
  prices_pending_task_status text not null default 'pending',
  prices_pending_task_priority integer not null default 0,
  prices_pending_task_max_attempts integer not null default 3,
  prices_pending_task_attempts integer not null default 0,
  prices_pending_task_last_error text,
  prices_pending_task_created_at timestamp with time zone not null default now(),
  prices_pending_task_started_at timestamp with time zone,
  prices_pending_task_completed_at timestamp with time zone,
  constraint prices_pending_tasks_entity_type_key unique (prices_pending_task_entity_id, prices_pending_task_type)
);

create table public.postal_pending_tasks (
  postal_pending_task_id bigserial not null constraint postal_pending_tasks_pkey primary key,
  postal_pending_task_entity_id text not null,
  postal_pending_task_type text not null,
  postal_pending_task_status text not null default 'pending',
  postal_pending_task_priority integer not null default 0,
  postal_pending_task_max_attempts integer not null default 3,
  postal_pending_task_attempts integer not null default 0,
  postal_pending_task_last_error text,
  postal_pending_task_created_at timestamp with time zone not null default now(),
  postal_pending_task_started_at timestamp with time zone,
  postal_pending_task_completed_at timestamp with time zone,
  constraint postal_pending_tasks_entity_type_key unique (postal_pending_task_entity_id, postal_pending_task_type)
);

