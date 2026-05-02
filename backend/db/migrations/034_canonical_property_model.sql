CREATE TABLE IF NOT EXISTS public.property_buildings (
    property_building_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_building_identity_key text NOT NULL UNIQUE,
    property_building_postal_norm text,
    property_building_city_norm text,
    property_building_address_norm text,
    property_building_housing_company text,
    property_building_business_id text,
    property_building_build_year integer,
    property_building_floor_count integer,
    property_building_apartment_count integer,
    property_building_elevator boolean,
    property_building_energy_efficiency_label text,
    property_building_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_building_created_at timestamptz DEFAULT now() NOT NULL,
    property_building_updated_at timestamptz DEFAULT now() NOT NULL
);
CREATE TABLE IF NOT EXISTS public.property_units (
    property_unit_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_building_id uuid NOT NULL REFERENCES public.property_buildings(property_building_id) ON DELETE CASCADE,
    property_unit_identity_key text NOT NULL UNIQUE,
    property_unit_address_norm text,
    property_unit_floor_level integer,
    property_unit_area_value double precision,
    property_unit_rooms_count integer,
    property_unit_room_layout text,
    property_unit_layout_match_key text,
    property_unit_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_unit_created_at timestamptz DEFAULT now() NOT NULL,
    property_unit_updated_at timestamptz DEFAULT now() NOT NULL
);
CREATE TABLE IF NOT EXISTS public.property_offerings (
    property_offering_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_unit_id uuid NOT NULL REFERENCES public.property_units(property_unit_id) ON DELETE CASCADE,
    property_offering_identity_key text NOT NULL UNIQUE,
    property_offering_type text NOT NULL,
    property_offering_headline text NOT NULL,
    property_offering_asking_price bigint,
    property_offering_debt_free_price bigint,
    property_offering_price_per_m2 double precision,
    property_offering_first_seen_at timestamptz,
    property_offering_last_seen_at timestamptz,
    property_offering_status text,
    primary_sale_listing_id uuid REFERENCES public.sale_listings(sale_listing_id) ON DELETE SET NULL,
    property_offering_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_offering_created_at timestamptz DEFAULT now() NOT NULL,
    property_offering_updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT property_offerings_type_check CHECK (property_offering_type = ANY (ARRAY['sale'::text]))
);
CREATE TABLE IF NOT EXISTS public.property_offering_sources (
    property_offering_source_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    sale_listing_id uuid NOT NULL REFERENCES public.sale_listings(sale_listing_id) ON DELETE CASCADE,
    property_offering_source_link_status text NOT NULL,
    property_offering_source_link_method text NOT NULL,
    property_offering_source_link_score integer NOT NULL,
    property_offering_source_link_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_offering_source_created_at timestamptz DEFAULT now() NOT NULL,
    property_offering_source_updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT property_offering_sources_status_check CHECK (property_offering_source_link_status = ANY (ARRAY['confirmed'::text, 'candidate'::text, 'rejected'::text])),
    CONSTRAINT property_offering_sources_method_check CHECK (property_offering_source_link_method = ANY (ARRAY['backfill_auto'::text, 'sync_auto'::text, 'source_match_auto'::text, 'manual'::text]))
);
CREATE TABLE IF NOT EXISTS public.property_offering_transactions (
    property_offering_transaction_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    prices_transaction_id uuid NOT NULL REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE CASCADE,
    property_offering_transaction_link_status text NOT NULL,
    property_offering_transaction_link_method text NOT NULL,
    property_offering_transaction_link_score integer NOT NULL,
    property_offering_transaction_link_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_offering_transaction_created_at timestamptz DEFAULT now() NOT NULL,
    property_offering_transaction_updated_at timestamptz DEFAULT now() NOT NULL,
    CONSTRAINT property_offering_transactions_unique UNIQUE (property_offering_id, prices_transaction_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_property_offering_sources_active_source ON public.property_offering_sources (sale_listing_id) WHERE property_offering_source_link_status <> 'rejected';
CREATE INDEX IF NOT EXISTS idx_property_offering_sources_offering ON public.property_offering_sources (property_offering_id);
CREATE INDEX IF NOT EXISTS idx_property_offerings_unit ON public.property_offerings (property_unit_id);
CREATE INDEX IF NOT EXISTS idx_property_offerings_primary_sale_listing ON public.property_offerings (primary_sale_listing_id);
CREATE INDEX IF NOT EXISTS idx_property_units_building ON public.property_units (property_building_id);
CREATE OR REPLACE FUNCTION public.fnc__canonical_identity_part(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF(public.fnc__match_alias_key(value), '')
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_canonical_property_for_sale_listing(listing_id uuid, link_method text DEFAULT 'sync_auto')
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    building_id uuid;
    unit_id uuid;
    offering_id uuid;
BEGIN
    WITH source_values AS (
        SELECT
            sl.sale_listing_id,
            sl.sale_listing_source_provider,
            sl.sale_listing_source_kind,
            sl.sale_listing_native_id,
            sl.sale_listing_headline,
            sl.sale_listing_postal_norm,
            sl.sale_listing_city_norm,
            sl.sale_listing_address_norm,
            sl.sale_listing_building_match_key,
            sl.sale_listing_unit_match_key,
            sl.sale_listing_floor_level,
            sl.sale_listing_area_value,
            sl.sale_listing_rooms_count,
            sl.sale_listing_room_layout,
            sl.sale_listing_asking_price,
            sl.sale_listing_debt_free_price,
            sl.sale_listing_price_per_m2,
            sl.sale_listing_first_seen_at,
            sl.sale_listing_last_seen_at,
            sl.sale_listing_build_year,
            sl.sale_listing_total_floors,
            sl.sale_listing_elevator,
            sl.sale_listing_energy_efficiency_label,
            COALESCE(
                sa.shortcut_ad_data #>> '{adData,housingCompanyBusinessId}',
                fa.frontdoor_ad_data #>> '{property,housingCompany,businessId}',
                fb.frontdoor_building_business_id
            ) AS business_id,
            COALESCE(
                sa.shortcut_ad_data #>> '{adData,housingCompanyName}',
                sb.shortcut_building_housing_company,
                fa.frontdoor_ad_data #>> '{property,housingCompany,name}',
                fb.frontdoor_building_company_name
            ) AS housing_company,
            COALESCE(
                sa.shortcut_ad_data #>> '{buildingId}',
                sa.shortcut_ad_data #>> '{adData,buildingId}',
                sb.shortcut_building_external_id::text,
                sb.shortcut_building_building_id,
                fa.frontdoor_ad_data #>> '{property,housingCompany,id}',
                fb.frontdoor_building_housing_company_id::text,
                fba.frontdoor_building_id::text
            ) AS provider_building_id
        FROM public.sale_listings sl
        LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
        LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
        LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE sl.sale_listing_id = listing_id
    ),
    identity_values AS (
        SELECT
            *,
            COALESCE(
                'business:' || public.fnc__canonical_identity_part(business_id),
                'provider_building:' || sale_listing_source_provider || ':' || public.fnc__canonical_identity_part(provider_building_id),
                'address:' || public.fnc__canonical_identity_part(concat_ws('|', sale_listing_postal_norm, sale_listing_city_norm, sale_listing_building_match_key, housing_company)),
                'source:' || sale_listing_source_provider || ':' || sale_listing_source_kind || ':' || sale_listing_native_id
            ) AS building_key
        FROM source_values
    ),
    inserted AS (
        INSERT INTO public.property_buildings (
            property_building_identity_key,
            property_building_postal_norm,
            property_building_city_norm,
            property_building_address_norm,
            property_building_housing_company,
            property_building_business_id,
            property_building_build_year,
            property_building_floor_count,
            property_building_elevator,
            property_building_energy_efficiency_label,
            property_building_match_reasons,
            property_building_updated_at
        )
        SELECT
            building_key,
            sale_listing_postal_norm,
            sale_listing_city_norm,
            sale_listing_address_norm,
            housing_company,
            business_id,
            sale_listing_build_year,
            sale_listing_total_floors,
            sale_listing_elevator,
            sale_listing_energy_efficiency_label,
            jsonb_build_object('source', sale_listing_source_provider, 'provider_building_id', provider_building_id),
            now()
        FROM identity_values
        ON CONFLICT (property_building_identity_key) DO UPDATE SET
            property_building_postal_norm = COALESCE(property_buildings.property_building_postal_norm, EXCLUDED.property_building_postal_norm),
            property_building_city_norm = COALESCE(property_buildings.property_building_city_norm, EXCLUDED.property_building_city_norm),
            property_building_address_norm = COALESCE(property_buildings.property_building_address_norm, EXCLUDED.property_building_address_norm),
            property_building_housing_company = COALESCE(property_buildings.property_building_housing_company, EXCLUDED.property_building_housing_company),
            property_building_business_id = COALESCE(property_buildings.property_building_business_id, EXCLUDED.property_building_business_id),
            property_building_build_year = COALESCE(property_buildings.property_building_build_year, EXCLUDED.property_building_build_year),
            property_building_floor_count = COALESCE(property_buildings.property_building_floor_count, EXCLUDED.property_building_floor_count),
            property_building_elevator = COALESCE(property_buildings.property_building_elevator, EXCLUDED.property_building_elevator),
            property_building_energy_efficiency_label = COALESCE(property_buildings.property_building_energy_efficiency_label, EXCLUDED.property_building_energy_efficiency_label),
            property_building_updated_at = now()
        RETURNING property_building_id
    )
    SELECT property_building_id INTO building_id FROM inserted;
    WITH source_values AS (
        SELECT
            sl.*,
            pb.property_building_id,
            pb.property_building_identity_key
        FROM public.sale_listings sl
        JOIN public.property_buildings pb ON pb.property_building_id = building_id
        WHERE sl.sale_listing_id = listing_id
    ),
    identity_values AS (
        SELECT
            *,
            property_building_identity_key || ':unit:' || COALESCE(
                public.fnc__canonical_identity_part(sale_listing_unit_match_key),
                public.fnc__canonical_identity_part(concat_ws('|', sale_listing_floor_level::text, sale_listing_area_value::text, public.fnc__layout_match_key(sale_listing_room_layout))),
                sale_listing_id::text
            ) AS unit_key
        FROM source_values
    ),
    inserted AS (
        INSERT INTO public.property_units (
            property_building_id,
            property_unit_identity_key,
            property_unit_address_norm,
            property_unit_floor_level,
            property_unit_area_value,
            property_unit_rooms_count,
            property_unit_room_layout,
            property_unit_layout_match_key,
            property_unit_match_reasons,
            property_unit_updated_at
        )
        SELECT
            property_building_id,
            unit_key,
            sale_listing_address_norm,
            sale_listing_floor_level,
            sale_listing_area_value,
            sale_listing_rooms_count,
            sale_listing_room_layout,
            public.fnc__layout_match_key(sale_listing_room_layout),
            jsonb_build_object('source_listing_id', sale_listing_id),
            now()
        FROM identity_values
        ON CONFLICT (property_unit_identity_key) DO UPDATE SET
            property_unit_address_norm = COALESCE(property_units.property_unit_address_norm, EXCLUDED.property_unit_address_norm),
            property_unit_floor_level = COALESCE(property_units.property_unit_floor_level, EXCLUDED.property_unit_floor_level),
            property_unit_area_value = COALESCE(property_units.property_unit_area_value, EXCLUDED.property_unit_area_value),
            property_unit_rooms_count = COALESCE(property_units.property_unit_rooms_count, EXCLUDED.property_unit_rooms_count),
            property_unit_room_layout = COALESCE(property_units.property_unit_room_layout, EXCLUDED.property_unit_room_layout),
            property_unit_layout_match_key = COALESCE(property_units.property_unit_layout_match_key, EXCLUDED.property_unit_layout_match_key),
            property_unit_updated_at = now()
        RETURNING property_unit_id
    )
    SELECT property_unit_id INTO unit_id FROM inserted;
    WITH source_values AS (
        SELECT
            sl.*,
            pu.property_unit_id,
            pu.property_unit_identity_key
        FROM public.sale_listings sl
        JOIN public.property_units pu ON pu.property_unit_id = unit_id
        WHERE sl.sale_listing_id = listing_id
    ),
    identity_values AS (
        SELECT
            *,
            property_unit_identity_key || ':sale:' || COALESCE(sale_listing_debt_free_price::text, sale_listing_asking_price::text, 'unknown') AS offering_key
        FROM source_values
    ),
    inserted AS (
        INSERT INTO public.property_offerings (
            property_unit_id,
            property_offering_identity_key,
            property_offering_type,
            property_offering_headline,
            property_offering_asking_price,
            property_offering_debt_free_price,
            property_offering_price_per_m2,
            property_offering_first_seen_at,
            property_offering_last_seen_at,
            property_offering_status,
            primary_sale_listing_id,
            property_offering_match_reasons,
            property_offering_updated_at
        )
        SELECT
            property_unit_id,
            offering_key,
            'sale',
            sale_listing_headline,
            sale_listing_asking_price,
            sale_listing_debt_free_price,
            sale_listing_price_per_m2,
            sale_listing_first_seen_at,
            sale_listing_last_seen_at,
            sale_listing_prices_match_status,
            sale_listing_id,
            jsonb_build_object('source_listing_id', sale_listing_id, 'identity_key', offering_key),
            now()
        FROM identity_values
        ON CONFLICT (property_offering_identity_key) DO UPDATE SET
            property_offering_headline = CASE
                WHEN EXCLUDED.property_offering_last_seen_at >= COALESCE(property_offerings.property_offering_last_seen_at, '-infinity'::timestamptz) THEN EXCLUDED.property_offering_headline
                ELSE property_offerings.property_offering_headline
            END,
            property_offering_asking_price = COALESCE(EXCLUDED.property_offering_asking_price, property_offerings.property_offering_asking_price),
            property_offering_debt_free_price = COALESCE(EXCLUDED.property_offering_debt_free_price, property_offerings.property_offering_debt_free_price),
            property_offering_price_per_m2 = COALESCE(EXCLUDED.property_offering_price_per_m2, property_offerings.property_offering_price_per_m2),
            property_offering_first_seen_at = LEAST(COALESCE(property_offerings.property_offering_first_seen_at, EXCLUDED.property_offering_first_seen_at), COALESCE(EXCLUDED.property_offering_first_seen_at, property_offerings.property_offering_first_seen_at)),
            property_offering_last_seen_at = GREATEST(COALESCE(property_offerings.property_offering_last_seen_at, EXCLUDED.property_offering_last_seen_at), COALESCE(EXCLUDED.property_offering_last_seen_at, property_offerings.property_offering_last_seen_at)),
            primary_sale_listing_id = CASE
                WHEN EXCLUDED.property_offering_last_seen_at >= COALESCE(property_offerings.property_offering_last_seen_at, '-infinity'::timestamptz) THEN EXCLUDED.primary_sale_listing_id
                ELSE property_offerings.primary_sale_listing_id
            END,
            property_offering_updated_at = now()
        RETURNING property_offering_id
    )
    SELECT property_offering_id INTO offering_id FROM inserted;
    INSERT INTO public.property_offering_sources (
        property_offering_id,
        sale_listing_id,
        property_offering_source_link_status,
        property_offering_source_link_method,
        property_offering_source_link_score,
        property_offering_source_link_reasons,
        property_offering_source_updated_at
    )
    VALUES (
        offering_id,
        listing_id,
        'confirmed',
        link_method,
        120,
        jsonb_build_object('matched_by', 'canonical_identity_key'),
        now()
    )
    ON CONFLICT (sale_listing_id) WHERE property_offering_source_link_status <> 'rejected' DO UPDATE SET
        property_offering_id = CASE
            WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_id
            ELSE EXCLUDED.property_offering_id
        END,
        property_offering_source_link_status = CASE
            WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_status
            ELSE EXCLUDED.property_offering_source_link_status
        END,
        property_offering_source_link_method = CASE
            WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_method
            ELSE EXCLUDED.property_offering_source_link_method
        END,
        property_offering_source_link_score = CASE
            WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_score
            ELSE EXCLUDED.property_offering_source_link_score
        END,
        property_offering_source_link_reasons = CASE
            WHEN property_offering_sources.property_offering_source_link_method = ANY (ARRAY['manual'::text, 'source_match_auto'::text]) THEN property_offering_sources.property_offering_source_link_reasons
            ELSE EXCLUDED.property_offering_source_link_reasons
        END,
        property_offering_source_updated_at = now();
    INSERT INTO public.property_offering_transactions (
        property_offering_id,
        prices_transaction_id,
        property_offering_transaction_link_status,
        property_offering_transaction_link_method,
        property_offering_transaction_link_score,
        property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at
    )
    SELECT
        COALESCE(pos.property_offering_id, offering_id),
        sl.prices_transaction_id,
        COALESCE(sl.sale_listing_prices_match_status, 'confirmed'),
        'sync_auto',
        COALESCE(c.sale_listing_prices_transaction_match_score, 120),
        COALESCE(c.sale_listing_prices_transaction_match_reasons, '{}'::jsonb),
        now()
    FROM public.sale_listings sl
    LEFT JOIN public.property_offering_sources pos ON pos.sale_listing_id = sl.sale_listing_id
        AND pos.property_offering_source_link_status <> 'rejected'
    LEFT JOIN LATERAL (
        SELECT
            c.sale_listing_prices_transaction_match_score,
            c.sale_listing_prices_transaction_match_reasons
        FROM public.sale_listing_prices_transaction_match_candidates c
        WHERE c.sale_listing_id = sl.sale_listing_id
            AND c.prices_transaction_id = sl.prices_transaction_id
        ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
        LIMIT 1
    ) c ON true
    WHERE sl.sale_listing_id = listing_id
        AND sl.prices_transaction_id IS NOT NULL
    ON CONFLICT (property_offering_id, prices_transaction_id) DO UPDATE SET
        property_offering_transaction_link_status = EXCLUDED.property_offering_transaction_link_status,
        property_offering_transaction_link_method = EXCLUDED.property_offering_transaction_link_method,
        property_offering_transaction_link_score = EXCLUDED.property_offering_transaction_link_score,
        property_offering_transaction_link_reasons = EXCLUDED.property_offering_transaction_link_reasons,
        property_offering_transaction_updated_at = now();
    RETURN offering_id;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__sync_canonical_property_for_sale_listing_trigger()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    PERFORM public.fnc__sync_canonical_property_for_sale_listing(NEW.sale_listing_id, 'sync_auto');
    RETURN NEW;
END;
$$;
DROP TRIGGER IF EXISTS trg__sync_canonical_property_for_sale_listing ON public.sale_listings;
CREATE TRIGGER trg__sync_canonical_property_for_sale_listing
AFTER INSERT OR UPDATE OF sale_listing_source_provider, sale_listing_source_kind, sale_listing_native_id, sale_listing_headline, sale_listing_postal_norm, sale_listing_city_norm, sale_listing_address_norm, sale_listing_building_match_key, sale_listing_unit_match_key, sale_listing_floor_level, sale_listing_area_value, sale_listing_rooms_count, sale_listing_room_layout, sale_listing_asking_price, sale_listing_debt_free_price, sale_listing_price_per_m2, sale_listing_first_seen_at, sale_listing_last_seen_at, sale_listing_build_year, sale_listing_total_floors, sale_listing_elevator, sale_listing_energy_efficiency_label, sale_listing_prices_match_status, prices_transaction_id ON public.sale_listings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_canonical_property_for_sale_listing_trigger();
CREATE TEMP TABLE canonical_property_backfill ON COMMIT DROP AS
SELECT
    sl.sale_listing_id,
    sl.prices_transaction_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.sale_listing_headline,
    sl.sale_listing_postal_norm,
    sl.sale_listing_city_norm,
    sl.sale_listing_address_norm,
    sl.sale_listing_unit_match_key,
    sl.sale_listing_floor_level,
    sl.sale_listing_area_value,
    sl.sale_listing_rooms_count,
    sl.sale_listing_room_layout,
    sl.sale_listing_asking_price,
    sl.sale_listing_debt_free_price,
    sl.sale_listing_price_per_m2,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_created_at,
    sl.sale_listing_build_year,
    sl.sale_listing_total_floors,
    sl.sale_listing_elevator,
    sl.sale_listing_energy_efficiency_label,
    sl.sale_listing_prices_match_status,
    business_id,
    housing_company,
    provider_building_id,
    building_key,
    building_key || ':unit:' || COALESCE(
        public.fnc__canonical_identity_part(sl.sale_listing_unit_match_key),
        public.fnc__canonical_identity_part(concat_ws('|', sl.sale_listing_floor_level::text, sl.sale_listing_area_value::text, public.fnc__layout_match_key(sl.sale_listing_room_layout))),
        sl.sale_listing_id::text
    ) AS unit_key,
    building_key || ':unit:' || COALESCE(
        public.fnc__canonical_identity_part(sl.sale_listing_unit_match_key),
        public.fnc__canonical_identity_part(concat_ws('|', sl.sale_listing_floor_level::text, sl.sale_listing_area_value::text, public.fnc__layout_match_key(sl.sale_listing_room_layout))),
        sl.sale_listing_id::text
    ) || ':sale:' || COALESCE(sl.sale_listing_debt_free_price::text, sl.sale_listing_asking_price::text, 'unknown') AS offering_key
FROM public.sale_listings sl
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
LEFT JOIN public.frontdoor_building_announcements fba ON fba.frontdoor_building_announcement_id = sl.frontdoor_building_announcement_id
LEFT JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
CROSS JOIN LATERAL (
    SELECT
        COALESCE(sa.shortcut_ad_data #>> '{adData,housingCompanyBusinessId}', fa.frontdoor_ad_data #>> '{property,housingCompany,businessId}', fb.frontdoor_building_business_id) AS business_id,
        COALESCE(sa.shortcut_ad_data #>> '{adData,housingCompanyName}', sb.shortcut_building_housing_company, fa.frontdoor_ad_data #>> '{property,housingCompany,name}', fb.frontdoor_building_company_name) AS housing_company,
        COALESCE(sa.shortcut_ad_data #>> '{buildingId}', sa.shortcut_ad_data #>> '{adData,buildingId}', sb.shortcut_building_external_id::text, sb.shortcut_building_building_id, fa.frontdoor_ad_data #>> '{property,housingCompany,id}', fb.frontdoor_building_housing_company_id::text, fba.frontdoor_building_id::text) AS provider_building_id
) source_ids
CROSS JOIN LATERAL (
    SELECT COALESCE(
        'business:' || public.fnc__canonical_identity_part(source_ids.business_id),
        'provider_building:' || sl.sale_listing_source_provider || ':' || public.fnc__canonical_identity_part(source_ids.provider_building_id),
        'address:' || public.fnc__canonical_identity_part(concat_ws('|', sl.sale_listing_postal_norm, sl.sale_listing_city_norm, sl.sale_listing_building_match_key, source_ids.housing_company)),
        'source:' || sl.sale_listing_source_provider || ':' || sl.sale_listing_source_kind || ':' || sl.sale_listing_native_id
    ) AS building_key
) identities;
CREATE INDEX canonical_property_backfill_building_idx ON canonical_property_backfill (building_key, sale_listing_last_seen_at DESC NULLS LAST, sale_listing_created_at DESC);
CREATE INDEX canonical_property_backfill_unit_idx ON canonical_property_backfill (unit_key, sale_listing_last_seen_at DESC NULLS LAST, sale_listing_created_at DESC);
CREATE INDEX canonical_property_backfill_offering_idx ON canonical_property_backfill (offering_key, sale_listing_last_seen_at DESC NULLS LAST, sale_listing_created_at DESC);
CREATE INDEX canonical_property_backfill_sale_listing_idx ON canonical_property_backfill (sale_listing_id);
ANALYZE canonical_property_backfill;
INSERT INTO public.property_buildings (
    property_building_identity_key,
    property_building_postal_norm,
    property_building_city_norm,
    property_building_address_norm,
    property_building_housing_company,
    property_building_business_id,
    property_building_build_year,
    property_building_floor_count,
    property_building_elevator,
    property_building_energy_efficiency_label,
    property_building_match_reasons,
    property_building_updated_at
)
SELECT
    building_key,
    sale_listing_postal_norm,
    sale_listing_city_norm,
    sale_listing_address_norm,
    housing_company,
    business_id,
    sale_listing_build_year,
    sale_listing_total_floors,
    sale_listing_elevator,
    sale_listing_energy_efficiency_label,
    jsonb_build_object('source', sale_listing_source_provider, 'provider_building_id', provider_building_id, 'backfilled', true),
    now()
FROM (
    SELECT DISTINCT ON (building_key)
        *
    FROM canonical_property_backfill
    ORDER BY building_key, sale_listing_last_seen_at DESC NULLS LAST, sale_listing_created_at DESC
) selected
ON CONFLICT (property_building_identity_key) DO NOTHING;
INSERT INTO public.property_units (
    property_building_id,
    property_unit_identity_key,
    property_unit_address_norm,
    property_unit_floor_level,
    property_unit_area_value,
    property_unit_rooms_count,
    property_unit_room_layout,
    property_unit_layout_match_key,
    property_unit_match_reasons,
    property_unit_updated_at
)
SELECT
    pb.property_building_id,
    selected.unit_key,
    selected.sale_listing_address_norm,
    selected.sale_listing_floor_level,
    selected.sale_listing_area_value,
    selected.sale_listing_rooms_count,
    selected.sale_listing_room_layout,
    public.fnc__layout_match_key(selected.sale_listing_room_layout),
    jsonb_build_object('source_listing_id', selected.sale_listing_id, 'backfilled', true),
    now()
FROM (
    SELECT DISTINCT ON (unit_key)
        *
    FROM canonical_property_backfill
    ORDER BY unit_key, sale_listing_last_seen_at DESC NULLS LAST, sale_listing_created_at DESC
) selected
JOIN public.property_buildings pb ON pb.property_building_identity_key = selected.building_key
ON CONFLICT (property_unit_identity_key) DO NOTHING;
INSERT INTO public.property_offerings (
    property_unit_id,
    property_offering_identity_key,
    property_offering_type,
    property_offering_headline,
    property_offering_asking_price,
    property_offering_debt_free_price,
    property_offering_price_per_m2,
    property_offering_first_seen_at,
    property_offering_last_seen_at,
    property_offering_status,
    primary_sale_listing_id,
    property_offering_match_reasons,
    property_offering_updated_at
)
SELECT
    pu.property_unit_id,
    selected.offering_key,
    'sale',
    selected.sale_listing_headline,
    selected.sale_listing_asking_price,
    selected.sale_listing_debt_free_price,
    selected.sale_listing_price_per_m2,
    selected.sale_listing_first_seen_at,
    selected.sale_listing_last_seen_at,
    selected.sale_listing_prices_match_status,
    selected.sale_listing_id,
    jsonb_build_object('source_listing_id', selected.sale_listing_id, 'identity_key', selected.offering_key, 'backfilled', true),
    now()
FROM (
    SELECT DISTINCT ON (offering_key)
        *
    FROM canonical_property_backfill
    ORDER BY offering_key, sale_listing_last_seen_at DESC NULLS LAST, sale_listing_created_at DESC
) selected
JOIN public.property_units pu ON pu.property_unit_identity_key = selected.unit_key
ON CONFLICT (property_offering_identity_key) DO NOTHING;
INSERT INTO public.property_offering_sources (
    property_offering_id,
    sale_listing_id,
    property_offering_source_link_status,
    property_offering_source_link_method,
    property_offering_source_link_score,
    property_offering_source_link_reasons,
    property_offering_source_updated_at
)
SELECT
    po.property_offering_id,
    backfill.sale_listing_id,
    'confirmed',
    'backfill_auto',
    120,
    jsonb_build_object('matched_by', 'canonical_identity_key', 'backfilled', true),
    now()
FROM canonical_property_backfill backfill
JOIN public.property_offerings po ON po.property_offering_identity_key = backfill.offering_key
ON CONFLICT (sale_listing_id) WHERE property_offering_source_link_status <> 'rejected' DO NOTHING;
INSERT INTO public.property_offering_transactions (
    property_offering_id,
    prices_transaction_id,
    property_offering_transaction_link_status,
    property_offering_transaction_link_method,
    property_offering_transaction_link_score,
    property_offering_transaction_link_reasons,
    property_offering_transaction_updated_at
)
SELECT
    pos.property_offering_id,
    sl.prices_transaction_id,
    COALESCE(sl.sale_listing_prices_match_status, 'confirmed'),
    'backfill_auto',
    COALESCE(c.sale_listing_prices_transaction_match_score, 120),
    COALESCE(c.sale_listing_prices_transaction_match_reasons, '{}'::jsonb),
    now()
FROM public.property_offering_sources pos
JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
LEFT JOIN LATERAL (
    SELECT
        c.sale_listing_prices_transaction_match_score,
        c.sale_listing_prices_transaction_match_reasons
    FROM public.sale_listing_prices_transaction_match_candidates c
    WHERE c.sale_listing_id = sl.sale_listing_id
        AND c.prices_transaction_id = sl.prices_transaction_id
    ORDER BY c.sale_listing_prices_transaction_match_created_at DESC
    LIMIT 1
) c ON true
WHERE sl.prices_transaction_id IS NOT NULL
ON CONFLICT (property_offering_id, prices_transaction_id) DO NOTHING;
