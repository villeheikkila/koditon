CREATE TABLE public.frontdoor_ads (
    frontdoor_ads_id uuid NOT NULL DEFAULT gen_random_uuid(),
    frontdoor_ads_external_id text NOT NULL,
    frontdoor_ads_url text NOT NULL,
    frontdoor_ads_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_updated_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_data jsonb,
    frontdoor_ads_processed_at timestamptz,
    frontdoor_ads_page_not_found bool NOT NULL DEFAULT false,
    frontdoor_ads_publishing_time timestamptz,
    PRIMARY KEY (frontdoor_ads_id)
);

CREATE UNIQUE INDEX frontdoor_ads_external_id_key ON public.frontdoor_ads USING btree (frontdoor_ads_external_id);
CREATE INDEX idx_frontdoor_ads_processed_at ON public.frontdoor_ads(frontdoor_ads_processed_at);
CREATE INDEX idx_frontdoor_ads_page_not_found ON public.frontdoor_ads(frontdoor_ads_page_not_found);

CREATE TABLE public.frontdoor_buildings (
    frontdoor_buildings_id uuid NOT NULL DEFAULT gen_random_uuid(),
    frontdoor_buildings_url text,
    frontdoor_buildings_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_updated_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_company_name text,
    frontdoor_buildings_business_id text,
    frontdoor_buildings_apartment_count int4,
    frontdoor_buildings_floor_count int4,
    frontdoor_buildings_construction_end_year int4,
    frontdoor_buildings_build_year int4,
    frontdoor_buildings_has_elevator bool,
    frontdoor_buildings_has_sauna bool,
    frontdoor_buildings_energy_certificate_code text,
    frontdoor_buildings_plot_holding_type text,
    frontdoor_buildings_outer_roof_material text,
    frontdoor_buildings_outer_roof_type text,
    frontdoor_buildings_heating text,
    frontdoor_buildings_heating_fuel text[],
    frontdoor_buildings_street_address text,
    frontdoor_buildings_house_number text,
    frontdoor_buildings_postcode text,
    frontdoor_buildings_post_area text,
    frontdoor_buildings_municipality text,
    frontdoor_buildings_district text,
    frontdoor_buildings_latitude float8,
    frontdoor_buildings_longitude float8,
    frontdoor_buildings_elevator_renovated bool,
    frontdoor_buildings_elevator_renovated_year int4,
    frontdoor_buildings_facade_renovated bool,
    frontdoor_buildings_facade_renovated_year int4,
    frontdoor_buildings_window_renovated bool,
    frontdoor_buildings_window_renovated_year int4,
    frontdoor_buildings_roof_renovated bool,
    frontdoor_buildings_roof_renovated_year int4,
    frontdoor_buildings_pipe_renovated bool,
    frontdoor_buildings_pipe_renovated_year int4,
    frontdoor_buildings_balcony_renovated bool,
    frontdoor_buildings_balcony_renovated_year int4,
    frontdoor_buildings_electricity_renovated bool,
    frontdoor_buildings_electricity_renovated_year int4,
    frontdoor_buildings_contact_phone text,
    frontdoor_buildings_contact_office_name text,
    frontdoor_buildings_contact_office_id int4,
    frontdoor_buildings_description text,
    frontdoor_buildings_car_storage_description text,
    frontdoor_buildings_other_info text,
    frontdoor_buildings_additional_addresses jsonb[],
    frontdoor_buildings_links jsonb[],
    frontdoor_buildings_data jsonb,
    frontdoor_buildings_processed_at timestamptz,
    frontdoor_buildings_housing_company_id int8,
    frontdoor_buildings_housing_company_friendly_id text,
    frontdoor_buildings_geom geometry(Point, 4326),
    PRIMARY KEY (frontdoor_buildings_id)
);

CREATE UNIQUE INDEX frontdoor_buildings_housing_company_id_unique ON public.frontdoor_buildings USING btree (frontdoor_buildings_housing_company_id);
CREATE INDEX idx_frontdoor_buildings_processed_at ON public.frontdoor_buildings(frontdoor_buildings_processed_at);
CREATE INDEX idx_frontdoor_buildings_business_id ON public.frontdoor_buildings(frontdoor_buildings_business_id);

-- frontdoor_building_announcements table
CREATE TABLE public.frontdoor_building_announcements (
    frontdoor_building_announcements_id uuid NOT NULL DEFAULT gen_random_uuid(),
    frontdoor_building_announcements_external_id int4,
    frontdoor_building_announcements_friendly_id text,
    frontdoor_building_announcements_unpublishing_time float8,
    frontdoor_building_announcements_address_line1 text,
    frontdoor_building_announcements_address_line2 text,
    frontdoor_building_announcements_location text,
    frontdoor_building_announcements_search_price float8,
    frontdoor_building_announcements_notify_price_changed bool,
    frontdoor_building_announcements_property_type text,
    frontdoor_building_announcements_property_subtype text,
    frontdoor_building_announcements_construction_finished_year int4,
    frontdoor_building_announcements_main_image_uri text,
    frontdoor_building_announcements_has_open_bidding bool,
    frontdoor_building_announcements_room_structure text,
    frontdoor_building_announcements_area float8,
    frontdoor_building_announcements_total_area float8,
    frontdoor_building_announcements_price_per_square float8,
    frontdoor_building_announcements_days_on_market int4,
    frontdoor_building_announcements_new_building bool,
    frontdoor_building_announcements_main_image_hidden bool,
    frontdoor_building_announcements_is_company_announcement bool,
    frontdoor_building_announcements_show_bidding_indicators bool,
    frontdoor_building_announcements_published bool,
    frontdoor_building_announcements_rent_period text,
    frontdoor_building_announcements_rental_unique_no int4,
    frontdoor_building_announcements_building_id uuid NOT NULL,
    frontdoor_building_announcements_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcements_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcements_unpublishing_time_date date,
    PRIMARY KEY (frontdoor_building_announcements_id),
    FOREIGN KEY (frontdoor_building_announcements_building_id) REFERENCES public.frontdoor_buildings(frontdoor_buildings_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX frontdoor_building_announcements_ext_id_unpub_time_price_key ON public.frontdoor_building_announcements USING btree (
    frontdoor_building_announcements_external_id,
    frontdoor_building_announcements_unpublishing_time,
    frontdoor_building_announcements_search_price
);
CREATE INDEX idx_frontdoor_building_announcements_building_id ON public.frontdoor_building_announcements(frontdoor_building_announcements_building_id);
