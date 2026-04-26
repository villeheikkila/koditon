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
  frontdoor_ad_search_text text,
  frontdoor_ad_description_text text,
  frontdoor_ad_availability_text text,
  frontdoor_ad_renovations_done_text text,
  frontdoor_ad_renovations_planned_text text,
  frontdoor_ad_additional_info_text text,
  frontdoor_ad_charges_text text,
  frontdoor_ad_maintenance_charge_monthly double precision,
  frontdoor_ad_total_charge_monthly double precision,
  frontdoor_ad_water_charge double precision,
  frontdoor_ad_debt_free_price bigint,
  frontdoor_ad_debt_share_amount bigint,
  frontdoor_ad_price_per_m2 double precision,
  frontdoor_ad_floor_level integer,
  frontdoor_ad_total_floors integer,
  frontdoor_ad_build_year integer,
  frontdoor_ad_condition text,
  frontdoor_ad_energy_class text,
  frontdoor_ad_plot_type text,
  frontdoor_ad_elevator boolean,
  frontdoor_ad_sauna boolean,
  frontdoor_ad_rooms_count integer
);

CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads USING btree (frontdoor_ad_page_not_found);
CREATE INDEX idx_frontdoor_ad_postal_postal_code_id ON public.frontdoor_ads USING btree (postal_postal_code_id);
CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads USING btree (frontdoor_ad_processed_at);
CREATE INDEX idx_frontdoor_ads_address_key ON public.frontdoor_ads USING btree (frontdoor_ad_address_key);
CREATE INDEX idx_frontdoor_ads_area_value ON public.frontdoor_ads USING btree (frontdoor_ad_area_value);
CREATE INDEX idx_frontdoor_ads_build_year ON public.frontdoor_ads USING btree (frontdoor_ad_build_year);
CREATE INDEX idx_frontdoor_ads_floor_level ON public.frontdoor_ads USING btree (frontdoor_ad_floor_level);
CREATE INDEX idx_frontdoor_ads_maintenance_charge ON public.frontdoor_ads USING btree (frontdoor_ad_maintenance_charge_monthly);
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

create table public.frontdoor_sync_tasks (
  frontdoor_sync_task_id bigint default nextval('frontdoor_sync_tasks_frontdoor_sync_task_id_seq'::regclass) not null constraint frontdoor_sync_tasks_pkey primary key,
  frontdoor_sync_task_entity_id text not null,
  frontdoor_sync_task_type text not null,
  frontdoor_sync_task_status text default 'pending'::text not null,
  frontdoor_sync_task_priority integer default 0 not null,
  frontdoor_sync_task_max_attempts integer default 3 not null,
  frontdoor_sync_task_attempts integer default 0 not null,
  frontdoor_sync_task_last_error text,
  frontdoor_sync_task_created_at timestamp with time zone default now() not null,
  frontdoor_sync_task_started_at timestamp with time zone,
  frontdoor_sync_task_completed_at timestamp with time zone,
  constraint frontdoor_sync_tasks_frontdoor_sync_task_entity_id_frontdoo_key UNIQUE (frontdoor_sync_task_entity_id, frontdoor_sync_task_type)
);

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

create table public.postal_sync_tasks (
  postal_sync_task_id bigint default nextval('postal_sync_tasks_postal_sync_task_id_seq'::regclass) not null constraint postal_sync_tasks_pkey primary key,
  postal_sync_task_entity_id text not null,
  postal_sync_task_type text not null,
  postal_sync_task_status text default 'pending'::text not null,
  postal_sync_task_priority integer default 0 not null,
  postal_sync_task_max_attempts integer default 3 not null,
  postal_sync_task_attempts integer default 0 not null,
  postal_sync_task_last_error text,
  postal_sync_task_created_at timestamp with time zone default now() not null,
  postal_sync_task_started_at timestamp with time zone,
  postal_sync_task_completed_at timestamp with time zone,
  constraint postal_sync_tasks_postal_sync_task_entity_id_postal_sync_ta_key UNIQUE (postal_sync_task_entity_id, postal_sync_task_type)
);

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

