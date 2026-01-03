INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('prices:neighborhood_postal_codes', 'prices_neighborhood_postal_codes', 'active', 'cron')
ON CONFLICT (entity_id) DO NOTHING;

INSERT INTO task_queue.task_type_entity_type_mapping (task_type, entity_type)
VALUES ('prices_neighborhood_postal_code_sync', 'prices_neighborhood_postal_codes')
ON CONFLICT (task_type, entity_type) DO NOTHING;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_prices_neighborhood_postal_code_sync() RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id, status INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'prices:neighborhood_postal_codes'
      AND task_type = 'prices_neighborhood_postal_code_sync'
      AND run_on = CURRENT_DATE
    LIMIT 1;
    IF FOUND THEN
        IF v_existing_task.status IN ('pending', 'processing') THEN
            RAISE NOTICE 'Prices neighborhood postal code sync already scheduled for today (task_id: %)', v_existing_task.task_id;
            RETURN v_existing_task.task_id;
        END IF;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        max_attempts,
        scheduled_for,
        run_on
    )
    VALUES (
        'prices:neighborhood_postal_codes',
        'prices_neighborhood_postal_code_sync',
        'pending',
        0,
        3,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task_id,
            'entity_id', 'prices:neighborhood_postal_codes',
            'attempt', 0
        )
    );
    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task_id;
    RAISE NOTICE 'Prices neighborhood postal code sync scheduled (task_id: %, msg_id: %)', v_task_id, v_msg_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION task_queue.fnc__schedule_prices_neighborhood_postal_code_sync() IS
'Creates a prices_neighborhood_postal_code_sync task that workers will process to map neighborhoods to postal codes by iterating through all postal codes and their transactions.';

SELECT cron.schedule(
    'trigger-prices-neighborhood-postal-code-sync',
    '0 5 * * 0',
    $$SELECT task_queue.fnc__schedule_prices_neighborhood_postal_code_sync()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-neighborhood-postal-code-sync'
);

INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('prices:sync_all', 'prices_sync_all', 'active', 'cron')
ON CONFLICT (entity_id) DO NOTHING;

INSERT INTO task_queue.task_type_entity_type_mapping (task_type, entity_type)
VALUES ('prices_sync_all', 'prices_sync_all')
ON CONFLICT (task_type, entity_type) DO NOTHING;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_prices_sync_all() RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id, status INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'prices:sync_all'
      AND task_type = 'prices_sync_all'
      AND run_on = CURRENT_DATE
    LIMIT 1;
    IF FOUND THEN
        IF v_existing_task.status IN ('pending', 'processing') THEN
            RAISE NOTICE 'Prices sync all already scheduled for today (task_id: %)', v_existing_task.task_id;
            RETURN v_existing_task.task_id;
        END IF;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        max_attempts,
        scheduled_for,
        run_on
    )
    VALUES (
        'prices:sync_all',
        'prices_sync_all',
        'pending',
        0,
        3,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task_id,
            'entity_id', 'prices:sync_all',
            'attempt', 0
        )
    );
    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task_id;
    RAISE NOTICE 'Prices sync all scheduled (task_id: %, msg_id: %)', v_task_id, v_msg_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION task_queue.fnc__schedule_prices_sync_all() IS
'Creates a prices_sync_all task that syncs all cities, postal codes, neighborhoods and transactions in one go with concurrency.';

SELECT cron.schedule(
    'trigger-prices-sync-all',
    '0 3 * * 0',
    $$SELECT task_queue.fnc__schedule_prices_sync_all()$$
)
WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-sync-all'
);

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
CREATE INDEX idx_postal_municipality_name_fi ON public.postal_municipalities(postal_municipality_name_fi);

INSERT INTO task_queue.entity_registry (entity_id, entity_type, status, scheduling_strategy)
VALUES ('postal:sync', 'postal_sync', 'active', 'manual')
ON CONFLICT (entity_id) DO NOTHING;

INSERT INTO task_queue.task_type_entity_type_mapping (task_type, entity_type)
VALUES ('postal_sync', 'postal_sync')
ON CONFLICT (task_type, entity_type) DO NOTHING;

