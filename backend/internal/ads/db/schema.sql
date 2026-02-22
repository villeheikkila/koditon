CREATE TABLE public.shortcut_buildings (
    shortcut_building_id uuid NOT NULL,
    shortcut_building_external_id int8 NOT NULL,
    shortcut_building_url text NOT NULL,
    shortcut_building_address text,
    shortcut_building_housing_company text,
    PRIMARY KEY (shortcut_building_id)
);

CREATE TABLE public.shortcut_ads (
    shortcut_ad_id int8 NOT NULL,
    shortcut_ad_url text NOT NULL,
    shortcut_ad_type text NOT NULL,
    shortcut_ad_last_seen_at timestamptz NOT NULL,
    shortcut_ad_data jsonb,
    shortcut_building_id uuid,
    PRIMARY KEY (shortcut_ad_id)
);

CREATE TABLE public.frontdoor_ads (
    frontdoor_ad_id uuid NOT NULL,
    frontdoor_ad_external_id text NOT NULL,
    frontdoor_ad_url text NOT NULL,
    frontdoor_ad_last_seen_at timestamptz NOT NULL,
    frontdoor_ad_data jsonb,
    frontdoor_ad_page_not_found bool NOT NULL,
    PRIMARY KEY (frontdoor_ad_id)
);

CREATE TABLE public.frontdoor_buildings (
    frontdoor_building_id uuid NOT NULL,
    frontdoor_building_url text,
    frontdoor_building_housing_company_id int8,
    frontdoor_building_housing_company_friendly_id text,
    frontdoor_building_company_name text,
    frontdoor_building_street_address text,
    frontdoor_building_house_number text,
    frontdoor_building_postcode text,
    frontdoor_building_post_area text,
    frontdoor_building_municipality text,
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
