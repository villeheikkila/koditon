CREATE TABLE public.shortcut_buildings (
    shortcut_buildings_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_buildings_external_id int8 NOT NULL,
    shortcut_buildings_building_id text,
    shortcut_buildings_building_type text,
    shortcut_buildings_building_subtype text,
    shortcut_buildings_construction_year int4,
    shortcut_buildings_floor_count int4,
    shortcut_buildings_apartment_count int4,
    shortcut_buildings_heating_system text,
    shortcut_buildings_building_material text,
    shortcut_buildings_plot_type text,
    shortcut_buildings_wall_structure text,
    shortcut_buildings_heat_source text,
    shortcut_buildings_has_elevator text,
    shortcut_buildings_has_sauna text,
    shortcut_buildings_latitude float8,
    shortcut_buildings_longitude float8,
    shortcut_buildings_additional_addresses text,
    shortcut_buildings_url text NOT NULL,
    shortcut_buildings_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_buildings_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_buildings_address text,
    shortcut_buildings_processed_at timestamptz,
    shortcut_buildings_page_not_found bool DEFAULT false,
    shortcut_buildings_frame_construction_method text,
    shortcut_buildings_housing_company text,
    shortcut_buildings_geom geometry(Point, 4326),
    PRIMARY KEY (shortcut_buildings_id)
);

CREATE UNIQUE INDEX shortcut_buildings_external_id_key ON public.shortcut_buildings USING btree (shortcut_buildings_external_id);
CREATE INDEX shortcut_buildings_geom_idx ON public.shortcut_buildings USING GIST (shortcut_buildings_geom);

CREATE TABLE public.shortcut_building_listings (
    shortcut_building_listings_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_building_listings_building_id uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_buildings_id) ON DELETE CASCADE,
    shortcut_building_listings_layout text,
    shortcut_building_listings_size float8,
    shortcut_building_listings_price float8,
    shortcut_building_listings_price_per_sqm float8,
    shortcut_building_listings_deleted_at timestamptz,
    shortcut_building_listings_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_listings_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_listings_marketing_time text,
    shortcut_building_listings_idx int4,
    PRIMARY KEY (shortcut_building_listings_id)
);

CREATE UNIQUE INDEX shortcut_building_listings_unique_constraint ON public.shortcut_building_listings USING btree (
    shortcut_building_listings_building_id,
    shortcut_building_listings_layout,
    shortcut_building_listings_size,
    shortcut_building_listings_price,
    shortcut_building_listings_price_per_sqm,
    shortcut_building_listings_deleted_at,
    shortcut_building_listings_marketing_time,
    shortcut_building_listings_idx
);

CREATE TABLE public.shortcut_building_rentals (
    shortcut_building_rentals_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_building_rentals_building_id uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_buildings_id) ON DELETE CASCADE,
    shortcut_building_rentals_layout text,
    shortcut_building_rentals_size float8,
    shortcut_building_rentals_price float8,
    shortcut_building_rentals_deleted_at timestamptz,
    shortcut_building_rentals_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_rentals_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_rentals_marketing_time text,
    shortcut_building_rentals_idx int4,
    PRIMARY KEY (shortcut_building_rentals_id)
);

CREATE UNIQUE INDEX shortcut_building_rentals_unique_constraint ON public.shortcut_building_rentals USING btree (
    shortcut_building_rentals_building_id,
    shortcut_building_rentals_layout,
    shortcut_building_rentals_size,
    shortcut_building_rentals_price,
    shortcut_building_rentals_deleted_at,
    shortcut_building_rentals_marketing_time,
    shortcut_building_rentals_idx
);

CREATE TABLE public.shortcut_ads (
    shortcut_ads_id int8 NOT NULL,
    shortcut_ads_url text NOT NULL,
    shortcut_ads_type text NOT NULL,
    shortcut_ads_first_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ads_last_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ads_data jsonb,
    shortcut_ads_updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    shortcut_ads_building_id uuid REFERENCES public.shortcut_buildings(shortcut_buildings_id) ON DELETE SET NULL,
    PRIMARY KEY (shortcut_ads_id)
);

CREATE INDEX idx_shortcut_ads_zipcode_name ON public.shortcut_ads(((((shortcut_ads_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));

CREATE TABLE public.shortcut_tokens (
    shortcut_tokens_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_tokens_cuid text NOT NULL,
    shortcut_tokens_token text NOT NULL,
    shortcut_tokens_loaded text NOT NULL,
    shortcut_tokens_created_at timestamptz NOT NULL DEFAULT now(),
    shortcut_tokens_updated_at timestamptz NOT NULL DEFAULT now(),
    shortcut_tokens_expires_at timestamptz NOT NULL,
    PRIMARY KEY (shortcut_tokens_id),
    UNIQUE(shortcut_tokens_cuid)
);

CREATE INDEX idx_shortcut_tokens_expires_at ON public.shortcut_tokens USING btree (shortcut_tokens_expires_at DESC);
CREATE INDEX idx_shortcut_tokens_cuid ON public.shortcut_tokens USING btree (shortcut_tokens_cuid);