CREATE OR REPLACE FUNCTION task_queue.fnc__schedule_postal_sync() RETURNS BIGINT AS $$
DECLARE
    v_task_id BIGINT;
    v_msg_id BIGINT;
    v_existing_task RECORD;
BEGIN
    SELECT task_id, status INTO v_existing_task
    FROM task_queue.task
    WHERE entity_id = 'postal:sync'
      AND task_type = 'postal_sync'
      AND run_on = CURRENT_DATE
    LIMIT 1;
    IF FOUND THEN
        IF v_existing_task.status IN ('pending', 'processing') THEN
            RAISE NOTICE 'Postal sync already scheduled for today (task_id: %)', v_existing_task.task_id;
            RETURN v_existing_task.task_id;
        END IF;
    END IF;
    INSERT INTO task_queue.task (
        entity_id,
        task_type,
        status,
        attempt,
        max_attempts,
        scheduled_for,
        run_on
    )
    VALUES (
        'postal:sync',
        'postal_sync',
        'pending',
        0,
        3,
        NOW(),
        CURRENT_DATE
    )
    RETURNING task_id INTO v_task_id;
    v_msg_id := pgmq.send(
        'tasks',
        jsonb_build_object(
            'task_id', v_task_id,
            'entity_id', 'postal:sync',
            'attempt', 0
        )
    );
    UPDATE task_queue.task
    SET queue_message_id = v_msg_id,
        updated_at = NOW()
    WHERE task_id = v_task_id;
    RAISE NOTICE 'Postal sync scheduled (task_id: %, msg_id: %)', v_task_id, v_msg_id;
    RETURN v_task_id;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION task_queue.fnc__schedule_postal_sync() IS
'Creates a postal_sync task to sync Finnish postal codes, municipalities, and administrative areas from Posti.';

ALTER TABLE public.prices_neighborhoods
ADD COLUMN prices_neighborhood_postal_postal_code_id uuid REFERENCES public.postal_postal_codes(postal_postal_code_id);

CREATE INDEX idx_prices_neighborhood_postal_postal_code_id
ON public.prices_neighborhoods(prices_neighborhood_postal_postal_code_id);

ALTER TABLE public.frontdoor_ads
ADD COLUMN postal_postal_code_id uuid
    REFERENCES public.postal_postal_codes(postal_postal_code_id);

CREATE INDEX idx_frontdoor_ad_postal_postal_code_id
ON public.frontdoor_ads(postal_postal_code_id);

UPDATE public.frontdoor_ads fa
SET postal_postal_code_id = pp.postal_postal_code_id
FROM public.postal_postal_codes pp
WHERE fa.frontdoor_ads_data->'property'->'postCode'->>'postCode' = pp.postal_postal_code_code;

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


ALTER TABLE public.prices_cities
    RENAME COLUMN prices_cities_id TO prices_city_id;
ALTER TABLE public.prices_cities
    RENAME COLUMN prices_cities_name TO prices_city_name;
ALTER TABLE public.prices_cities
    RENAME COLUMN prices_cities_created_at TO prices_city_created_at;
ALTER TABLE public.prices_cities
    RENAME COLUMN prices_cities_updated_at TO prices_city_updated_at;

ALTER TABLE public.prices_postal_codes
    RENAME COLUMN prices_postal_codes_id TO prices_postal_code_id;
ALTER TABLE public.prices_postal_codes
    RENAME COLUMN prices_postal_codes_code TO prices_postal_code_code;
ALTER TABLE public.prices_postal_codes
    RENAME COLUMN prices_postal_codes_city_id TO prices_city_id;
ALTER TABLE public.prices_postal_codes
    RENAME COLUMN prices_postal_codes_created_at TO prices_postal_code_created_at;
ALTER TABLE public.prices_postal_codes
    RENAME COLUMN prices_postal_codes_updated_at TO prices_postal_code_updated_at;

ALTER TABLE public.prices_neighborhoods
    RENAME COLUMN prices_neighborhoods_id TO prices_neighborhood_id;