create table public.prices_sync_tasks (
  prices_sync_task_id bigint default nextval('prices_sync_tasks_prices_sync_task_id_seq'::regclass) not null constraint prices_sync_tasks_pkey primary key,
  prices_sync_task_entity_id text not null,
  prices_sync_task_type text not null,
  prices_sync_task_status text default 'pending'::text not null,
  prices_sync_task_priority integer default 0 not null,
  prices_sync_task_max_attempts integer default 3 not null,
  prices_sync_task_attempts integer default 0 not null,
  prices_sync_task_last_error text,
  prices_sync_task_created_at timestamp with time zone default now() not null,
  prices_sync_task_started_at timestamp with time zone,
  prices_sync_task_completed_at timestamp with time zone,
  constraint prices_sync_tasks_prices_sync_task_entity_id_prices_sync_ta_key UNIQUE (prices_sync_task_entity_id, prices_sync_task_type)
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
  shortcut_ad_search_text text,
  shortcut_ad_description_text text,
  shortcut_ad_availability_text text,
  shortcut_ad_renovations_done_text text,
  shortcut_ad_renovations_planned_text text,
  shortcut_ad_additional_info_text text,
  shortcut_ad_charges_text text,
  shortcut_ad_maintenance_charge_monthly double precision,
  shortcut_ad_total_charge_monthly double precision,
  shortcut_ad_water_charge double precision,
  shortcut_ad_debt_free_price bigint,
  shortcut_ad_debt_share_amount bigint,
  shortcut_ad_price_per_m2 double precision,
  shortcut_ad_floor_level integer,
  shortcut_ad_total_floors integer,
  shortcut_ad_build_year integer,
  shortcut_ad_condition text,
  shortcut_ad_energy_class text,
  shortcut_ad_plot_type text,
  shortcut_ad_elevator boolean,
  shortcut_ad_sauna boolean,
  shortcut_ad_rooms_count integer
);

