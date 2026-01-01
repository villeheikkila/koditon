CREATE TABLE public.frontdoor_ads (
    frontdoor_ad_id uuid NOT NULL DEFAULT gen_random_uuid(),
    frontdoor_ad_external_id text NOT NULL,
    frontdoor_ad_url text NOT NULL,
    frontdoor_ad_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ad_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ad_updated_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_ad_data jsonb,
    frontdoor_ad_processed_at timestamptz,
    frontdoor_ad_page_not_found bool NOT NULL DEFAULT false,
    frontdoor_ad_publishing_time timestamptz,
    PRIMARY KEY (frontdoor_ad_id)
);

CREATE UNIQUE INDEX frontdoor_ad_external_id_key ON public.frontdoor_ads USING btree (frontdoor_ad_external_id);
CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads(frontdoor_ad_processed_at);
CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads(frontdoor_ad_page_not_found);

CREATE TABLE public.frontdoor_buildings (
    frontdoor_building_id uuid NOT NULL DEFAULT gen_random_uuid(),
    frontdoor_building_url text,
    frontdoor_building_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_updated_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_company_name text,
    frontdoor_building_business_id text,
    frontdoor_building_apartment_count int4,
    frontdoor_building_floor_count int4,
    frontdoor_building_construction_end_year int4,
    frontdoor_building_build_year int4,
    frontdoor_building_has_elevator bool,
    frontdoor_building_has_sauna bool,
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
    frontdoor_building_latitude float8,
    frontdoor_building_longitude float8,
    frontdoor_building_elevator_renovated bool,
    frontdoor_building_elevator_renovated_year int4,
    frontdoor_building_facade_renovated bool,
    frontdoor_building_facade_renovated_year int4,
    frontdoor_building_window_renovated bool,
    frontdoor_building_window_renovated_year int4,
    frontdoor_building_roof_renovated bool,
    frontdoor_building_roof_renovated_year int4,
    frontdoor_building_pipe_renovated bool,
    frontdoor_building_pipe_renovated_year int4,
    frontdoor_building_balcony_renovated bool,
    frontdoor_building_balcony_renovated_year int4,
    frontdoor_building_electricity_renovated bool,
    frontdoor_building_electricity_renovated_year int4,
    frontdoor_building_contact_phone text,
    frontdoor_building_contact_office_name text,
    frontdoor_building_contact_office_id int4,
    frontdoor_building_description text,
    frontdoor_building_car_storage_description text,
    frontdoor_building_other_info text,
    frontdoor_building_additional_addresses jsonb[],
    frontdoor_building_links jsonb[],
    frontdoor_building_data jsonb,
    frontdoor_building_processed_at timestamptz,
    frontdoor_building_housing_company_id int8,
    frontdoor_building_housing_company_friendly_id text,
    frontdoor_building_geom geometry(Point, 4326),
    PRIMARY KEY (frontdoor_building_id)
);

CREATE UNIQUE INDEX frontdoor_building_housing_company_id_unique ON public.frontdoor_buildings USING btree (frontdoor_building_housing_company_id);
CREATE UNIQUE INDEX frontdoor_building_housing_company_friendly_id_unique ON public.frontdoor_buildings USING btree (frontdoor_building_housing_company_friendly_id) WHERE frontdoor_building_housing_company_friendly_id IS NOT NULL;
CREATE UNIQUE INDEX frontdoor_building_url_unique ON public.frontdoor_buildings USING btree (frontdoor_building_url);
CREATE INDEX idx_frontdoor_building_processed_at ON public.frontdoor_buildings(frontdoor_building_processed_at);
CREATE INDEX idx_frontdoor_building_business_id ON public.frontdoor_buildings(frontdoor_building_business_id);

-- frontdoor_building_announcements table
CREATE TABLE public.frontdoor_building_announcements (
    frontdoor_building_announcement_id uuid NOT NULL DEFAULT gen_random_uuid(),
    frontdoor_building_announcement_external_id int4,
    frontdoor_building_announcement_friendly_id text,
    frontdoor_building_announcement_unpublishing_time float8,
    frontdoor_building_announcement_address_line1 text,
    frontdoor_building_announcement_address_line2 text,
    frontdoor_building_announcement_location text,
    frontdoor_building_announcement_search_price float8,
    frontdoor_building_announcement_notify_price_changed bool,
    frontdoor_building_announcement_property_type text,
    frontdoor_building_announcement_property_subtype text,
    frontdoor_building_announcement_construction_finished_year int4,
    frontdoor_building_announcement_main_image_uri text,
    frontdoor_building_announcement_has_open_bidding bool,
    frontdoor_building_announcement_room_structure text,
    frontdoor_building_announcement_area float8,
    frontdoor_building_announcement_total_area float8,
    frontdoor_building_announcement_price_per_square float8,
    frontdoor_building_announcement_days_on_market int4,
    frontdoor_building_announcement_new_building bool,
    frontdoor_building_announcement_main_image_hidden bool,
    frontdoor_building_announcement_is_company_announcement bool,
    frontdoor_building_announcement_show_bidding_indicators bool,
    frontdoor_building_announcement_published bool,
    frontdoor_building_announcement_rent_period text,
    frontdoor_building_announcement_rental_unique_no int4,
    frontdoor_building_id uuid NOT NULL,
    frontdoor_building_announcement_first_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcement_last_seen_at timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcement_unpublishing_time_date date,
    PRIMARY KEY (frontdoor_building_announcement_id),
    FOREIGN KEY (frontdoor_building_id) REFERENCES public.frontdoor_buildings(frontdoor_building_id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX frontdoor_building_announcement_ext_id_unpub_time_price_key ON public.frontdoor_building_announcements USING btree (
    frontdoor_building_announcement_external_id,
    frontdoor_building_announcement_unpublishing_time,
    frontdoor_building_announcement_search_price
);
CREATE INDEX idx_frontdoor_building_id ON public.frontdoor_building_announcements(frontdoor_building_id);
