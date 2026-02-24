CREATE TABLE public.shortcut_buildings (
    shortcut_building_id uuid NOT NULL,
    shortcut_building_external_id int8 NOT NULL,
    shortcut_building_building_type text,
    shortcut_building_building_subtype text,
    shortcut_building_construction_year int4,
    shortcut_building_floor_count int4,
    shortcut_building_apartment_count int4,
    shortcut_building_heating_system text,
    shortcut_building_building_material text,
    shortcut_building_plot_type text,
    shortcut_building_wall_structure text,
    shortcut_building_heat_source text,
    shortcut_building_has_elevator text,
    shortcut_building_has_sauna text,
    shortcut_building_latitude float8,
    shortcut_building_longitude float8,
    shortcut_building_url text NOT NULL,
    shortcut_building_address text,
    shortcut_building_housing_company text,
    shortcut_building_updated_at timestamptz,
    shortcut_building_processed_at timestamptz,
    shortcut_building_page_not_found bool,
    PRIMARY KEY (shortcut_building_id)
);

CREATE TABLE public.shortcut_ads (
    shortcut_ad_id int8 NOT NULL,
    shortcut_ad_url text NOT NULL,
    shortcut_ad_type text NOT NULL,
    shortcut_ad_last_seen_at timestamptz NOT NULL,
    shortcut_ad_data jsonb,
    shortcut_ad_street_address text,
    shortcut_ad_city text,
    shortcut_ad_postal text,
    shortcut_ad_price int8,
    shortcut_ad_area_value float8,
    shortcut_ad_address_key text,
    shortcut_ad_search_text text,
    shortcut_building_id uuid,
    PRIMARY KEY (shortcut_ad_id)
);

CREATE TABLE public.shortcut_building_listings (
    shortcut_building_listing_id uuid NOT NULL,
    shortcut_building_id uuid NOT NULL,
    PRIMARY KEY (shortcut_building_listing_id)
);

CREATE TABLE public.shortcut_building_rentals (
    shortcut_building_rental_id uuid NOT NULL,
    shortcut_building_id uuid NOT NULL,
    PRIMARY KEY (shortcut_building_rental_id)
);

CREATE TABLE public.frontdoor_ads (
    frontdoor_ad_id uuid NOT NULL,
    frontdoor_ad_external_id text NOT NULL,
    frontdoor_ad_url text NOT NULL,
    frontdoor_ad_last_seen_at timestamptz NOT NULL,
    frontdoor_ad_data jsonb,
    frontdoor_ad_street_address text,
    frontdoor_ad_city text,
    frontdoor_ad_postal text,
    frontdoor_ad_price int8,
    frontdoor_ad_area_value float8,
    frontdoor_ad_address_key text,
    frontdoor_ad_search_text text,
    frontdoor_ad_page_not_found bool NOT NULL,
    frontdoor_ad_publishing_time timestamptz,
    PRIMARY KEY (frontdoor_ad_id)
);

CREATE TABLE public.frontdoor_buildings (
    frontdoor_building_id uuid NOT NULL,
    frontdoor_building_url text,
    frontdoor_building_last_seen_at timestamptz,
    frontdoor_building_company_name text,
    frontdoor_building_business_id text,
    frontdoor_building_apartment_count int4,
    frontdoor_building_floor_count int4,
    frontdoor_building_build_year int4,
    frontdoor_building_has_elevator bool,
    frontdoor_building_has_sauna bool,
    frontdoor_building_energy_certificate_code text,
    frontdoor_building_heating text,
    frontdoor_building_street_address text,
    frontdoor_building_house_number text,
    frontdoor_building_postcode text,
    frontdoor_building_post_area text,
    frontdoor_building_municipality text,
    frontdoor_building_latitude float8,
    frontdoor_building_longitude float8,
    frontdoor_building_data jsonb,
    frontdoor_building_housing_company_id int8,
    frontdoor_building_housing_company_friendly_id text,
    PRIMARY KEY (frontdoor_building_id)
);

CREATE TABLE public.frontdoor_building_announcements (
    frontdoor_building_announcement_id uuid NOT NULL,
    frontdoor_building_announcement_external_id int4,
    frontdoor_building_announcement_friendly_id text,
    frontdoor_building_announcement_address_line1 text,
    frontdoor_building_announcement_address_line2 text,
    frontdoor_building_announcement_location text,
    frontdoor_building_announcement_search_price float8,
    frontdoor_building_announcement_room_structure text,
    frontdoor_building_announcement_area float8,
    frontdoor_building_announcement_property_type text,
    frontdoor_building_announcement_property_subtype text,
    frontdoor_building_announcement_published bool,
    frontdoor_building_announcement_last_seen_at timestamptz NOT NULL,
    frontdoor_building_id uuid NOT NULL,
    PRIMARY KEY (frontdoor_building_announcement_id)
);
