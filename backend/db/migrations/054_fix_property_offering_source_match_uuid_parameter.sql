-- Replace source-offering matching function after canonical building table was renamed.
DROP FUNCTION IF EXISTS public.fnc__refresh_property_offering_source_matches(boolean, integer, integer, uuid);
DROP FUNCTION IF EXISTS public.fnc__refresh_property_offering_source_matches(boolean, integer, integer, text);
DROP FUNCTION IF EXISTS public.fnc__refresh_property_offering_source_matches(boolean, integer, integer);
CREATE OR REPLACE FUNCTION public.fnc__refresh_property_offering_source_matches(auto_link_safe boolean DEFAULT false, score_threshold integer DEFAULT 95, competitor_margin integer DEFAULT 10, p_sale_listing_id uuid DEFAULT NULL)
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
            spb.housing_company_identity_key AS source_building_identity_key,
            spu.property_unit_identity_key AS source_unit_identity_key
        FROM public.property_offering_sources pos
        JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
        JOIN public.property_offerings spo ON spo.property_offering_id = pos.property_offering_id
        JOIN public.property_units spu ON spu.property_unit_id = spo.property_unit_id
        JOIN public.housing_companies spb ON spb.housing_company_id = spu.housing_company_id
        WHERE pos.property_offering_source_link_status <> 'rejected'
            AND pos.property_offering_source_link_method <> 'manual'
            AND sl.sale_listing_source_kind = 'ad'
            AND (p_sale_listing_id IS NULL OR sl.sale_listing_id = p_sale_listing_id)
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
            tpb.housing_company_identity_key AS target_building_identity_key,
            tpu.property_unit_identity_key AS target_unit_identity_key
        FROM public.property_offering_sources pos
        JOIN public.sale_listings sl ON sl.sale_listing_id = pos.sale_listing_id
        JOIN public.property_offerings tpo ON tpo.property_offering_id = pos.property_offering_id
        JOIN public.property_units tpu ON tpu.property_unit_id = tpo.property_unit_id
        JOIN public.housing_companies tpb ON tpb.housing_company_id = tpu.housing_company_id
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