ALTER TABLE public.prices_neighborhoods
    RENAME COLUMN prices_neighborhoods_name TO prices_neighborhood_name;
ALTER TABLE public.prices_neighborhoods
    RENAME COLUMN prices_neighborhoods_city_id TO prices_city_id;
ALTER TABLE public.prices_neighborhoods
    RENAME COLUMN prices_neighborhoods_postal_code_id TO prices_postal_code_id;
ALTER TABLE public.prices_neighborhoods
    RENAME COLUMN prices_neighborhoods_created_at TO prices_neighborhood_created_at;
ALTER TABLE public.prices_neighborhoods
    RENAME COLUMN prices_neighborhoods_updated_at TO prices_neighborhood_updated_at;

ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_id TO prices_transaction_id;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_description TO prices_transaction_description;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_type TO prices_transaction_type;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_area TO prices_transaction_area;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_price TO prices_transaction_price;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_price_per_square_meter TO prices_transaction_price_per_square_meter;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_build_year TO prices_transaction_build_year;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_floor TO prices_transaction_floor;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_elevator TO prices_transaction_elevator;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_condition TO prices_transaction_condition;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_plot TO prices_transaction_plot;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_energy_class TO prices_transaction_energy_class;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_period_identifier TO prices_transaction_period_identifier;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_created_at TO prices_transaction_created_at;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_updated_at TO prices_transaction_updated_at;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_transactions_category TO prices_transaction_category;
ALTER TABLE public.prices_transactions
    RENAME COLUMN prices_neighborhoods_id TO prices_neighborhood_id;

DROP INDEX IF EXISTS idx_prices_transactions_period_identifier;
CREATE INDEX idx_prices_transaction_period_identifier
    ON public.prices_transactions(prices_transaction_period_identifier);

ALTER TABLE public.prices_transactions DROP CONSTRAINT IF EXISTS prices_transactions_unique_key;
ALTER TABLE public.prices_transactions
    ADD CONSTRAINT prices_transaction_unique_key UNIQUE NULLS NOT DISTINCT (
        prices_neighborhood_id,
        prices_transaction_description,
        prices_transaction_type,
        prices_transaction_area,
        prices_transaction_price,
        prices_transaction_price_per_square_meter,
        prices_transaction_build_year,
        prices_transaction_floor,
        prices_transaction_elevator,
        prices_transaction_condition,
        prices_transaction_plot,
        prices_transaction_energy_class,
        prices_transaction_category
    );

-- shortcut_buildings
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_id TO shortcut_building_id;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_external_id TO shortcut_building_external_id;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_building_id TO shortcut_building_building_id;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_building_type TO shortcut_building_building_type;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_building_subtype TO shortcut_building_building_subtype;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_construction_year TO shortcut_building_construction_year;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_floor_count TO shortcut_building_floor_count;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_apartment_count TO shortcut_building_apartment_count;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_heating_system TO shortcut_building_heating_system;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_building_material TO shortcut_building_building_material;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_plot_type TO shortcut_building_plot_type;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_wall_structure TO shortcut_building_wall_structure;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_heat_source TO shortcut_building_heat_source;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_has_elevator TO shortcut_building_has_elevator;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_has_sauna TO shortcut_building_has_sauna;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_latitude TO shortcut_building_latitude;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_longitude TO shortcut_building_longitude;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_additional_addresses TO shortcut_building_additional_addresses;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_url TO shortcut_building_url;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_created_at TO shortcut_building_created_at;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_updated_at TO shortcut_building_updated_at;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_address TO shortcut_building_address;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_processed_at TO shortcut_building_processed_at;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_page_not_found TO shortcut_building_page_not_found;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_frame_construction_method TO shortcut_building_frame_construction_method;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_housing_company TO shortcut_building_housing_company;
ALTER TABLE public.shortcut_buildings
    RENAME COLUMN shortcut_buildings_geom TO shortcut_building_geom;