CREATE INDEX idx_shortcut_ad_zipcode_name ON public.shortcut_ads USING btree (((((shortcut_ad_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));
CREATE INDEX idx_shortcut_ads_address_key ON public.shortcut_ads USING btree (shortcut_ad_address_key);
CREATE INDEX idx_shortcut_ads_area_value ON public.shortcut_ads USING btree (shortcut_ad_area_value);
CREATE INDEX idx_shortcut_ads_build_year ON public.shortcut_ads USING btree (shortcut_ad_build_year);
CREATE INDEX idx_shortcut_ads_floor_level ON public.shortcut_ads USING btree (shortcut_ad_floor_level);
CREATE INDEX idx_shortcut_ads_maintenance_charge ON public.shortcut_ads USING btree (shortcut_ad_maintenance_charge_monthly);
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

create table public.shortcut_sync_tasks (
  shortcut_sync_task_id bigint default nextval('shortcut_sync_tasks_shortcut_sync_task_id_seq'::regclass) not null constraint shortcut_sync_tasks_pkey primary key,
  shortcut_sync_task_entity_id text not null,
  shortcut_sync_task_type text not null,
  shortcut_sync_task_status text default 'pending'::text not null,
  shortcut_sync_task_priority integer default 0 not null,
  shortcut_sync_task_max_attempts integer default 3 not null,
  shortcut_sync_task_attempts integer default 0 not null,
  shortcut_sync_task_last_error text,
  shortcut_sync_task_created_at timestamp with time zone default now() not null,
  shortcut_sync_task_started_at timestamp with time zone,
  shortcut_sync_task_completed_at timestamp with time zone,
  constraint shortcut_sync_tasks_shortcut_sync_task_entity_id_shortcut_s_key UNIQUE (shortcut_sync_task_entity_id, shortcut_sync_task_type)
);

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


-- ============================================================
-- Auth Schema
-- ============================================================

CREATE TYPE public.enum__name_display AS ENUM (
    'full_name',
    'username'
);

CREATE TABLE public.users (
    user_uuid uuid NOT NULL,
    user_first_name text,
    user_last_name text,
    user_username text,
    user_name_display public.enum__name_display DEFAULT 'username'::public.enum__name_display,
    user_is_private boolean DEFAULT false NOT NULL,
    user_is_onboarded boolean DEFAULT false NOT NULL,
    user_joined_at timestamp with time zone DEFAULT now() NOT NULL,
    user_search text GENERATED ALWAYS AS (((user_username || COALESCE(user_first_name, ''::text)) || COALESCE(user_last_name, ''::text))) STORED,
    user_preferred_name text GENERATED ALWAYS AS (
CASE
    WHEN ((user_name_display = 'full_name'::public.enum__name_display) AND (user_first_name IS NOT NULL) AND (user_last_name IS NOT NULL)) THEN ((user_first_name || ' '::text) || user_last_name)
    ELSE user_username
END) STORED,
    user_id bigint NOT NULL,
    user_email text,
    user_has_seen_passkey_onboarding boolean DEFAULT false NOT NULL,
    CONSTRAINT users_pkey PRIMARY KEY (user_id),
    CONSTRAINT users_uuid_key UNIQUE (user_uuid)
);

CREATE TABLE public.user_identities (
    user_identity_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_identity_external_id text NOT NULL,
    user_identity_email text,
    user_identity_email_verified boolean DEFAULT false NOT NULL,
    user_identity_data jsonb DEFAULT '{}'::jsonb,
    user_identity_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_identity_provider text NOT NULL,
    user_identity_id bigint NOT NULL,
    user_id bigint NOT NULL,
    CONSTRAINT user_identities_pkey PRIMARY KEY (user_identity_id),
    CONSTRAINT user_identities_uuid_key UNIQUE (user_identity_uuid),
    CONSTRAINT user_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);

CREATE TABLE public.user_devices (
    user_device_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_device_name text,
    user_device_os text,
    user_device_app_version text,
    user_device_push_token text,
    user_device_push_token_updated_at timestamp with time zone,
    user_device_created_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_last_seen_at timestamp with time zone DEFAULT now() NOT NULL,
    user_device_push_token_type text,
    user_device_id bigint NOT NULL,
    user_id bigint NOT NULL,
    user_device_push_is_development boolean DEFAULT false NOT NULL,
    user_device_push_token_invalidated_at timestamp with time zone,
    user_device_push_token_invalidated_reason text,
    user_device_model text,
    user_device_locale text,
    user_device_time_zone text,
    CONSTRAINT user_devices_pkey PRIMARY KEY (user_device_id),
    CONSTRAINT user_devices_uuid_key UNIQUE (user_device_uuid),
    CONSTRAINT user_devices_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);

CREATE TABLE public.device_sessions (
    device_session_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    device_session_user_agent text,
    device_session_ip inet,
    device_session_created_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    device_session_refreshed_at timestamp with time zone,
    device_session_not_after timestamp with time zone,
    device_session_revoked_at timestamp with time zone,
    device_session_provider text NOT NULL,
    device_session_id bigint NOT NULL,
    device_session_user_device_id bigint NOT NULL,
    user_id bigint NOT NULL,
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
    CONSTRAINT device_sessions_pkey PRIMARY KEY (device_session_id),
    CONSTRAINT device_sessions_uuid_key UNIQUE (device_session_uuid),
    CONSTRAINT device_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT device_sessions_device_session_user_device_id_fkey FOREIGN KEY (device_session_user_device_id) REFERENCES public.user_devices(user_device_id) ON DELETE CASCADE
);

CREATE TABLE public.feature_flags (
    flag_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    flag_name text NOT NULL,
    flag_description text,
    flag_default_enabled boolean DEFAULT false NOT NULL,
    flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint NOT NULL,
    CONSTRAINT feature_flags_pkey PRIMARY KEY (flag_id),
    CONSTRAINT feature_flags_flag_name_key UNIQUE (flag_name)
);

CREATE TABLE public.roles (
    role_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    role_name text NOT NULL,
    role_description text,
    role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint NOT NULL,
    CONSTRAINT roles_pkey PRIMARY KEY (role_id),
    CONSTRAINT roles_role_name_key UNIQUE (role_name)
);

CREATE TABLE public.user_roles (
    user_role_created_at timestamp with time zone DEFAULT now() NOT NULL,
    role_id bigint NOT NULL,
    user_id bigint NOT NULL,
    CONSTRAINT user_roles_pkey PRIMARY KEY (user_id, role_id),
    CONSTRAINT user_roles_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT user_roles_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id) ON DELETE CASCADE
);

CREATE TABLE public.role_feature_flags (
    flag_id bigint NOT NULL,
    role_id bigint NOT NULL,
    CONSTRAINT role_feature_flags_pkey PRIMARY KEY (role_id, flag_id),
    CONSTRAINT role_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE,
    CONSTRAINT role_feature_flags_role_id_fkey FOREIGN KEY (role_id) REFERENCES public.roles(role_id) ON DELETE CASCADE
);

CREATE TABLE public.user_feature_flags (
    user_flag_enabled boolean NOT NULL,
    user_flag_created_at timestamp with time zone DEFAULT now() NOT NULL,
    flag_id bigint NOT NULL,
    user_id bigint NOT NULL,
    CONSTRAINT user_feature_flags_pkey PRIMARY KEY (user_id, flag_id),
    CONSTRAINT user_feature_flags_flag_id_fkey FOREIGN KEY (flag_id) REFERENCES public.feature_flags(flag_id) ON DELETE CASCADE,
    CONSTRAINT user_feature_flags_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);

CREATE TABLE public.personal_access_tokens (
    personal_access_token_id uuid DEFAULT gen_random_uuid() NOT NULL,
    personal_access_token_name text NOT NULL,
    personal_access_token_prefix text NOT NULL,
    personal_access_token_token_hash text NOT NULL,
    personal_access_token_scopes text[],
    personal_access_token_created_at timestamp with time zone DEFAULT now() NOT NULL,
    personal_access_token_last_used_at timestamp with time zone,
    personal_access_token_expires_at timestamp with time zone,
    personal_access_token_revoked_at timestamp with time zone,
    user_id bigint NOT NULL,
    CONSTRAINT personal_access_tokens_pkey PRIMARY KEY (personal_access_token_id),
    CONSTRAINT personal_access_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);

CREATE TABLE public.user_email_change_tokens (
    user_email_change_token_id bigint NOT NULL,
    user_email_change_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id bigint NOT NULL,
    user_email_change_target_email text NOT NULL,
    user_email_change_token_hash text NOT NULL,
    user_email_change_expires_at timestamp with time zone NOT NULL,
    user_email_change_consumed_at timestamp with time zone,
    user_email_change_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT user_email_change_tokens_pkey PRIMARY KEY (user_email_change_token_id),
    CONSTRAINT user_email_change_tokens_uuid_key UNIQUE (user_email_change_token_uuid),
    CONSTRAINT user_email_change_tokens_token_hash_key UNIQUE (user_email_change_token_hash),
    CONSTRAINT user_email_change_tokens_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);

CREATE TABLE public.user_passkeys (
    user_passkey_id bigint NOT NULL,
    user_passkey_uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    user_id bigint NOT NULL,
    user_identity_id bigint NOT NULL,
    user_passkey_credential_id bytea NOT NULL,
    user_passkey_credential_id_b64url text NOT NULL,
    user_passkey_public_key bytea NOT NULL,
    user_passkey_attestation_type text NOT NULL,
    user_passkey_transports text[] NOT NULL DEFAULT '{}',
    user_passkey_user_handle bytea NOT NULL,
    user_passkey_sign_count bigint NOT NULL DEFAULT 0,
    user_passkey_flags integer,
    user_passkey_aaguid uuid,
    user_passkey_name text,
    user_passkey_backup_eligible boolean,
    user_passkey_backup_state boolean,
    user_passkey_last_used_at timestamptz,
    user_passkey_created_at timestamptz NOT NULL DEFAULT now(),
    user_passkey_updated_at timestamptz NOT NULL DEFAULT now(),
    user_passkey_revoked_at timestamptz,
    CONSTRAINT user_passkeys_pkey PRIMARY KEY (user_passkey_id),
    CONSTRAINT user_passkeys_uuid_key UNIQUE (user_passkey_uuid),
    CONSTRAINT user_passkeys_credential_id_key UNIQUE (user_passkey_credential_id),
    CONSTRAINT user_passkeys_credential_id_b64url_key UNIQUE (user_passkey_credential_id_b64url),
    CONSTRAINT user_passkeys_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE,
    CONSTRAINT user_passkeys_user_identity_id_fkey FOREIGN KEY (user_identity_id) REFERENCES public.user_identities(user_identity_id) ON DELETE CASCADE
);

CREATE TABLE public.auth_webauthn_challenges (
    auth_webauthn_challenge_id bigint NOT NULL,
    auth_webauthn_challenge_uuid uuid NOT NULL DEFAULT gen_random_uuid(),
    auth_webauthn_challenge_flow text NOT NULL,
    auth_webauthn_challenge_session jsonb NOT NULL,
    auth_webauthn_challenge_expires_at timestamptz NOT NULL,
    auth_webauthn_challenge_user_handle bytea,
    auth_webauthn_challenge_user_display_name text,
    auth_webauthn_challenge_device_id uuid,
    auth_webauthn_challenge_consumed_at timestamptz,
    auth_webauthn_challenge_created_at timestamptz NOT NULL DEFAULT now(),
    auth_webauthn_challenge_verified_email text,
    user_id bigint,
    CONSTRAINT auth_webauthn_challenges_pkey PRIMARY KEY (auth_webauthn_challenge_id),
    CONSTRAINT auth_webauthn_challenges_uuid_key UNIQUE (auth_webauthn_challenge_uuid),
    CONSTRAINT auth_webauthn_challenges_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE
);

CREATE TABLE public.auth_signup_email_tokens (
    auth_signup_email_token_id bigint NOT NULL,
    auth_signup_email_token_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_signup_email_target_email text NOT NULL,
    auth_signup_email_token_hash text NOT NULL,
    auth_signup_email_expires_at timestamp with time zone NOT NULL,
    auth_signup_email_consumed_at timestamp with time zone,
    auth_signup_email_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_email_tokens_pkey PRIMARY KEY (auth_signup_email_token_id),
    CONSTRAINT auth_signup_email_tokens_uuid_key UNIQUE (auth_signup_email_token_uuid),
    CONSTRAINT auth_signup_email_tokens_token_hash_key UNIQUE (auth_signup_email_token_hash)
);

CREATE TABLE public.auth_signup_tickets (
    auth_signup_ticket_id bigint NOT NULL,
    auth_signup_ticket_uuid uuid DEFAULT gen_random_uuid() NOT NULL,
    auth_signup_ticket_target_email text NOT NULL,
    auth_signup_ticket_hash text NOT NULL,
    auth_signup_ticket_expires_at timestamp with time zone NOT NULL,
    auth_signup_ticket_consumed_at timestamp with time zone,
    auth_signup_ticket_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT auth_signup_tickets_pkey PRIMARY KEY (auth_signup_ticket_id),
    CONSTRAINT auth_signup_tickets_uuid_key UNIQUE (auth_signup_ticket_uuid),
    CONSTRAINT auth_signup_tickets_hash_key UNIQUE (auth_signup_ticket_hash)
);

CREATE TABLE public.oauth_authorization_codes (
    oauth_authorization_code_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_authorization_code_code_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL,
    oauth_authorization_code_redirect_uri text NOT NULL,
    oauth_authorization_code_scopes text[] NOT NULL DEFAULT '{}',
    oauth_authorization_code_code_challenge text NOT NULL,
    oauth_authorization_code_code_challenge_method text NOT NULL,
    oauth_authorization_code_audience text NOT NULL DEFAULT '',
    oauth_authorization_code_expires_at timestamptz NOT NULL,
    oauth_authorization_code_consumed_at timestamptz,
    oauth_authorization_code_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_authorization_code_updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_authorization_codes_user_uuid_fkey FOREIGN KEY (user_uuid) REFERENCES public.users(user_uuid) ON DELETE CASCADE
);

CREATE TABLE public.oauth_refresh_tokens (
    oauth_refresh_token_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_refresh_token_token_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    user_uuid uuid NOT NULL,
    oauth_refresh_token_scopes text[] NOT NULL DEFAULT '{}',
    oauth_refresh_token_audience text NOT NULL DEFAULT '',
    oauth_refresh_token_expires_at timestamptz NOT NULL,
    oauth_refresh_token_revoked_at timestamptz,
    oauth_refresh_token_rotated_from uuid,
    oauth_refresh_token_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_refresh_token_updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT oauth_refresh_tokens_user_uuid_fkey FOREIGN KEY (user_uuid) REFERENCES public.users(user_uuid) ON DELETE CASCADE
);

CREATE TABLE public.oauth_device_authorizations (
    oauth_device_authorization_id uuid DEFAULT gen_random_uuid() NOT NULL,
    oauth_device_authorization_device_code_hash text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    oauth_device_authorization_user_code text NOT NULL UNIQUE,
    oauth_device_authorization_scopes text[] DEFAULT '{}'::text[] NOT NULL,
    oauth_device_authorization_audience text NOT NULL DEFAULT '',
    user_uuid uuid,
    oauth_device_authorization_expires_at timestamp with time zone NOT NULL,
    oauth_device_authorization_approved_at timestamp with time zone,
    oauth_device_authorization_denied_at timestamp with time zone,
    oauth_device_authorization_consumed_at timestamp with time zone,
    oauth_device_authorization_created_at timestamp with time zone DEFAULT now() NOT NULL,
    oauth_device_authorization_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT oauth_device_authorizations_pkey PRIMARY KEY (oauth_device_authorization_id)
);

CREATE TABLE public.oauth_dynamic_clients (
    oauth_dynamic_client_id text PRIMARY KEY,
    oauth_dynamic_client_type text NOT NULL DEFAULT 'public',
    oauth_dynamic_client_redirect_uris text[] NOT NULL DEFAULT '{}',
    oauth_dynamic_client_scopes text[] NOT NULL DEFAULT '{}',
    oauth_dynamic_client_token_endpoint_auth_method text NOT NULL DEFAULT 'none',
    oauth_dynamic_client_name text,
    oauth_dynamic_client_metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    oauth_dynamic_client_issued_at timestamptz NOT NULL DEFAULT now(),
    oauth_dynamic_client_disabled_at timestamptz,
    oauth_dynamic_client_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_dynamic_client_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.oauth_authorization_handoffs (
    oauth_authorization_handoff_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    oauth_authorization_handoff_token_hash text NOT NULL UNIQUE,
    oauth_authorization_handoff_user_code text NOT NULL UNIQUE,
    oauth_client_id text NOT NULL,
    oauth_authorization_handoff_redirect_uri text NOT NULL,
    oauth_authorization_handoff_scopes text[] NOT NULL DEFAULT '{}',
    oauth_authorization_handoff_audience text NOT NULL DEFAULT '',
    oauth_authorization_handoff_state text NOT NULL DEFAULT '',
    oauth_authorization_handoff_code_challenge text NOT NULL,
    oauth_authorization_handoff_code_challenge_method text NOT NULL,
    user_uuid uuid,
    oauth_authorization_handoff_authorization_code text,
    oauth_authorization_handoff_redirect_url text,
    oauth_authorization_handoff_denied_at timestamptz,
    oauth_authorization_handoff_completed_at timestamptz,
    oauth_authorization_handoff_expires_at timestamptz NOT NULL,
    oauth_authorization_handoff_created_at timestamptz NOT NULL DEFAULT now(),
    oauth_authorization_handoff_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE SCHEMA IF NOT EXISTS runtime;

CREATE TABLE runtime.idempotency_keys (
    scope text NOT NULL,
    actor text NOT NULL,
    idempotency_key text NOT NULL,
    request_hash text NOT NULL,
    response_payload bytea,
    lock_expires_at timestamptz NOT NULL,
    result_expires_at timestamptz NOT NULL,
    completed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (scope, actor, idempotency_key)
);

CREATE TABLE runtime.kv_store (
    kv_key text NOT NULL,
    kv_value bytea NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);

ALTER TABLE ONLY runtime.kv_store
    ADD CONSTRAINT kv_store_pkey PRIMARY KEY (kv_key);

CREATE INDEX runtime_kv_store_expires_at_idx ON runtime.kv_store USING btree (expires_at);
