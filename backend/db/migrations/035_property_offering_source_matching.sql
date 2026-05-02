ALTER TABLE public.property_offering_sources
DROP CONSTRAINT IF EXISTS property_offering_sources_method_check;
ALTER TABLE public.property_offering_sources
ADD CONSTRAINT property_offering_sources_method_check CHECK (
    property_offering_source_link_method = ANY (ARRAY['backfill_auto'::text, 'sync_auto'::text, 'source_match_auto'::text, 'manual'::text])
);
CREATE TABLE IF NOT EXISTS public.property_offering_source_match_runs (
    property_offering_source_match_run_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_offering_source_match_run_mode text NOT NULL,
    property_offering_source_match_score_threshold integer DEFAULT 95 NOT NULL,
    property_offering_source_match_competitor_margin integer DEFAULT 10 NOT NULL,
    property_offering_source_match_candidates_count integer DEFAULT 0 NOT NULL,
    property_offering_source_match_auto_linked_count integer DEFAULT 0 NOT NULL,
    property_offering_source_match_ambiguous_count integer DEFAULT 0 NOT NULL,
    property_offering_source_match_started_at timestamp with time zone DEFAULT now() NOT NULL,
    property_offering_source_match_finished_at timestamp with time zone,
    CONSTRAINT property_offering_source_match_run_mode_check CHECK (property_offering_source_match_run_mode = ANY (ARRAY['dry_run'::text, 'auto_link_safe'::text])),
    CONSTRAINT property_offering_source_match_threshold_check CHECK (property_offering_source_match_score_threshold >= 0),
    CONSTRAINT property_offering_source_match_margin_check CHECK (property_offering_source_match_competitor_margin >= 0)
);
CREATE TABLE IF NOT EXISTS public.property_offering_source_match_candidates (
    property_offering_source_match_candidate_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    property_offering_source_match_run_id uuid NOT NULL REFERENCES public.property_offering_source_match_runs(property_offering_source_match_run_id) ON DELETE CASCADE,
    source_sale_listing_id uuid NOT NULL REFERENCES public.sale_listings(sale_listing_id) ON DELETE CASCADE,
    source_property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    target_property_offering_id uuid NOT NULL REFERENCES public.property_offerings(property_offering_id) ON DELETE CASCADE,
    target_sale_listing_id uuid NOT NULL REFERENCES public.sale_listings(sale_listing_id) ON DELETE CASCADE,
    property_offering_source_match_score integer NOT NULL,
    property_offering_source_match_confidence text NOT NULL,
    property_offering_source_match_status text DEFAULT 'candidate'::text NOT NULL,
    property_offering_source_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    property_offering_source_match_price_delta_percent double precision,
    property_offering_source_match_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT property_offering_source_match_candidate_unique UNIQUE (property_offering_source_match_run_id, source_sale_listing_id, target_property_offering_id),
    CONSTRAINT property_offering_source_match_confidence_check CHECK (property_offering_source_match_confidence = ANY (ARRAY['high'::text, 'medium'::text, 'low'::text])),
    CONSTRAINT property_offering_source_match_status_check CHECK (property_offering_source_match_status = ANY (ARRAY['candidate'::text, 'auto_linked'::text, 'ambiguous'::text, 'rejected'::text]))
);
CREATE INDEX IF NOT EXISTS idx_property_offering_source_match_candidates_run_status ON public.property_offering_source_match_candidates (property_offering_source_match_run_id, property_offering_source_match_status);
CREATE INDEX IF NOT EXISTS idx_property_offering_source_match_candidates_source_score ON public.property_offering_source_match_candidates (source_sale_listing_id, property_offering_source_match_score DESC);
CREATE INDEX IF NOT EXISTS idx_property_offering_source_match_candidates_target_score ON public.property_offering_source_match_candidates (target_property_offering_id, property_offering_source_match_score DESC);
ALTER TABLE public.sale_listings
ADD COLUMN IF NOT EXISTS sale_listing_source_match_status text,
ADD COLUMN IF NOT EXISTS sale_listing_source_match_next_attempt_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS sale_listing_source_match_last_attempted_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS sale_listing_source_match_attempt_count integer DEFAULT 0 NOT NULL,
ADD COLUMN IF NOT EXISTS sale_listing_source_match_run_id uuid REFERENCES public.property_offering_source_match_runs(property_offering_source_match_run_id) ON DELETE SET NULL,
ADD CONSTRAINT sale_listings_source_match_status_check CHECK (
    sale_listing_source_match_status IS NULL
    OR sale_listing_source_match_status = ANY (ARRAY['pending'::text, 'deferred'::text, 'auto_linked'::text, 'needs_review'::text, 'manual_linked'::text, 'rejected'::text, 'noop'::text])
);
CREATE INDEX IF NOT EXISTS idx_sale_listings_source_match_queue ON public.sale_listings (sale_listing_source_match_status, sale_listing_source_match_next_attempt_at)
WHERE sale_listing_source_kind = 'ad';
CREATE OR REPLACE FUNCTION public.fnc__refresh_property_offering_source_matches(auto_link_safe boolean DEFAULT false, score_threshold integer DEFAULT 95, competitor_margin integer DEFAULT 10, listing_public_id text DEFAULT NULL)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    run_id uuid;
    candidate_count integer := 0;
    auto_linked_count integer := 0;
    ambiguous_count integer := 0;