DROP INDEX IF EXISTS shortcut_buildings_geom_idx;
CREATE INDEX shortcut_building_geom_idx ON public.shortcut_buildings USING GIST (shortcut_building_geom);

ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_id TO shortcut_building_listing_id;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_building_id TO shortcut_building_id;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_layout TO shortcut_building_listing_layout;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_size TO shortcut_building_listing_size;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_price TO shortcut_building_listing_price;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_price_per_sqm TO shortcut_building_listing_price_per_sqm;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_deleted_at TO shortcut_building_listing_deleted_at;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_created_at TO shortcut_building_listing_created_at;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_updated_at TO shortcut_building_listing_updated_at;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_marketing_time TO shortcut_building_listing_marketing_time;
ALTER TABLE public.shortcut_building_listings
    RENAME COLUMN shortcut_building_listings_idx TO shortcut_building_listing_idx;

ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_id TO shortcut_building_rental_id;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_building_id TO shortcut_building_id;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_layout TO shortcut_building_rental_layout;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_size TO shortcut_building_rental_size;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_price TO shortcut_building_rental_price;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_deleted_at TO shortcut_building_rental_deleted_at;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_created_at TO shortcut_building_rental_created_at;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_updated_at TO shortcut_building_rental_updated_at;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_marketing_time TO shortcut_building_rental_marketing_time;
ALTER TABLE public.shortcut_building_rentals
    RENAME COLUMN shortcut_building_rentals_idx TO shortcut_building_rental_idx;

ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_id TO shortcut_ad_id;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_url TO shortcut_ad_url;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_type TO shortcut_ad_type;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_first_seen_at TO shortcut_ad_first_seen_at;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_last_seen_at TO shortcut_ad_last_seen_at;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_data TO shortcut_ad_data;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_updated_at TO shortcut_ad_updated_at;
ALTER TABLE public.shortcut_ads
    RENAME COLUMN shortcut_ads_building_id TO shortcut_building_id;

