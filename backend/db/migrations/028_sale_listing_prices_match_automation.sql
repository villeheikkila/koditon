ALTER TABLE public.sale_listings
ADD COLUMN IF NOT EXISTS sale_listing_prices_match_status text,
ADD COLUMN IF NOT EXISTS sale_listing_prices_match_next_attempt_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS sale_listing_prices_match_last_attempted_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS sale_listing_prices_match_attempt_count integer DEFAULT 0 NOT NULL,
ADD COLUMN IF NOT EXISTS sale_listing_prices_match_expires_at timestamp with time zone,
ADD COLUMN IF NOT EXISTS sale_listing_prices_match_run_id uuid REFERENCES public.sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE SET NULL,
ADD CONSTRAINT sale_listings_prices_match_status_check CHECK (
    sale_listing_prices_match_status IS NULL
    OR sale_listing_prices_match_status = ANY (ARRAY['pending'::text, 'deferred'::text, 'auto_linked'::text, 'needs_review'::text, 'manual_linked'::text, 'rejected'::text, 'expired'::text, 'noop'::text])
);
CREATE INDEX IF NOT EXISTS idx_sale_listings_prices_match_queue ON public.sale_listings (sale_listing_prices_match_status, sale_listing_prices_match_next_attempt_at)
WHERE prices_transaction_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_sale_listings_prices_match_last_seen ON public.sale_listings (sale_listing_last_seen_at)
WHERE prices_transaction_id IS NULL AND sale_listing_source_kind = 'ad';
DROP FUNCTION IF EXISTS public.fnc__refresh_sale_listing_prices_transaction_matches(boolean, integer, integer);
CREATE OR REPLACE FUNCTION public.fnc__refresh_sale_listing_prices_transaction_matches(auto_link_safe boolean DEFAULT false, score_threshold integer DEFAULT 90, competitor_margin integer DEFAULT 15, listing_public_id text DEFAULT NULL)
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
    INSERT INTO public.sale_listing_prices_transaction_match_runs (
        sale_listing_prices_transaction_match_run_mode,
        sale_listing_prices_transaction_match_score_threshold,
        sale_listing_prices_transaction_match_competitor_margin
    )
    VALUES (
        CASE WHEN auto_link_safe THEN 'auto_link_safe' ELSE 'dry_run' END,
        score_threshold,
        competitor_margin
    )
    RETURNING sale_listing_prices_transaction_match_run_id INTO run_id;
    WITH candidate_base AS (
        SELECT
            sl.sale_listing_id,
            pt.prices_transaction_id,
            CASE WHEN sl.sale_listing_floor_level IS NOT NULL AND sl.sale_listing_floor_level = public.fnc__prices_transaction_floor_level(pt.prices_transaction_floor) THEN 10 ELSE 0 END AS floor_score,
            CASE WHEN sl.sale_listing_total_floors IS NOT NULL AND sl.sale_listing_total_floors = public.fnc__prices_transaction_total_floors(pt.prices_transaction_floor) THEN 5 ELSE 0 END AS total_floor_score,
            CASE WHEN sl.sale_listing_energy_efficiency_match_code IS NOT NULL AND sl.sale_listing_energy_efficiency_match_code = public.fnc__prices_transaction_energy_match_code(pt.prices_transaction_energy_class) THEN 8 ELSE 0 END AS energy_score,
            CASE WHEN sl.sale_listing_build_year IS NOT NULL AND pt.prices_transaction_build_year > 0 AND sl.sale_listing_build_year = pt.prices_transaction_build_year THEN 8 ELSE 0 END AS build_year_score,
            CASE WHEN sl.sale_listing_room_category_code IS NOT NULL AND sl.sale_listing_room_category_code = public.fnc__prices_transaction_room_category_code(pt.prices_transaction_category) THEN 5 ELSE 0 END AS room_category_score,
            CASE WHEN sl.sale_listing_elevator IS NOT NULL AND sl.sale_listing_elevator = pt.prices_transaction_elevator THEN 3 ELSE 0 END AS elevator_score,
            CASE WHEN sl.sale_listing_plot_type_code IS NOT NULL AND sl.sale_listing_plot_type_code = public.fnc__prices_transaction_plot_type_code(pt.prices_transaction_plot) THEN 3 ELSE 0 END AS plot_score,
            CASE WHEN pt.prices_transaction_created_at >= COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_updated_at) - interval '14 days' AND pt.prices_transaction_created_at <= COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_updated_at) + interval '9 months' THEN 10 ELSE 0 END AS temporal_score,
            CASE WHEN sl.sale_listing_asking_price IS NOT NULL AND pt.prices_transaction_price > 0 THEN abs(sl.sale_listing_asking_price::double precision - pt.prices_transaction_price::double precision) / pt.prices_transaction_price::double precision ELSE NULL END AS price_delta_percent,
            jsonb_build_object(
                'postal', sl.sale_listing_postal_norm,
                'area', public.fnc__area_match_key(sl.sale_listing_area_value),
                'layout_prefix', true,
                'property_type', sl.sale_listing_property_type_code,
                'floor_level', jsonb_build_object('listing', sl.sale_listing_floor_level, 'transaction', public.fnc__prices_transaction_floor_level(pt.prices_transaction_floor)),
                'total_floors', jsonb_build_object('listing', sl.sale_listing_total_floors, 'transaction', public.fnc__prices_transaction_total_floors(pt.prices_transaction_floor)),
                'energy', jsonb_build_object('listing', sl.sale_listing_energy_efficiency_match_code, 'transaction', public.fnc__prices_transaction_energy_match_code(pt.prices_transaction_energy_class)),
                'build_year', jsonb_build_object('listing', sl.sale_listing_build_year, 'transaction', pt.prices_transaction_build_year),
                'room_category', jsonb_build_object('listing', sl.sale_listing_room_category_code, 'transaction', public.fnc__prices_transaction_room_category_code(pt.prices_transaction_category)),
                'elevator', jsonb_build_object('listing', sl.sale_listing_elevator, 'transaction', pt.prices_transaction_elevator),
                'plot', jsonb_build_object('listing', sl.sale_listing_plot_type_code, 'transaction', public.fnc__prices_transaction_plot_type_code(pt.prices_transaction_plot)),
                'transaction_created_at', pt.prices_transaction_created_at,
                'transaction_period_month', public.fnc__prices_transaction_period_month(pt.prices_transaction_period_identifier),
                'listing_first_seen_at', sl.sale_listing_first_seen_at,
                'listing_last_seen_at', sl.sale_listing_last_seen_at
            ) AS reasons
        FROM public.sale_listings sl
        JOIN public.prices_transactions pt ON true
        JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
        LEFT JOIN public.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
        LEFT JOIN public.postal_postal_codes postal ON postal.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
        WHERE sl.prices_transaction_id IS NULL
            AND NOT EXISTS (
                SELECT 1
                FROM public.sale_listings linked
                WHERE linked.prices_transaction_id = pt.prices_transaction_id
            )
            AND sl.sale_listing_source_kind = 'ad'
            AND (listing_public_id IS NULL OR sl.sale_listing_public_id = listing_public_id)
            AND sl.sale_listing_postal_norm = public.fnc__normalize_postal(COALESCE(ppc.prices_postal_code_code, postal.postal_postal_code_code))
            AND sl.sale_listing_area_value IS NOT NULL
            AND public.fnc__area_match_key(sl.sale_listing_area_value) = public.fnc__area_match_key(pt.prices_transaction_area)
            AND public.fnc__layout_prefix_match(sl.sale_listing_room_layout, pt.prices_transaction_description)
            AND sl.sale_listing_property_type_code IS NOT NULL
            AND sl.sale_listing_property_type_code = public.fnc__prices_transaction_property_type_code(pt.prices_transaction_type)
            AND pt.prices_transaction_created_at >= COALESCE(sl.sale_listing_first_seen_at, sl.sale_listing_created_at) - interval '45 days'
            AND pt.prices_transaction_created_at <= COALESCE(sl.sale_listing_last_seen_at, sl.sale_listing_updated_at, now()) + interval '18 months'
    ),
    scored AS (
        SELECT
            sale_listing_id,
            prices_transaction_id,
            (90 + floor_score + total_floor_score + energy_score + build_year_score + room_category_score + elevator_score + plot_score + temporal_score)::integer AS score,
            price_delta_percent,
            reasons || jsonb_build_object(
                'score', jsonb_build_object(
                    'postal', 25,
                    'area', 25,
                    'layout', 25,
                    'property_type', 15,
                    'floor', floor_score,
                    'total_floors', total_floor_score,
                    'energy', energy_score,
                    'build_year', build_year_score,
                    'room_category', room_category_score,
                    'elevator', elevator_score,
                    'plot', plot_score,
                    'temporal', temporal_score
                ),
                'price_delta_percent', price_delta_percent
            ) AS reasons
        FROM candidate_base
    )
    INSERT INTO public.sale_listing_prices_transaction_match_candidates (
        sale_listing_prices_transaction_match_run_id,
        sale_listing_id,
        prices_transaction_id,
        sale_listing_prices_transaction_match_score,
        sale_listing_prices_transaction_match_confidence,
        sale_listing_prices_transaction_match_reasons,
        sale_listing_prices_transaction_match_price_delta_percent
    )
    SELECT
        run_id,
        sale_listing_id,
        prices_transaction_id,
        score,
        CASE WHEN score >= score_threshold THEN 'high' WHEN score >= 75 THEN 'medium' ELSE 'low' END,
        reasons,
        price_delta_percent
    FROM scored;
    GET DIAGNOSTICS candidate_count = ROW_COUNT;
    IF auto_link_safe THEN
        WITH ranked AS (
            SELECT
                c.sale_listing_prices_transaction_match_candidate_id,
                c.sale_listing_id,
                c.prices_transaction_id,
                c.sale_listing_prices_transaction_match_score,
                c.sale_listing_prices_transaction_match_price_delta_percent,
                row_number() OVER (PARTITION BY c.sale_listing_id ORDER BY c.sale_listing_prices_transaction_match_score DESC, c.sale_listing_prices_transaction_match_price_delta_percent ASC NULLS LAST, c.sale_listing_prices_transaction_match_candidate_id) AS listing_rank,
                lead(c.sale_listing_prices_transaction_match_score) OVER (PARTITION BY c.sale_listing_id ORDER BY c.sale_listing_prices_transaction_match_score DESC, c.sale_listing_prices_transaction_match_price_delta_percent ASC NULLS LAST, c.sale_listing_prices_transaction_match_candidate_id) AS listing_next_score,
                row_number() OVER (PARTITION BY c.prices_transaction_id ORDER BY c.sale_listing_prices_transaction_match_score DESC, c.sale_listing_prices_transaction_match_price_delta_percent ASC NULLS LAST, c.sale_listing_prices_transaction_match_candidate_id) AS transaction_rank,
                lead(c.sale_listing_prices_transaction_match_score) OVER (PARTITION BY c.prices_transaction_id ORDER BY c.sale_listing_prices_transaction_match_score DESC, c.sale_listing_prices_transaction_match_price_delta_percent ASC NULLS LAST, c.sale_listing_prices_transaction_match_candidate_id) AS transaction_next_score
            FROM public.sale_listing_prices_transaction_match_candidates c
            WHERE c.sale_listing_prices_transaction_match_run_id = run_id
        ),
        selected AS (
            SELECT
                r.sale_listing_prices_transaction_match_candidate_id,
                r.sale_listing_id,
                r.prices_transaction_id
            FROM ranked r
            WHERE r.sale_listing_prices_transaction_match_score >= score_threshold
                AND r.listing_rank = 1
                AND r.transaction_rank = 1
                AND COALESCE(r.listing_next_score, -2147483648) <= r.sale_listing_prices_transaction_match_score - competitor_margin
                AND COALESCE(r.transaction_next_score, -2147483648) <= r.sale_listing_prices_transaction_match_score - competitor_margin
        ),
        linked AS (
            UPDATE public.sale_listings sl
            SET
                prices_transaction_id = s.prices_transaction_id,
                sale_listing_prices_match_status = 'auto_linked',
                sale_listing_prices_match_run_id = run_id,
                sale_listing_updated_at = now()
            FROM selected s
            WHERE sl.sale_listing_id = s.sale_listing_id
                AND sl.prices_transaction_id IS NULL
                AND NOT EXISTS (
                    SELECT 1
                    FROM public.sale_listings existing
                    WHERE existing.prices_transaction_id = s.prices_transaction_id
                        AND existing.sale_listing_id <> sl.sale_listing_id
                )
            RETURNING sl.sale_listing_id, sl.prices_transaction_id
        ),
        marked AS (
            UPDATE public.sale_listing_prices_transaction_match_candidates c
            SET sale_listing_prices_transaction_match_status = 'auto_linked'
            FROM linked l
            WHERE c.sale_listing_prices_transaction_match_run_id = run_id
                AND c.sale_listing_id = l.sale_listing_id
                AND c.prices_transaction_id = l.prices_transaction_id
            RETURNING 1
        )
        SELECT count(*)::integer INTO auto_linked_count
        FROM marked;
        UPDATE public.sale_listing_prices_transaction_match_candidates c
        SET sale_listing_prices_transaction_match_status = 'ambiguous'
        WHERE c.sale_listing_prices_transaction_match_run_id = run_id
            AND c.sale_listing_prices_transaction_match_status = 'candidate'
            AND c.sale_listing_prices_transaction_match_score >= score_threshold;
    END IF;
    SELECT count(*)::integer INTO ambiguous_count
    FROM public.sale_listing_prices_transaction_match_candidates c
    WHERE c.sale_listing_prices_transaction_match_run_id = run_id
        AND c.sale_listing_prices_transaction_match_status = 'ambiguous';
    UPDATE public.sale_listing_prices_transaction_match_runs
    SET
        sale_listing_prices_transaction_match_candidates_count = candidate_count,
        sale_listing_prices_transaction_match_auto_linked_count = auto_linked_count,
        sale_listing_prices_transaction_match_ambiguous_count = ambiguous_count,
        sale_listing_prices_transaction_match_finished_at = now()
    WHERE sale_listing_prices_transaction_match_run_id = run_id;
    RETURN run_id;
END;
$$;
SELECT cron.schedule(
    'trigger-prices-sale-listing-match-fanout',
    '0 4 * * *',
    $$SELECT public.fnc__enqueue_sync_job('prices', 'prices_match_sale_listings_fanout', 'prices:match_sale_listings', 3, 0)$$
) WHERE NOT EXISTS (
    SELECT 1 FROM cron.job WHERE jobname = 'trigger-prices-sale-listing-match-fanout'
);
