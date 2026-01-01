CREATE TABLE public.shortcut_buildings (
    shortcut_building_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_building_external_id int8 NOT NULL,
    shortcut_building_building_id text,
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
    shortcut_building_additional_addresses text,
    shortcut_building_url text NOT NULL,
    shortcut_building_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_address text,
    shortcut_building_processed_at timestamptz,
    shortcut_building_page_not_found bool DEFAULT false,
    shortcut_building_frame_construction_method text,
    shortcut_building_housing_company text,
    shortcut_building_geom geometry(Point, 4326),
    PRIMARY KEY (shortcut_building_id)
);

CREATE UNIQUE INDEX shortcut_building_external_id_key ON public.shortcut_buildings USING btree (shortcut_building_external_id);
CREATE INDEX shortcut_building_geom_idx ON public.shortcut_buildings USING GIST (shortcut_building_geom);

CREATE TABLE public.shortcut_building_listings (
    shortcut_building_listing_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_building_id uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_building_id) ON DELETE CASCADE,
    shortcut_building_listing_layout text,
    shortcut_building_listing_size float8,
    shortcut_building_listing_price float8,
    shortcut_building_listing_price_per_sqm float8,
    shortcut_building_listing_deleted_at timestamptz,
    shortcut_building_listing_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_listing_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_listing_marketing_time text,
    shortcut_building_listing_idx int4,
    PRIMARY KEY (shortcut_building_listing_id)
);

CREATE UNIQUE INDEX shortcut_building_listing_unique_constraint ON public.shortcut_building_listings USING btree (
    shortcut_building_id,
    shortcut_building_listing_layout,
    shortcut_building_listing_size,
    shortcut_building_listing_price,
    shortcut_building_listing_price_per_sqm,
    shortcut_building_listing_deleted_at,
    shortcut_building_listing_marketing_time,
    shortcut_building_listing_idx
);

CREATE TABLE public.shortcut_building_rentals (
    shortcut_building_rental_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_building_id uuid NOT NULL REFERENCES public.shortcut_buildings(shortcut_building_id) ON DELETE CASCADE,
    shortcut_building_rental_layout text,
    shortcut_building_rental_size float8,
    shortcut_building_rental_price float8,
    shortcut_building_rental_deleted_at timestamptz,
    shortcut_building_rental_created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_rental_updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_rental_marketing_time text,
    shortcut_building_rental_idx int4,
    PRIMARY KEY (shortcut_building_rental_id)
);

CREATE UNIQUE INDEX shortcut_building_rental_unique_constraint ON public.shortcut_building_rentals USING btree (
    shortcut_building_id,
    shortcut_building_rental_layout,
    shortcut_building_rental_size,
    shortcut_building_rental_price,
    shortcut_building_rental_deleted_at,
    shortcut_building_rental_marketing_time,
    shortcut_building_rental_idx
);

CREATE TABLE public.shortcut_ads (
    shortcut_ad_id int8 NOT NULL,
    shortcut_ad_url text NOT NULL,
    shortcut_ad_type text NOT NULL,
    shortcut_ad_first_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ad_last_seen_at timestamptz NOT NULL DEFAULT now(),
    shortcut_ad_data jsonb,
    shortcut_ad_updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    shortcut_building_id uuid REFERENCES public.shortcut_buildings(shortcut_building_id) ON DELETE SET NULL,
    PRIMARY KEY (shortcut_ad_id)
);

CREATE INDEX idx_shortcut_ad_zipcode_name ON public.shortcut_ads(((((shortcut_ad_data -> 'address'::text) -> 'zipCode'::text) ->> 'name'::text)));

CREATE TABLE public.shortcut_tokens (
    shortcut_token_id uuid NOT NULL DEFAULT gen_random_uuid(),
    shortcut_token_cuid text NOT NULL,
    shortcut_token_token text NOT NULL,
    shortcut_token_loaded text NOT NULL,
    shortcut_token_created_at timestamptz NOT NULL DEFAULT now(),
    shortcut_token_updated_at timestamptz NOT NULL DEFAULT now(),
    shortcut_token_expires_at timestamptz NOT NULL,
    PRIMARY KEY (shortcut_token_id),
    UNIQUE(shortcut_token_cuid)
);

CREATE INDEX idx_shortcut_token_expires_at ON public.shortcut_tokens USING btree (shortcut_token_expires_at DESC);
CREATE INDEX idx_shortcut_token_cuid ON public.shortcut_tokens USING btree (shortcut_token_cuid);