BEGIN
    IF score_threshold < 0 THEN
        RAISE EXCEPTION 'score_threshold must be non-negative';
    END IF;
    IF competitor_margin < 0 THEN
        RAISE EXCEPTION 'competitor_margin must be non-negative';
    END IF;
    INSERT INTO public.property_offering_source_match_runs (
        property_offering_source_match_run_mode,
        property_offering_source_match_score_threshold,
        property_offering_source_match_competitor_margin
    )
    VALUES (
        CASE WHEN auto_link_safe THEN 'auto_link_safe' ELSE 'dry_run' END,
        score_threshold,
        competitor_margin
    )
    RETURNING property_offering_source_match_run_id INTO run_id;
    WITH source_links AS (
        SELECT
            pos.property_offering_id AS source_property_offering_id,
            sl.sale_listing_id AS source_sale_listing_id,
            sl.sale_listing_public_id AS source_public_id,
            sl.sale_listing_source_provider AS source_provider,
            sl.sale_listing_postal_norm AS source_postal_norm,
            sl.sale_listing_city_norm AS source_city_norm,
            sl.sale_listing_address_norm AS source_address_norm,
            sl.sale_listing_street_name_norm AS source_street_name_norm,
            sl.sale_listing_street_match_key AS source_street_match_key,
            sl.sale_listing_building_match_key AS source_building_match_key,
            sl.sale_listing_unit_match_key AS source_unit_match_key,
            sl.sale_listing_floor_level AS source_floor_level,
            sl.sale_listing_area_value AS source_area_value,
            sl.sale_listing_room_layout AS source_room_layout,
            public.fnc__layout_match_key(sl.sale_listing_room_layout) AS source_layout_match_key,
            sl.sale_listing_asking_price AS source_asking_price,
            sl.sale_listing_debt_free_price AS source_debt_free_price,
            sl.sale_listing_first_seen_at AS source_first_seen_at,
            sl.sale_listing_last_seen_at AS source_last_seen_at,
            sl.sale_listing_build_year AS source_build_year,
            sl.sale_listing_elevator AS source_elevator,
            sl.sale_listing_plot_owned AS source_plot_owned,
            sl.sale_listing_energy_efficiency_match_code AS source_energy_match_code,
            sl.sale_listing_condition AS source_condition,
            sl.prices_transaction_id AS source_prices_transaction_id,
            spb.property_building_identity_key AS source_building_identity_key,
            spu.property_unit_identity_key AS source_unit_identity_key
        FROM public.property_offering_sources pos
        JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
        JOIN public.property_offerings spo ON spo.property_offering_id = pos.property_offering_id
        JOIN public.property_units spu ON spu.property_unit_id = spo.property_unit_id
        JOIN public.property_buildings spb ON spb.property_building_id = spu.property_building_id
        WHERE pos.property_offering_source_link_status <> 'rejected'
            AND pos.property_offering_source_link_method <> 'manual'
            AND sl.sale_listing_source_kind = 'ad'
            AND (listing_public_id IS NULL OR sl.sale_listing_public_id = listing_public_id)
    ),
    target_links AS (
        SELECT
            pos.property_offering_id AS target_property_offering_id,
            sl.sale_listing_id AS target_sale_listing_id,
            sl.sale_listing_source_provider AS target_provider,
            sl.sale_listing_postal_norm AS target_postal_norm,
            sl.sale_listing_city_norm AS target_city_norm,
            sl.sale_listing_address_norm AS target_address_norm,
            sl.sale_listing_street_name_norm AS target_street_name_norm,
            sl.sale_listing_street_match_key AS target_street_match_key,
            sl.sale_listing_building_match_key AS target_building_match_key,
            sl.sale_listing_unit_match_key AS target_unit_match_key,
            sl.sale_listing_floor_level AS target_floor_level,
            sl.sale_listing_area_value AS target_area_value,
            sl.sale_listing_room_layout AS target_room_layout,
            public.fnc__layout_match_key(sl.sale_listing_room_layout) AS target_layout_match_key,
            sl.sale_listing_asking_price AS target_asking_price,
            sl.sale_listing_debt_free_price AS target_debt_free_price,
            sl.sale_listing_first_seen_at AS target_first_seen_at,
            sl.sale_listing_last_seen_at AS target_last_seen_at,
            sl.sale_listing_build_year AS target_build_year,
            sl.sale_listing_elevator AS target_elevator,
            sl.sale_listing_plot_owned AS target_plot_owned,
            sl.sale_listing_energy_efficiency_match_code AS target_energy_match_code,
            sl.sale_listing_condition AS target_condition,
            sl.prices_transaction_id AS target_prices_transaction_id,
            tpb.property_building_identity_key AS target_building_identity_key,
            tpu.property_unit_identity_key AS target_unit_identity_key
        FROM public.property_offering_sources pos
        JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
        JOIN public.property_offerings tpo ON tpo.property_offering_id = pos.property_offering_id
        JOIN public.property_units tpu ON tpu.property_unit_id = tpo.property_unit_id
        JOIN public.property_buildings tpb ON tpb.property_building_id = tpu.property_building_id
        WHERE pos.property_offering_source_link_status <> 'rejected'
    ),
    candidate_base AS (
        SELECT
            src.source_sale_listing_id,
            src.source_property_offering_id,
            tgt.target_property_offering_id,
            tgt.target_sale_listing_id,
            CASE WHEN src.source_postal_norm IS NOT NULL AND src.source_postal_norm = tgt.target_postal_norm THEN 15 ELSE 0 END AS postal_score,
            CASE WHEN src.source_city_norm IS NOT NULL AND src.source_city_norm = tgt.target_city_norm THEN 5 ELSE 0 END AS city_score,
            CASE WHEN src.source_address_norm IS NOT NULL AND src.source_address_norm = tgt.target_address_norm THEN 25 ELSE 0 END AS address_score,
            CASE WHEN src.source_street_name_norm IS NOT NULL AND src.source_street_name_norm = tgt.target_street_name_norm THEN 10 ELSE 0 END AS street_name_score,
            CASE WHEN src.source_street_match_key IS NOT NULL AND src.source_street_match_key = tgt.target_street_match_key THEN 12 ELSE 0 END AS street_score,
            CASE WHEN src.source_building_match_key IS NOT NULL AND src.source_building_match_key = tgt.target_building_match_key THEN 20 ELSE 0 END AS building_score,
            CASE WHEN src.source_unit_match_key IS NOT NULL AND src.source_unit_match_key = tgt.target_unit_match_key THEN 30 ELSE 0 END AS unit_score,
            CASE WHEN src.source_prices_transaction_id IS NOT NULL AND src.source_prices_transaction_id = tgt.target_prices_transaction_id THEN 80 ELSE 0 END AS transaction_score,
            CASE
                WHEN src.source_street_match_key IS NOT NULL
                    AND src.source_street_match_key = tgt.target_street_match_key
                    AND src.source_area_value IS NOT NULL
                    AND tgt.target_area_value IS NOT NULL
                    AND abs(src.source_area_value - tgt.target_area_value) <= 1.0
                    AND src.source_layout_match_key IS NOT NULL
                    AND src.source_layout_match_key = tgt.target_layout_match_key THEN 35
                ELSE 0
            END AS street_area_layout_score,
            CASE
                WHEN src.source_street_match_key IS NOT NULL
                    AND src.source_street_match_key = tgt.target_street_match_key
                    AND src.source_area_value IS NOT NULL
                    AND tgt.target_area_value IS NOT NULL
                    AND abs(src.source_area_value - tgt.target_area_value) <= 1.0
                    AND src.source_floor_level IS NOT NULL
                    AND src.source_floor_level = tgt.target_floor_level
                    AND COALESCE(src.source_debt_free_price, src.source_asking_price) IS NOT NULL
                    AND COALESCE(tgt.target_debt_free_price, tgt.target_asking_price) IS NOT NULL
                    AND abs(COALESCE(src.source_debt_free_price, src.source_asking_price)::double precision - COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision) / GREATEST(COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision, 1) <= 0.05 THEN 25
                ELSE 0
            END AS street_area_floor_price_score,
            CASE
                WHEN src.source_area_value IS NOT NULL AND tgt.target_area_value IS NOT NULL AND abs(src.source_area_value - tgt.target_area_value) <= 0.2 THEN 20
                WHEN src.source_area_value IS NOT NULL AND tgt.target_area_value IS NOT NULL AND abs(src.source_area_value - tgt.target_area_value) <= 1.0 THEN 15
                WHEN src.source_area_value IS NOT NULL AND tgt.target_area_value IS NOT NULL AND abs(src.source_area_value - tgt.target_area_value) <= 2.0 THEN 8
                ELSE 0
            END AS area_score,
            CASE WHEN src.source_layout_match_key IS NOT NULL AND src.source_layout_match_key = tgt.target_layout_match_key THEN 12 ELSE 0 END AS layout_score,
            CASE WHEN src.source_floor_level IS NOT NULL AND src.source_floor_level = tgt.target_floor_level THEN 8 ELSE 0 END AS floor_score,
            CASE WHEN src.source_build_year IS NOT NULL AND src.source_build_year = tgt.target_build_year THEN 6 ELSE 0 END AS build_year_score,
            CASE WHEN src.source_elevator IS NOT NULL AND src.source_elevator = tgt.target_elevator THEN 3 ELSE 0 END AS elevator_score,
            CASE WHEN src.source_plot_owned IS NOT NULL AND src.source_plot_owned = tgt.target_plot_owned THEN 3 ELSE 0 END AS plot_score,
            CASE WHEN src.source_energy_match_code IS NOT NULL AND src.source_energy_match_code = tgt.target_energy_match_code THEN 4 ELSE 0 END AS energy_score,
            CASE WHEN src.source_condition IS NOT NULL AND public.fnc__match_alias_key(src.source_condition) = public.fnc__match_alias_key(tgt.target_condition) THEN 3 ELSE 0 END AS condition_score,
            CASE
                WHEN COALESCE(src.source_debt_free_price, src.source_asking_price) IS NOT NULL
                    AND COALESCE(tgt.target_debt_free_price, tgt.target_asking_price) IS NOT NULL
                    AND abs(COALESCE(src.source_debt_free_price, src.source_asking_price)::double precision - COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision) / GREATEST(COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision, 1) <= 0.02 THEN 10
                WHEN COALESCE(src.source_debt_free_price, src.source_asking_price) IS NOT NULL
                    AND COALESCE(tgt.target_debt_free_price, tgt.target_asking_price) IS NOT NULL
                    AND abs(COALESCE(src.source_debt_free_price, src.source_asking_price)::double precision - COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision) / GREATEST(COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision, 1) <= 0.05 THEN 5
                ELSE 0
            END AS price_score,
            CASE
                WHEN src.source_last_seen_at IS NOT NULL AND tgt.target_last_seen_at IS NOT NULL AND abs(extract(epoch FROM src.source_last_seen_at - tgt.target_last_seen_at)) <= 30 * 24 * 60 * 60 THEN 6
                WHEN src.source_first_seen_at IS NOT NULL AND tgt.target_first_seen_at IS NOT NULL AND abs(extract(epoch FROM src.source_first_seen_at - tgt.target_first_seen_at)) <= 30 * 24 * 60 * 60 THEN 4
                ELSE 0
            END AS temporal_score,
            CASE
                WHEN COALESCE(src.source_debt_free_price, src.source_asking_price) IS NOT NULL
                    AND COALESCE(tgt.target_debt_free_price, tgt.target_asking_price) IS NOT NULL THEN
                    abs(COALESCE(src.source_debt_free_price, src.source_asking_price)::double precision - COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision) / GREATEST(COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision, 1)
                ELSE NULL
            END AS price_delta_percent,
            jsonb_build_object(
                'source_provider', src.source_provider,
                'target_provider', tgt.target_provider,
                'postal', jsonb_build_object('source', src.source_postal_norm, 'target', tgt.target_postal_norm),
                'address', jsonb_build_object('source', src.source_address_norm, 'target', tgt.target_address_norm),
                'street_name', jsonb_build_object('source', src.source_street_name_norm, 'target', tgt.target_street_name_norm),
                'street_match_key', jsonb_build_object('source', src.source_street_match_key, 'target', tgt.target_street_match_key),
                'building_match_key', jsonb_build_object('source', src.source_building_match_key, 'target', tgt.target_building_match_key),
                'unit_match_key', jsonb_build_object('source', src.source_unit_match_key, 'target', tgt.target_unit_match_key),
                'prices_transaction_id', jsonb_build_object('source', src.source_prices_transaction_id, 'target', tgt.target_prices_transaction_id),
                'area', jsonb_build_object('source', src.source_area_value, 'target', tgt.target_area_value),
                'layout', jsonb_build_object('source', src.source_room_layout, 'target', tgt.target_room_layout),
                'floor', jsonb_build_object('source', src.source_floor_level, 'target', tgt.target_floor_level),
                'build_year', jsonb_build_object('source', src.source_build_year, 'target', tgt.target_build_year),
                'elevator', jsonb_build_object('source', src.source_elevator, 'target', tgt.target_elevator),
                'plot_owned', jsonb_build_object('source', src.source_plot_owned, 'target', tgt.target_plot_owned),
                'energy', jsonb_build_object('source', src.source_energy_match_code, 'target', tgt.target_energy_match_code),
                'condition', jsonb_build_object('source', src.source_condition, 'target', tgt.target_condition)
            ) AS reasons
        FROM source_links src
        JOIN target_links tgt ON tgt.target_provider <> src.source_provider
            AND tgt.target_property_offering_id <> src.source_property_offering_id
            AND tgt.target_property_offering_id::text < src.source_property_offering_id::text
            AND (
                (src.source_postal_norm IS NOT NULL AND src.source_postal_norm = tgt.target_postal_norm AND src.source_unit_match_key IS NOT NULL AND src.source_unit_match_key = tgt.target_unit_match_key)
                OR (src.source_postal_norm IS NOT NULL AND src.source_postal_norm = tgt.target_postal_norm AND src.source_address_norm IS NOT NULL AND src.source_address_norm = tgt.target_address_norm)
                OR (src.source_prices_transaction_id IS NOT NULL AND src.source_prices_transaction_id = tgt.target_prices_transaction_id)
                OR (
                    src.source_postal_norm IS NOT NULL
                    AND src.source_postal_norm = tgt.target_postal_norm
                    AND src.source_building_match_key IS NOT NULL
                    AND src.source_building_match_key = tgt.target_building_match_key
                    AND src.source_area_value IS NOT NULL
                    AND tgt.target_area_value IS NOT NULL
                    AND abs(src.source_area_value - tgt.target_area_value) <= 2.0
                )
                OR (
                    src.source_postal_norm IS NOT NULL
                    AND src.source_postal_norm = tgt.target_postal_norm
                    AND src.source_street_match_key IS NOT NULL
                    AND src.source_street_match_key = tgt.target_street_match_key
                    AND src.source_area_value IS NOT NULL
                    AND tgt.target_area_value IS NOT NULL
                    AND abs(src.source_area_value - tgt.target_area_value) <= 1.0
                    AND (
                        (src.source_layout_match_key IS NOT NULL AND src.source_layout_match_key = tgt.target_layout_match_key)
                        OR (
                            COALESCE(src.source_debt_free_price, src.source_asking_price) IS NOT NULL
                            AND COALESCE(tgt.target_debt_free_price, tgt.target_asking_price) IS NOT NULL
                            AND abs(COALESCE(src.source_debt_free_price, src.source_asking_price)::double precision - COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision) / GREATEST(COALESCE(tgt.target_debt_free_price, tgt.target_asking_price)::double precision, 1) <= 0.05
                        )
                    )
                )
            )
    ),
    scored AS (
        SELECT
            source_sale_listing_id,
            source_property_offering_id,
            target_property_offering_id,
            target_sale_listing_id,
            (
                postal_score + city_score + address_score + street_name_score + street_score + building_score + unit_score + transaction_score + street_area_layout_score + street_area_floor_price_score + area_score + layout_score + floor_score + build_year_score + elevator_score + plot_score + energy_score + condition_score + price_score + temporal_score
            )::integer AS score,
            price_delta_percent,
            reasons || jsonb_build_object(
                'score', jsonb_build_object(
                    'postal', postal_score,
                    'city', city_score,
                    'address', address_score,
                    'street_name', street_name_score,
                    'street', street_score,
                    'building', building_score,
                    'unit', unit_score,
                    'transaction', transaction_score,
                    'street_area_layout', street_area_layout_score,
                    'street_area_floor_price', street_area_floor_price_score,
                    'area', area_score,
                    'layout', layout_score,
                    'floor', floor_score,
                    'build_year', build_year_score,
                    'elevator', elevator_score,
                    'plot', plot_score,
                    'energy', energy_score,
                    'condition', condition_score,
                    'price', price_score,
                    'temporal', temporal_score
                ),
                'price_delta_percent', price_delta_percent
            ) AS reasons
        FROM candidate_base
    )
    INSERT INTO public.property_offering_source_match_candidates (
        property_offering_source_match_run_id,
        source_sale_listing_id,
        source_property_offering_id,
        target_property_offering_id,
        target_sale_listing_id,
        property_offering_source_match_score,
        property_offering_source_match_confidence,
        property_offering_source_match_reasons,
        property_offering_source_match_price_delta_percent
    )
    SELECT
        run_id,
        source_sale_listing_id,
        source_property_offering_id,
        target_property_offering_id,
        target_sale_listing_id,
        score,
        CASE WHEN score >= score_threshold THEN 'high' WHEN score >= 75 THEN 'medium' ELSE 'low' END,
        reasons,
        price_delta_percent
    FROM (
        SELECT DISTINCT ON (source_sale_listing_id, target_property_offering_id)
            source_sale_listing_id,
            source_property_offering_id,
            target_property_offering_id,
            target_sale_listing_id,
            score,
            reasons,
            price_delta_percent
        FROM scored
        WHERE score >= 60
        ORDER BY source_sale_listing_id, target_property_offering_id, score DESC, price_delta_percent ASC NULLS LAST, target_sale_listing_id
    ) deduped;
    GET DIAGNOSTICS candidate_count = ROW_COUNT;
    IF auto_link_safe THEN
        WITH ranked AS (
            SELECT
                c.property_offering_source_match_candidate_id,
                c.source_sale_listing_id,
                c.source_property_offering_id,
                c.target_property_offering_id,
                c.property_offering_source_match_score,
                c.property_offering_source_match_reasons,
                c.property_offering_source_match_price_delta_percent,
                row_number() OVER (PARTITION BY c.source_sale_listing_id ORDER BY c.property_offering_source_match_score DESC, c.property_offering_source_match_price_delta_percent ASC NULLS LAST, c.property_offering_source_match_candidate_id) AS source_rank,
                lead(c.property_offering_source_match_score) OVER (PARTITION BY c.source_sale_listing_id ORDER BY c.property_offering_source_match_score DESC, c.property_offering_source_match_price_delta_percent ASC NULLS LAST, c.property_offering_source_match_candidate_id) AS source_next_score
            FROM public.property_offering_source_match_candidates c
            WHERE c.property_offering_source_match_run_id = run_id
        ),
        selected AS (
            SELECT
                r.property_offering_source_match_candidate_id,
                r.source_sale_listing_id,
                r.source_property_offering_id,
                r.target_property_offering_id,
                r.property_offering_source_match_score,
                r.property_offering_source_match_reasons
            FROM ranked r
            JOIN public.property_offering_sources pos ON pos.sale_listing_id = r.source_sale_listing_id
                AND pos.property_offering_id = r.source_property_offering_id
                AND pos.property_offering_source_link_status <> 'rejected'
                AND pos.property_offering_source_link_method <> 'manual'
            WHERE r.property_offering_source_match_score >= score_threshold
                AND r.source_rank = 1
                AND COALESCE(r.source_next_score, -2147483648) <= r.property_offering_source_match_score - competitor_margin
        ),
        linked AS (
            UPDATE public.property_offering_sources pos
            SET
                property_offering_id = s.target_property_offering_id,
                property_offering_source_link_status = 'confirmed',
                property_offering_source_link_method = 'source_match_auto',
                property_offering_source_link_score = s.property_offering_source_match_score,
                property_offering_source_link_reasons = s.property_offering_source_match_reasons || jsonb_build_object('source_match_run_id', run_id),
                property_offering_source_updated_at = now()
            FROM selected s
            WHERE pos.sale_listing_id = s.source_sale_listing_id
                AND pos.property_offering_id = s.source_property_offering_id
                AND pos.property_offering_source_link_status <> 'rejected'
                AND pos.property_offering_source_link_method <> 'manual'
            RETURNING pos.sale_listing_id, pos.property_offering_id
        ),
        listing_state AS (
            UPDATE public.sale_listings sl
            SET
                sale_listing_source_match_status = 'auto_linked',
                sale_listing_source_match_run_id = run_id,
                sale_listing_source_match_last_attempted_at = now(),
                sale_listing_updated_at = now()
            FROM linked l
            WHERE sl.sale_listing_id = l.sale_listing_id
            RETURNING sl.sale_listing_id
        ),
        transaction_links AS (
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
                l.property_offering_id,
                sl.prices_transaction_id,
                COALESCE(sl.sale_listing_prices_match_status, 'confirmed'),
                'sync_auto',
                120,
                jsonb_build_object('source_match_run_id', run_id, 'source_listing_id', sl.sale_listing_id),
                now()
            FROM linked l
            JOIN public.sale_listings sl ON sl.sale_listing_id = l.sale_listing_id
            WHERE sl.prices_transaction_id IS NOT NULL
            ON CONFLICT (property_offering_id, prices_transaction_id) DO UPDATE SET
                property_offering_transaction_link_status = EXCLUDED.property_offering_transaction_link_status,
                property_offering_transaction_link_method = EXCLUDED.property_offering_transaction_link_method,
                property_offering_transaction_link_score = EXCLUDED.property_offering_transaction_link_score,
                property_offering_transaction_link_reasons = EXCLUDED.property_offering_transaction_link_reasons,
                property_offering_transaction_updated_at = now()
            RETURNING 1
        ),
        marked AS (
            UPDATE public.property_offering_source_match_candidates c
            SET property_offering_source_match_status = 'auto_linked'
            FROM linked l
            WHERE c.property_offering_source_match_run_id = run_id
                AND c.source_sale_listing_id = l.sale_listing_id
                AND c.target_property_offering_id = l.property_offering_id
            RETURNING 1
        )
        SELECT count(*)::integer INTO auto_linked_count
        FROM marked;
        UPDATE public.property_offering_source_match_candidates c
        SET property_offering_source_match_status = 'ambiguous'
        WHERE c.property_offering_source_match_run_id = run_id
            AND c.property_offering_source_match_status = 'candidate'
            AND c.property_offering_source_match_score >= score_threshold;
        UPDATE public.sale_listings sl
        SET
            sale_listing_source_match_status = 'needs_review',
            sale_listing_source_match_run_id = run_id,
            sale_listing_source_match_last_attempted_at = now(),
            sale_listing_updated_at = now()
        WHERE EXISTS (
            SELECT 1
            FROM public.property_offering_source_match_candidates c
            WHERE c.property_offering_source_match_run_id = run_id
                AND c.source_sale_listing_id = sl.sale_listing_id
                AND c.property_offering_source_match_status = 'ambiguous'
        );
    END IF;
    SELECT count(*)::integer INTO ambiguous_count
    FROM public.property_offering_source_match_candidates c
    WHERE c.property_offering_source_match_run_id = run_id
        AND c.property_offering_source_match_status = 'ambiguous';
    UPDATE public.property_offering_source_match_runs
    SET
        property_offering_source_match_candidates_count = candidate_count,
        property_offering_source_match_auto_linked_count = auto_linked_count,
        property_offering_source_match_ambiguous_count = ambiguous_count,
        property_offering_source_match_finished_at = now()
    WHERE property_offering_source_match_run_id = run_id;
    RETURN run_id;
