-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgmq;
CREATE EXTENSION IF NOT EXISTS pg_cron;
CREATE SCHEMA IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis SCHEMA postgis;

SET search_path TO public, postgis;

-- ============================================
-- Public Schema Tables
-- ============================================

CREATE TABLE public.prices_cities (
    prices_cities_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_cities_name         text NOT NULL UNIQUE,
    prices_cities_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_cities_updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_postal_codes (
    prices_postal_codes_id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_postal_codes_code       text NOT NULL UNIQUE,
    prices_postal_codes_city_id    uuid NOT NULL REFERENCES public.prices_cities(prices_cities_id),
    prices_postal_codes_created_at timestamptz NOT NULL DEFAULT now(),
    prices_postal_codes_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.prices_neighborhoods (
    prices_neighborhoods_id           uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_neighborhoods_name         text NOT NULL,
    prices_neighborhoods_city_id      uuid NOT NULL REFERENCES public.prices_cities(prices_cities_id),
    prices_neighborhoods_postal_code_id uuid REFERENCES public.prices_postal_codes(prices_postal_codes_id),
    prices_neighborhoods_created_at   timestamptz NOT NULL DEFAULT now(),
    prices_neighborhoods_updated_at   timestamptz NOT NULL DEFAULT now(),
    prices_neighborhood_postal_postal_code_id uuid,
    CONSTRAINT prices_neighborhoods_name_city_unique UNIQUE (prices_neighborhoods_name, prices_neighborhoods_city_id)
);

CREATE TABLE public.prices_transactions (
    prices_transactions_id                          uuid             PRIMARY KEY DEFAULT uuid_generate_v4(),
    prices_transactions_description                 text             NOT NULL,
    prices_transactions_type                        text             NOT NULL,
    prices_transactions_area                        double precision NOT NULL,
    prices_transactions_price                       integer          NOT NULL,
    prices_transactions_price_per_square_meter      integer          NOT NULL,
    prices_transactions_build_year                  integer          NOT NULL,
    prices_transactions_floor                       text,
    prices_transactions_elevator                    boolean          NOT NULL,
    prices_transactions_condition                   text,
    prices_transactions_plot                        text,
    prices_transactions_energy_class                text,
    prices_transactions_period_identifier           text             NOT NULL,
    prices_transactions_created_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transactions_updated_at                  timestamptz      NOT NULL DEFAULT now(),
    prices_transactions_category                    text             NOT NULL,
    prices_neighborhoods_id                         uuid             REFERENCES public.prices_neighborhoods(prices_neighborhoods_id),
    CONSTRAINT prices_transactions_unique_key UNIQUE NULLS NOT DISTINCT (
        prices_neighborhoods_id,
        prices_transactions_description,
        prices_transactions_type,
        prices_transactions_area,
        prices_transactions_price,
        prices_transactions_price_per_square_meter,
        prices_transactions_build_year,
        prices_transactions_floor,
        prices_transactions_elevator,
        prices_transactions_condition,
        prices_transactions_plot,
        prices_transactions_energy_class,
        prices_transactions_category
    )
);

CREATE INDEX idx_prices_transactions_period_identifier
    ON public.prices_transactions(prices_transactions_period_identifier);

CREATE INDEX idx_prices_neighborhood_postal_postal_code_id
    ON public.prices_neighborhoods(prices_neighborhood_postal_postal_code_id);

-- ============================================
-- Task Queue Schema
-- ============================================

CREATE SCHEMA IF NOT EXISTS task_queue;

CREATE TABLE task_queue.entity_registry (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id            text NOT NULL UNIQUE,
    entity_type          text NOT NULL,
    status               text NOT NULL DEFAULT 'active',
    scheduling_strategy  text NOT NULL,
    config               jsonb,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_entity_registry_status ON task_queue.entity_registry(status);
CREATE INDEX idx_entity_registry_entity_type ON task_queue.entity_registry(entity_type);
CREATE INDEX idx_entity_registry_scheduling_strategy ON task_queue.entity_registry(scheduling_strategy);
CREATE INDEX idx_entity_registry_schedulable ON task_queue.entity_registry(scheduling_strategy, status)
    WHERE scheduling_strategy IN ('rate_limit', 'batch') AND status = 'active';

CREATE TABLE task_queue.task_type_entity_type_mapping (
    task_type   text NOT NULL,
    entity_type text NOT NULL,
    PRIMARY KEY (task_type, entity_type)
);

CREATE TABLE task_queue.task (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id      text NOT NULL,
    task_type      text NOT NULL,
    priority       int NOT NULL DEFAULT 0,
    status         text NOT NULL DEFAULT 'pending',
    payload        jsonb,
    result         jsonb,
    error          text,
    retry_count    int NOT NULL DEFAULT 0,
    max_retries    int NOT NULL DEFAULT 3,
    scheduled_for  timestamptz NOT NULL DEFAULT now(),
    started_at     timestamptz,
    completed_at   timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    worker_id      text,
    run_on         text
);

CREATE INDEX idx_task_entity ON task_queue.task(entity_id);
CREATE INDEX idx_task_status ON task_queue.task(status);
CREATE INDEX idx_task_worker ON task_queue.task(worker_id) WHERE status = 'processing';
CREATE INDEX idx_task_scheduled ON task_queue.task(scheduled_for) WHERE status = 'pending';
CREATE INDEX idx_task_updated ON task_queue.task(updated_at);
CREATE INDEX idx_task_run_on ON task_queue.task(run_on);
CREATE INDEX idx_task_priority_scheduled ON task_queue.task(priority DESC, scheduled_for ASC) WHERE status = 'pending';

CREATE TABLE task_queue.dead_letter_queue (
    id                  uuid PRIMARY KEY,
    entity_id           text NOT NULL,
    task_type           text NOT NULL,
    priority            int NOT NULL,
    payload             jsonb,
    error               text NOT NULL,
    retry_count         int NOT NULL,
    original_created_at timestamptz NOT NULL,
    moved_to_dlq_at     timestamptz NOT NULL DEFAULT now(),
    requeued_at         timestamptz
);

CREATE INDEX idx_dlq_entity_id ON task_queue.dead_letter_queue(entity_id);
CREATE INDEX idx_dlq_task_type ON task_queue.dead_letter_queue(task_type);
CREATE INDEX idx_dlq_moved_at ON task_queue.dead_letter_queue(moved_to_dlq_at DESC);
CREATE INDEX idx_dlq_not_requeued ON task_queue.dead_letter_queue(moved_to_dlq_at DESC) WHERE requeued_at IS NULL;

-- ============================================
-- Shortcut.com Tables
-- ============================================

CREATE TABLE public.shortcut_buildings (
    shortcut_buildings_id                       uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    shortcut_buildings_external_id              text NOT NULL UNIQUE,
    shortcut_buildings_name                     text NOT NULL,
    shortcut_buildings_address                  text NOT NULL,
    shortcut_buildings_postal_code              text,
    shortcut_buildings_city                     text,
    shortcut_buildings_country                  text,
    shortcut_buildings_latitude                 double precision,
    shortcut_buildings_longitude                double precision,
    shortcut_buildings_year_built               int,
    shortcut_buildings_year_renovated           int,
    shortcut_buildings_elevator                 boolean,
    shortcut_buildings_sauna                    boolean,
    shortcut_buildings_balcony                  boolean,
    shortcut_buildings_parking                  text,
    shortcut_buildings_heating_type             text,
    shortcut_buildings_data                     jsonb NOT NULL,
    shortcut_buildings_first_seen_at            timestamptz NOT NULL DEFAULT now(),
    shortcut_buildings_last_seen_at             timestamptz NOT NULL DEFAULT now(),
    shortcut_buildings_updated_at               timestamptz NOT NULL DEFAULT now(),
    shortcut_buildings_geom                     postgis.geometry(Point, 4326)
);

CREATE INDEX shortcut_buildings_geom_idx ON public.shortcut_buildings USING GIST (shortcut_buildings_geom);

CREATE TABLE public.shortcut_building_listings (
    shortcut_building_listings_id               uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    shortcut_building_listings_external_id      text NOT NULL UNIQUE,
    shortcut_building_listings_building_id      uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_buildings_id),
    shortcut_building_listings_apartment_number text,
    shortcut_building_listings_floor            text,
    shortcut_building_listings_rooms            text,
    shortcut_building_listings_area             double precision,
    shortcut_building_listings_price            int,
    shortcut_building_listings_price_per_sqm    double precision,
    shortcut_building_listings_available_from   date,
    shortcut_building_listings_status           text,
    shortcut_building_listings_data             jsonb NOT NULL,
    shortcut_building_listings_first_seen_at    timestamptz NOT NULL DEFAULT now(),
    shortcut_building_listings_last_seen_at     timestamptz NOT NULL DEFAULT now(),
    shortcut_building_listings_updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.shortcut_building_rentals (
    shortcut_building_rentals_id               uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    shortcut_building_rentals_external_id      text NOT NULL UNIQUE,
    shortcut_building_rentals_building_id      uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_buildings_id),
    shortcut_building_rentals_apartment_number text,
    shortcut_building_rentals_floor            text,
    shortcut_building_rentals_rooms            text,
    shortcut_building_rentals_area             double precision,
    shortcut_building_rentals_rent             int,
    shortcut_building_rentals_rent_per_sqm     double precision,
    shortcut_building_rentals_available_from   date,
    shortcut_building_rentals_lease_length     text,
    shortcut_building_rentals_status           text,
    shortcut_building_rentals_data             jsonb NOT NULL,
    shortcut_building_rentals_first_seen_at    timestamptz NOT NULL DEFAULT now(),
    shortcut_building_rentals_last_seen_at     timestamptz NOT NULL DEFAULT now(),
    shortcut_building_rentals_updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.shortcut_ads (
    shortcut_ads_id            uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    shortcut_ads_external_id   text NOT NULL UNIQUE,
    shortcut_ads_url           text NOT NULL,
    shortcut_ads_title         text,
    shortcut_ads_description   text,
    shortcut_ads_price         int,
    shortcut_ads_area          double precision,
    shortcut_ads_rooms         text,
    shortcut_ads_data          jsonb NOT NULL,
    shortcut_ads_first_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ads_last_seen_at  timestamptz NOT NULL DEFAULT now(),
    shortcut_ads_updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_shortcut_ads_zipcode_name ON public.shortcut_ads(((((shortcut_ads_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));

CREATE TABLE public.shortcut_tokens (
    shortcut_tokens_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    shortcut_tokens_cuid TEXT NOT NULL,
    shortcut_tokens_token TEXT NOT NULL,
    shortcut_tokens_loaded TEXT NOT NULL,
    shortcut_tokens_created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    shortcut_tokens_updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    shortcut_tokens_expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE(shortcut_tokens_cuid)
);

CREATE INDEX idx_shortcut_tokens_expires_at ON public.shortcut_tokens(shortcut_tokens_expires_at DESC);
CREATE INDEX idx_shortcut_tokens_cuid ON public.shortcut_tokens(shortcut_tokens_cuid);

-- ============================================
-- Frontdoor Tables
-- ============================================

CREATE TABLE public.frontdoor_ads (
    frontdoor_ads_id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    frontdoor_ads_external_id     text NOT NULL UNIQUE,
    frontdoor_ads_url             text NOT NULL UNIQUE,
    frontdoor_ads_first_seen_at   timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_last_seen_at    timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_updated_at      timestamptz NOT NULL DEFAULT now(),
    frontdoor_ads_data            jsonb,
    frontdoor_ads_processed_at    timestamptz,
    frontdoor_ads_page_not_found  boolean DEFAULT false,
    frontdoor_ads_publishing_time timestamptz,
    postal_postal_code_id         uuid
);

CREATE INDEX idx_frontdoor_ads_processed_at ON public.frontdoor_ads(frontdoor_ads_processed_at);
CREATE INDEX idx_frontdoor_ads_page_not_found ON public.frontdoor_ads(frontdoor_ads_page_not_found);
CREATE INDEX idx_frontdoor_ads_postal_postal_code_id ON public.frontdoor_ads(postal_postal_code_id);

CREATE TABLE public.frontdoor_buildings (
    frontdoor_buildings_id                        uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    frontdoor_buildings_url                       text NOT NULL UNIQUE,
    frontdoor_buildings_first_seen_at             timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_last_seen_at              timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_updated_at                timestamptz NOT NULL DEFAULT now(),
    frontdoor_buildings_company_name              text,
    frontdoor_buildings_business_id               text,
    frontdoor_buildings_apartment_count           int,
    frontdoor_buildings_floor_count               int,
    frontdoor_buildings_construction_end_year     int,
    frontdoor_buildings_build_year                int,
    frontdoor_buildings_has_elevator              boolean,
    frontdoor_buildings_has_sauna                 boolean,
    frontdoor_buildings_energy_certificate_code   text,
    frontdoor_buildings_plot_holding_type         text,
    frontdoor_buildings_outer_roof_material       text,
    frontdoor_buildings_outer_roof_type           text,
    frontdoor_buildings_heating                   text,
    frontdoor_buildings_heating_fuel              text,
    frontdoor_buildings_street_address            text,
    frontdoor_buildings_house_number              text,
    frontdoor_buildings_postcode                  text,
    frontdoor_buildings_post_area                 text,
    frontdoor_buildings_municipality              text,
    frontdoor_buildings_district                  text,
    frontdoor_buildings_latitude                  double precision,
    frontdoor_buildings_longitude                 double precision,
    frontdoor_buildings_elevator_renovated        boolean,
    frontdoor_buildings_elevator_renovated_year   int,
    frontdoor_buildings_facade_renovated          boolean,
    frontdoor_buildings_facade_renovated_year     int,
    frontdoor_buildings_window_renovated          boolean,
    frontdoor_buildings_window_renovated_year     int,
    frontdoor_buildings_roof_renovated            boolean,
    frontdoor_buildings_roof_renovated_year       int,
    frontdoor_buildings_pipe_renovated            boolean,
    frontdoor_buildings_pipe_renovated_year       int,
    frontdoor_buildings_balcony_renovated         boolean,
    frontdoor_buildings_balcony_renovated_year    int,
    frontdoor_buildings_electricity_renovated     boolean,
    frontdoor_buildings_electricity_renovated_year int,
    frontdoor_buildings_contact_phone             text,
    frontdoor_buildings_contact_office_name       text,
    frontdoor_buildings_contact_office_id         text,
    frontdoor_buildings_description               text,
    frontdoor_buildings_car_storage_description   text,
    frontdoor_buildings_other_info                text,
    frontdoor_buildings_additional_addresses      jsonb,
    frontdoor_buildings_links                     jsonb,
    frontdoor_buildings_data                      jsonb,
    frontdoor_buildings_processed_at              timestamptz,
    frontdoor_buildings_housing_company_id        text,
    frontdoor_buildings_housing_company_friendly_id text,
    frontdoor_buildings_geom                      postgis.geometry(Point, 4326)
);

CREATE INDEX idx_frontdoor_buildings_processed_at ON public.frontdoor_buildings(frontdoor_buildings_processed_at);
CREATE INDEX idx_frontdoor_buildings_business_id ON public.frontdoor_buildings(frontdoor_buildings_business_id);

CREATE TABLE public.frontdoor_building_announcements (
    frontdoor_building_announcements_id                       uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    frontdoor_building_announcements_external_id              text NOT NULL UNIQUE,
    frontdoor_building_announcements_friendly_id              text,
    frontdoor_building_announcements_unpublishing_time        timestamptz,
    frontdoor_building_announcements_address_line1            text,
    frontdoor_building_announcements_address_line2            text,
    frontdoor_building_announcements_location                 text,
    frontdoor_building_announcements_search_price             int,
    frontdoor_building_announcements_notify_price_changed     boolean,
    frontdoor_building_announcements_property_type            text,
    frontdoor_building_announcements_property_subtype         text,
    frontdoor_building_announcements_construction_finished_year int,
    frontdoor_building_announcements_main_image_uri           text,
    frontdoor_building_announcements_has_open_bidding         boolean,
    frontdoor_building_announcements_room_structure           text,
    frontdoor_building_announcements_area                     double precision,
    frontdoor_building_announcements_total_area               double precision,
    frontdoor_building_announcements_price_per_square         int,
    frontdoor_building_announcements_days_on_market           int,
    frontdoor_building_announcements_new_building             boolean,
    frontdoor_building_announcements_main_image_hidden        boolean,
    frontdoor_building_announcements_is_company_announcement  boolean,
    frontdoor_building_announcements_show_bidding_indicators  boolean,
    frontdoor_building_announcements_published                boolean,
    frontdoor_building_announcements_rent_period              text,
    frontdoor_building_announcements_rental_unique_no         text,
    frontdoor_building_announcements_building_id              uuid REFERENCES public.frontdoor_buildings(frontdoor_buildings_id),
    frontdoor_building_announcements_first_seen_at            timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcements_last_seen_at             timestamptz NOT NULL DEFAULT now(),
    frontdoor_building_announcements_unpublishing_time_date   date
);

CREATE INDEX idx_frontdoor_building_announcements_building_id
    ON public.frontdoor_building_announcements(frontdoor_building_announcements_building_id);

-- ============================================
-- Postal Tables (Finnish Postal Service)
-- ============================================

CREATE TABLE public.postal_ad_areas (
    postal_ad_area_id         uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    postal_ad_area_code       text NOT NULL UNIQUE,
    postal_ad_area_name_fi    text NOT NULL,
    postal_ad_area_name_sv    text,
    postal_ad_area_created_at timestamptz NOT NULL DEFAULT now(),
    postal_ad_area_updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public.postal_municipalities (
    postal_municipality_id                  uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    postal_municipality_code                text NOT NULL UNIQUE,
    postal_municipality_name_fi             text NOT NULL,
    postal_municipality_name_sv             text,
    postal_municipality_language_ratio_code text,
    postal_municipality_created_at          timestamptz NOT NULL DEFAULT now(),
    postal_municipality_updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_postal_municipality_name_fi ON public.postal_municipalities(postal_municipality_name_fi);

CREATE TABLE public.postal_postal_codes (
    postal_postal_code_id              uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    postal_postal_code_date            date NOT NULL,
    postal_postal_code_code            text NOT NULL UNIQUE,
    postal_postal_code_name_fi         text NOT NULL,
    postal_postal_code_name_sv         text,
    postal_postal_code_abbr_fi         text,
    postal_postal_code_abbr_sv         text,
    postal_postal_code_valid_from      date,
    postal_postal_code_type_code       text,
    postal_ad_area_id                  uuid REFERENCES public.postal_ad_areas(postal_ad_area_id),
    postal_municipality_id             uuid REFERENCES public.postal_municipalities(postal_municipality_id),
    postal_postal_code_created_at      timestamptz NOT NULL DEFAULT now(),
    postal_postal_code_updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_postal_postal_code_name_fi ON public.postal_postal_codes(postal_postal_code_name_fi);
CREATE INDEX idx_postal_postal_code_ad_area_id ON public.postal_postal_codes(postal_ad_area_id);
CREATE INDEX idx_postal_postal_code_municipality_id ON public.postal_postal_codes(postal_municipality_id);

-- Add foreign key from frontdoor_ads to postal_postal_codes
ALTER TABLE public.frontdoor_ads
    ADD CONSTRAINT frontdoor_ads_postal_postal_code_id_fkey
    FOREIGN KEY (postal_postal_code_id) REFERENCES public.postal_postal_codes(postal_postal_code_id);

-- Add foreign key from prices_neighborhoods to postal_postal_codes
ALTER TABLE public.prices_neighborhoods
    ADD CONSTRAINT prices_neighborhoods_postal_postal_code_id_fkey
    FOREIGN KEY (prices_neighborhood_postal_postal_code_id) REFERENCES public.postal_postal_codes(postal_postal_code_id);

-- ============================================
-- Functions and Triggers
-- ============================================

CREATE OR REPLACE FUNCTION fnc__link_frontdoor_ads_postal_code()
RETURNS TRIGGER AS $$
BEGIN
    SELECT postal_postal_code_id INTO NEW.postal_postal_code_id
    FROM public.postal_postal_codes
    WHERE postal_postal_code_code = NEW.frontdoor_ads_data->'property'->'postCode'->>'postCode'
    LIMIT 1;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg__frontdoor_ads_link_postal_code
BEFORE INSERT OR UPDATE OF frontdoor_ads_data
ON public.frontdoor_ads
FOR EACH ROW
EXECUTE FUNCTION fnc__link_frontdoor_ads_postal_code();

-- ============================================
-- PGMQ Queues
-- ============================================

SELECT pgmq.create('tasks');
SELECT pgmq.create('tasks_dlq');

-- ============================================
-- Views
-- ============================================

CREATE OR REPLACE VIEW view__prices_transactions AS
SELECT
    t.prices_transactions_id,
    t.prices_transactions_description,
    t.prices_transactions_type,
    t.prices_transactions_area,
    t.prices_transactions_price,
    t.prices_transactions_price_per_square_meter,
    t.prices_transactions_build_year,
    t.prices_transactions_floor,
    t.prices_transactions_elevator,
    t.prices_transactions_condition,
    t.prices_transactions_plot,
    t.prices_transactions_energy_class,
    t.prices_transactions_period_identifier,
    t.prices_transactions_category,
    t.prices_transactions_created_at,
    t.prices_transactions_updated_at,
    n.prices_neighborhoods_id,
    n.prices_neighborhoods_name,
    n.prices_neighborhoods_city_id,
    c.prices_cities_name,
    p.prices_postal_codes_code
FROM public.prices_transactions t
LEFT JOIN public.prices_neighborhoods n ON t.prices_neighborhoods_id = n.prices_neighborhoods_id
LEFT JOIN public.prices_cities c ON n.prices_neighborhoods_city_id = c.prices_cities_id
LEFT JOIN public.prices_postal_codes p ON n.prices_neighborhoods_postal_code_id = p.prices_postal_codes_id;