DROP INDEX IF EXISTS idx_shortcut_ads_zipcode_name;
CREATE INDEX idx_shortcut_ad_zipcode_name ON public.shortcut_ads(((((shortcut_ad_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));

ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_id TO shortcut_token_id;
ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_cuid TO shortcut_token_cuid;
ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_token TO shortcut_token_token;
ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_loaded TO shortcut_token_loaded;
ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_created_at TO shortcut_token_created_at;
ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_updated_at TO shortcut_token_updated_at;
ALTER TABLE public.shortcut_tokens
    RENAME COLUMN shortcut_tokens_expires_at TO shortcut_token_expires_at;

DROP INDEX IF EXISTS idx_shortcut_tokens_expires_at;
DROP INDEX IF EXISTS idx_shortcut_tokens_cuid;
CREATE INDEX idx_shortcut_token_expires_at ON public.shortcut_tokens(shortcut_token_expires_at DESC);
CREATE INDEX idx_shortcut_token_cuid ON public.shortcut_tokens(shortcut_token_cuid);

ALTER TABLE public.shortcut_tokens DROP CONSTRAINT IF EXISTS shortcut_tokens_shortcut_tokens_cuid_key;
ALTER TABLE public.shortcut_tokens ADD CONSTRAINT shortcut_token_cuid_key UNIQUE(shortcut_token_cuid);

ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_id TO frontdoor_ad_id;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_external_id TO frontdoor_ad_external_id;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_url TO frontdoor_ad_url;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_first_seen_at TO frontdoor_ad_first_seen_at;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_last_seen_at TO frontdoor_ad_last_seen_at;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_updated_at TO frontdoor_ad_updated_at;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_data TO frontdoor_ad_data;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_processed_at TO frontdoor_ad_processed_at;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_page_not_found TO frontdoor_ad_page_not_found;
ALTER TABLE public.frontdoor_ads
    RENAME COLUMN frontdoor_ads_publishing_time TO frontdoor_ad_publishing_time;

DROP INDEX IF EXISTS idx_frontdoor_ads_processed_at;
DROP INDEX IF EXISTS idx_frontdoor_ads_page_not_found;
CREATE INDEX idx_frontdoor_ad_processed_at ON public.frontdoor_ads(frontdoor_ad_processed_at);
CREATE INDEX idx_frontdoor_ad_page_not_found ON public.frontdoor_ads(frontdoor_ad_page_not_found);

ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_id TO frontdoor_building_id;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_url TO frontdoor_building_url;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_first_seen_at TO frontdoor_building_first_seen_at;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_last_seen_at TO frontdoor_building_last_seen_at;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_updated_at TO frontdoor_building_updated_at;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_company_name TO frontdoor_building_company_name;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_business_id TO frontdoor_building_business_id;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_apartment_count TO frontdoor_building_apartment_count;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_floor_count TO frontdoor_building_floor_count;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_construction_end_year TO frontdoor_building_construction_end_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_build_year TO frontdoor_building_build_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_has_elevator TO frontdoor_building_has_elevator;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_has_sauna TO frontdoor_building_has_sauna;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_energy_certificate_code TO frontdoor_building_energy_certificate_code;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_plot_holding_type TO frontdoor_building_plot_holding_type;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_outer_roof_material TO frontdoor_building_outer_roof_material;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_outer_roof_type TO frontdoor_building_outer_roof_type;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_heating TO frontdoor_building_heating;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_heating_fuel TO frontdoor_building_heating_fuel;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_street_address TO frontdoor_building_street_address;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_house_number TO frontdoor_building_house_number;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_postcode TO frontdoor_building_postcode;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_post_area TO frontdoor_building_post_area;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_municipality TO frontdoor_building_municipality;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_district TO frontdoor_building_district;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_latitude TO frontdoor_building_latitude;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_longitude TO frontdoor_building_longitude;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_elevator_renovated TO frontdoor_building_elevator_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_elevator_renovated_year TO frontdoor_building_elevator_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_facade_renovated TO frontdoor_building_facade_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_facade_renovated_year TO frontdoor_building_facade_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_window_renovated TO frontdoor_building_window_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_window_renovated_year TO frontdoor_building_window_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_roof_renovated TO frontdoor_building_roof_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_roof_renovated_year TO frontdoor_building_roof_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_pipe_renovated TO frontdoor_building_pipe_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_pipe_renovated_year TO frontdoor_building_pipe_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_balcony_renovated TO frontdoor_building_balcony_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_balcony_renovated_year TO frontdoor_building_balcony_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_electricity_renovated TO frontdoor_building_electricity_renovated;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_electricity_renovated_year TO frontdoor_building_electricity_renovated_year;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_contact_phone TO frontdoor_building_contact_phone;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_contact_office_name TO frontdoor_building_contact_office_name;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_contact_office_id TO frontdoor_building_contact_office_id;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_description TO frontdoor_building_description;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_car_storage_description TO frontdoor_building_car_storage_description;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_other_info TO frontdoor_building_other_info;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_additional_addresses TO frontdoor_building_additional_addresses;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_links TO frontdoor_building_links;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_data TO frontdoor_building_data;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_processed_at TO frontdoor_building_processed_at;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_housing_company_id TO frontdoor_building_housing_company_id;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_housing_company_friendly_id TO frontdoor_building_housing_company_friendly_id;
ALTER TABLE public.frontdoor_buildings
    RENAME COLUMN frontdoor_buildings_geom TO frontdoor_building_geom;

DROP INDEX IF EXISTS idx_frontdoor_buildings_processed_at;
DROP INDEX IF EXISTS idx_frontdoor_buildings_business_id;
CREATE INDEX idx_frontdoor_building_processed_at ON public.frontdoor_buildings(frontdoor_building_processed_at);
CREATE INDEX idx_frontdoor_building_business_id ON public.frontdoor_buildings(frontdoor_building_business_id);

ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_id TO frontdoor_building_announcement_id;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_external_id TO frontdoor_building_announcement_external_id;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_friendly_id TO frontdoor_building_announcement_friendly_id;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_unpublishing_time TO frontdoor_building_announcement_unpublishing_time;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_address_line1 TO frontdoor_building_announcement_address_line1;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_address_line2 TO frontdoor_building_announcement_address_line2;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_location TO frontdoor_building_announcement_location;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_search_price TO frontdoor_building_announcement_search_price;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_notify_price_changed TO frontdoor_building_announcement_notify_price_changed;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_property_type TO frontdoor_building_announcement_property_type;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_property_subtype TO frontdoor_building_announcement_property_subtype;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_construction_finished_year TO frontdoor_building_announcement_construction_finished_year;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_main_image_uri TO frontdoor_building_announcement_main_image_uri;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_has_open_bidding TO frontdoor_building_announcement_has_open_bidding;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_room_structure TO frontdoor_building_announcement_room_structure;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_area TO frontdoor_building_announcement_area;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_total_area TO frontdoor_building_announcement_total_area;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_price_per_square TO frontdoor_building_announcement_price_per_square;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_days_on_market TO frontdoor_building_announcement_days_on_market;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_new_building TO frontdoor_building_announcement_new_building;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_main_image_hidden TO frontdoor_building_announcement_main_image_hidden;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_is_company_announcement TO frontdoor_building_announcement_is_company_announcement;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_show_bidding_indicators TO frontdoor_building_announcement_show_bidding_indicators;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_published TO frontdoor_building_announcement_published;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_rent_period TO frontdoor_building_announcement_rent_period;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_rental_unique_no TO frontdoor_building_announcement_rental_unique_no;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_building_id TO frontdoor_building_id;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_first_seen_at TO frontdoor_building_announcement_first_seen_at;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_last_seen_at TO frontdoor_building_announcement_last_seen_at;
ALTER TABLE public.frontdoor_building_announcements
    RENAME COLUMN frontdoor_building_announcements_unpublishing_time_date TO frontdoor_building_announcement_unpublishing_time_date;

DROP INDEX IF EXISTS idx_frontdoor_building_announcements_building_id;
CREATE INDEX idx_frontdoor_building_announcement_building_id
    ON public.frontdoor_building_announcements(frontdoor_building_id);

DROP TRIGGER IF EXISTS tg__frontdoor_ads_link_postal_code ON public.frontdoor_ads;
DROP FUNCTION IF EXISTS fnc__link_frontdoor_ads_postal_code();

CREATE OR REPLACE FUNCTION fnc__link_frontdoor_ads_postal_code()
RETURNS TRIGGER AS $$
BEGIN
    SELECT postal_postal_code_id INTO NEW.postal_postal_code_id
    FROM public.postal_postal_codes
    WHERE postal_postal_code_code = NEW.frontdoor_ad_data->'property'->'postCode'->>'postCode'
    LIMIT 1;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg__frontdoor_ads_link_postal_code
BEFORE INSERT OR UPDATE OF frontdoor_ad_data
ON public.frontdoor_ads
FOR EACH ROW
EXECUTE FUNCTION fnc__link_frontdoor_ads_postal_code();

DROP VIEW IF EXISTS view__prices_transactions;

CREATE OR REPLACE VIEW view__prices_transactions AS
SELECT
    t.prices_transaction_id,
    t.prices_transaction_description,
    t.prices_transaction_type,
    t.prices_transaction_area,
    t.prices_transaction_price,
    t.prices_transaction_price_per_square_meter,
    t.prices_transaction_build_year,
    t.prices_transaction_floor,
    t.prices_transaction_elevator,
    t.prices_transaction_condition,
    t.prices_transaction_plot,
    t.prices_transaction_energy_class,
    t.prices_transaction_period_identifier,
    t.prices_transaction_category,
    t.prices_transaction_created_at,
    t.prices_transaction_updated_at,
    n.prices_neighborhood_id,
    n.prices_neighborhood_name,
    n.prices_city_id,
    c.prices_city_name,
    p.prices_postal_code_code
FROM public.prices_transactions t
LEFT JOIN public.prices_neighborhoods n ON t.prices_neighborhood_id = n.prices_neighborhood_id
LEFT JOIN public.prices_cities c ON n.prices_city_id = c.prices_city_id
LEFT JOIN public.prices_postal_codes p ON n.prices_postal_code_id = p.prices_postal_code_id;