END;
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
CREATE OR REPLACE FUNCTION public.fnc__enqueue_sync_job(
    p_provider text,
    p_kind text,
    p_entity_id text,
    p_max_attempts integer DEFAULT 3,
    p_delay_seconds integer DEFAULT 0
) RETURNS uuid AS $$
DECLARE
    v_queue_name text;
    v_job public.sync_jobs%ROWTYPE;
    v_msg_id bigint;
    v_run_after timestamptz;
BEGIN
    v_queue_name := CASE p_provider
        WHEN 'frontdoor' THEN 'frontdoor'
        WHEN 'shortcut' THEN 'shortcut'
        WHEN 'prices' THEN 'prices'
        WHEN 'canonical' THEN 'prices'
        WHEN 'postal' THEN 'postal'
        ELSE NULL
    END;
    IF v_queue_name IS NULL THEN
        RAISE EXCEPTION 'unknown sync provider: %', p_provider;
    END IF;
    v_run_after := now() + make_interval(secs => greatest(coalesce(p_delay_seconds, 0), 0));
    INSERT INTO public.sync_jobs (
        sync_job_provider,
        sync_job_kind,
        sync_job_entity_id,
        sync_job_dedup_key,
        sync_job_status,
        sync_job_attempt_count,
        sync_job_max_attempts,
        sync_job_run_after,
        sync_job_capacity_class,
        sync_job_payload
    ) VALUES (
        p_provider,
        p_kind,
        p_entity_id,
        concat_ws(':', p_provider, p_kind, p_entity_id),
        'pending',
        0,
        greatest(coalesce(p_max_attempts, 3), 1),
        v_run_after,
        CASE p_provider
            WHEN 'frontdoor' THEN 'provider_frontdoor'
            WHEN 'shortcut' THEN CASE WHEN p_kind = 'shortcut_scraper_sync' THEN 'provider_shortcut_scraper' ELSE 'provider_shortcut_api' END
            WHEN 'prices' THEN 'provider_prices'
            WHEN 'canonical' THEN 'internal_db'
            WHEN 'postal' THEN 'provider_postal'
            ELSE 'default'
        END,
        '{}'::jsonb
    )
    ON CONFLICT (sync_job_dedup_key) DO UPDATE
    SET sync_job_provider = EXCLUDED.sync_job_provider,
        sync_job_kind = EXCLUDED.sync_job_kind,
        sync_job_entity_id = EXCLUDED.sync_job_entity_id,
        sync_job_status = 'pending',
        sync_job_attempt_count = 0,
        sync_job_max_attempts = EXCLUDED.sync_job_max_attempts,
        sync_job_run_after = EXCLUDED.sync_job_run_after,
        sync_job_capacity_class = EXCLUDED.sync_job_capacity_class,
        sync_job_payload = EXCLUDED.sync_job_payload,
        sync_job_checkpoint = NULL,
        sync_job_result = NULL,
        sync_job_last_error = NULL,
        sync_job_last_error_code = NULL,
        sync_job_last_http_status = NULL,
        sync_job_claim_token = NULL,
        sync_job_last_finished_at = NULL,
        sync_job_updated_at = now()
    WHERE sync_jobs.sync_job_status IN ('succeeded', 'failed', 'not_found', 'noop', 'skipped_lock')
       OR (sync_jobs.sync_job_status = 'pending' AND sync_jobs.sync_job_run_after > EXCLUDED.sync_job_run_after)
    RETURNING * INTO v_job;
    IF v_job.sync_job_id IS NULL THEN
        SELECT *
        INTO v_job
        FROM public.sync_jobs
        WHERE sync_job_dedup_key = concat_ws(':', p_provider, p_kind, p_entity_id);
        RETURN v_job.sync_job_id;
    END IF;
    v_msg_id := pgmq.send(
        v_queue_name,
        jsonb_build_object(
            'sync_job_id', v_job.sync_job_id,
            'entity_id', v_job.sync_job_entity_id,
            'task_type', v_job.sync_job_kind
        ),
        greatest(coalesce(p_delay_seconds, 0), 0)
    );
    UPDATE public.sync_jobs
    SET sync_job_last_pgmq_message_id = v_msg_id,
        sync_job_last_enqueued_at = now(),
        sync_job_updated_at = now()
    WHERE sync_job_id = v_job.sync_job_id;
    RETURN v_job.sync_job_id;
END;
$$ LANGUAGE plpgsql;
SELECT cron.schedule(
    'trigger-canonical-source-match-fanout',
    '30 4 * * *',
    $$SELECT public.fnc__enqueue_sync_job('canonical', 'canonical_match_sale_listing_sources_fanout', 'canonical:match_sale_listing_sources', 3, 0)$$
) WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-canonical-source-match-fanout'
);
