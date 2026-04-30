ALTER TABLE public.sale_listings
ADD COLUMN IF NOT EXISTS sale_listing_first_seen_at timestamp with time zone;
CREATE INDEX IF NOT EXISTS idx_sale_listings_first_seen ON public.sale_listings (sale_listing_first_seen_at);
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_period_month(value text)
RETURNS date
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN value ~ '^[0-9]{4}-[0-9]{2}$' THEN to_date(value || '-01', 'YYYY-MM-DD')
        ELSE NULL
    END
$$;
CREATE OR REPLACE FUNCTION public.fnc__layout_match_key(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF(regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]åäö]+', '', 'g'), '')
$$;
CREATE OR REPLACE FUNCTION public.fnc__layout_prefix_match(listing_layout text, transaction_description text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    WITH values AS (
        SELECT
            public.fnc__layout_match_key(listing_layout) AS listing_key,
            public.fnc__layout_match_key(transaction_description) AS transaction_key
    )
    SELECT listing_key IS NOT NULL
        AND transaction_key IS NOT NULL
        AND length(transaction_key) >= 2
        AND left(listing_key, length(transaction_key)) = transaction_key
    FROM values
$$;
CREATE OR REPLACE FUNCTION public.fnc__area_match_key(value double precision)
RETURNS numeric
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT round(value::numeric, 1)
$$;
CREATE OR REPLACE FUNCTION public.fnc__sale_listings_set_transaction_match_fields()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
    property_raw text;
    plot_raw text;
    elevator_value boolean;
    first_seen_at timestamp with time zone;
    energy_label text;
    energy_match_label text;
    energy_normalized record;
BEGIN
    IF TG_OP = 'UPDATE'
        AND NEW.shortcut_ad_id IS NOT DISTINCT FROM OLD.shortcut_ad_id
        AND NEW.frontdoor_ad_id IS NOT DISTINCT FROM OLD.frontdoor_ad_id
        AND NEW.frontdoor_building_announcement_id IS NOT DISTINCT FROM OLD.frontdoor_building_announcement_id
        AND NEW.sale_listing_rooms_count IS NOT DISTINCT FROM OLD.sale_listing_rooms_count
        AND NEW.sale_listing_room_layout IS NOT DISTINCT FROM OLD.sale_listing_room_layout
        AND NEW.sale_listing_floor_level IS NOT DISTINCT FROM OLD.sale_listing_floor_level
        AND NEW.sale_listing_total_floors IS NOT DISTINCT FROM OLD.sale_listing_total_floors
        AND NEW.sale_listing_energy_class IS NOT DISTINCT FROM OLD.sale_listing_energy_class
    THEN
        RETURN NEW;
    END IF;
    IF NEW.shortcut_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,habitationType}', sa.shortcut_ad_data #>> '{adData,buildingType}', sa.shortcut_ad_data #>> '{buildingData,buildingType}')), ''),
            NULLIF(trim(COALESCE(sa.shortcut_ad_data #>> '{adData,plotType}', sa.shortcut_ad_data #>> '{property,plotType}', sb.shortcut_building_plot_type)), ''),
            COALESCE(sa.shortcut_ad_elevator, public.fnc__try_parse_bool(sb.shortcut_building_has_elevator)),
            sa.shortcut_ad_first_seen_at,
            public.fnc__energy_efficiency_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class),
            public.fnc__energy_efficiency_match_label(sa.shortcut_ad_data #>> '{adData,buildingOverrideEnergyClass}', sa.shortcut_ad_data #>> '{adData,energyClass}', sa.shortcut_ad_data #>> '{property,energyClass}', sa.shortcut_ad_energy_class)
        INTO property_raw, plot_raw, elevator_value, first_seen_at, energy_label, energy_match_label
        FROM public.shortcut_ads sa
        LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
        WHERE sa.shortcut_ad_id = NEW.shortcut_ad_id;
    ELSIF NEW.frontdoor_ad_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,residentialPropertyType}', fa.frontdoor_ad_data #>> '{property,specificType}', fa.frontdoor_ad_data #>> '{property,propertyType}')), ''),
            NULLIF(trim(COALESCE(fa.frontdoor_ad_data #>> '{property,plot,holdingType}', fa.frontdoor_ad_data #>> '{property,plotOwnershipType}', fa.frontdoor_ad_plot_type)), ''),
            fa.frontdoor_ad_elevator,
            fa.frontdoor_ad_first_seen_at,
            public.fnc__energy_efficiency_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class),
            public.fnc__energy_efficiency_match_label(fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateDescription}', fa.frontdoor_ad_data #>> '{property,housingCompany,energyCertificate,energyCertificateType}', fa.frontdoor_ad_data #>> '{property,energyCertificate,energyCertificateType}', fa.frontdoor_ad_energy_class)
        INTO property_raw, plot_raw, elevator_value, first_seen_at, energy_label, energy_match_label
        FROM public.frontdoor_ads fa
        WHERE fa.frontdoor_ad_id = NEW.frontdoor_ad_id;
    ELSIF NEW.frontdoor_building_announcement_id IS NOT NULL THEN
        SELECT
            NULLIF(trim(COALESCE(fba.frontdoor_building_announcement_property_subtype, fba.frontdoor_building_announcement_property_type)), ''),
            NULL::text,
            fb.frontdoor_building_has_elevator,
            fba.frontdoor_building_announcement_last_seen_at,
            public.fnc__energy_efficiency_label(fb.frontdoor_building_energy_certificate_code),
            public.fnc__energy_efficiency_match_label(fb.frontdoor_building_energy_certificate_code)
        INTO property_raw, plot_raw, elevator_value, first_seen_at, energy_label, energy_match_label
        FROM public.frontdoor_building_announcements fba
        JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
        WHERE fba.frontdoor_building_announcement_id = NEW.frontdoor_building_announcement_id;
    END IF;
    SELECT * INTO energy_normalized FROM public.fnc__energy_efficiency_normalized(energy_match_label);
    NEW.sale_listing_first_seen_at := COALESCE(first_seen_at, NEW.sale_listing_first_seen_at, NEW.sale_listing_created_at);
    NEW.sale_listing_property_type_raw := property_raw;
    NEW.sale_listing_property_type_code := public.fnc__sale_listing_property_type_code(property_raw);
    NEW.sale_listing_room_category_code := public.fnc__sale_listing_room_category_code(NEW.sale_listing_rooms_count, NEW.sale_listing_room_layout);
    NEW.sale_listing_floor_text := public.fnc__sale_listing_floor_text(NEW.sale_listing_floor_level, NEW.sale_listing_total_floors);
    NEW.sale_listing_elevator := elevator_value;
    NEW.sale_listing_plot_type_raw := plot_raw;
    NEW.sale_listing_plot_type_code := public.fnc__sale_listing_plot_type_code(plot_raw);
    NEW.sale_listing_energy_efficiency_label := energy_label;
    NEW.sale_listing_energy_efficiency_class_code := energy_normalized.energy_efficiency_class_code;
    NEW.sale_listing_energy_efficiency_standard_year := energy_normalized.energy_efficiency_standard_year;
    NEW.sale_listing_energy_efficiency_status := energy_normalized.energy_efficiency_status;
    NEW.sale_listing_energy_efficiency_match_code := energy_normalized.energy_efficiency_match_code;
    RETURN NEW;
END;
$$;
UPDATE public.sale_listings sl
SET sale_listing_first_seen_at = COALESCE(sa.shortcut_ad_first_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_created_at)
FROM public.shortcut_ads sa
WHERE sl.shortcut_ad_id = sa.shortcut_ad_id;
UPDATE public.sale_listings sl
SET sale_listing_first_seen_at = COALESCE(fa.frontdoor_ad_first_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_created_at)
FROM public.frontdoor_ads fa
WHERE sl.frontdoor_ad_id = fa.frontdoor_ad_id;
UPDATE public.sale_listings sl
SET sale_listing_first_seen_at = COALESCE(fba.frontdoor_building_announcement_last_seen_at, sl.sale_listing_first_seen_at, sl.sale_listing_created_at)
FROM public.frontdoor_building_announcements fba
WHERE sl.frontdoor_building_announcement_id = fba.frontdoor_building_announcement_id;
UPDATE public.sale_listings
SET sale_listing_first_seen_at = sale_listing_created_at
WHERE sale_listing_first_seen_at IS NULL;
CREATE TABLE IF NOT EXISTS public.sale_listing_prices_transaction_match_runs (
    sale_listing_prices_transaction_match_run_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    sale_listing_prices_transaction_match_run_mode text NOT NULL,
    sale_listing_prices_transaction_match_score_threshold integer DEFAULT 90 NOT NULL,
    sale_listing_prices_transaction_match_competitor_margin integer DEFAULT 15 NOT NULL,
    sale_listing_prices_transaction_match_candidates_count integer DEFAULT 0 NOT NULL,
    sale_listing_prices_transaction_match_auto_linked_count integer DEFAULT 0 NOT NULL,
    sale_listing_prices_transaction_match_ambiguous_count integer DEFAULT 0 NOT NULL,
    sale_listing_prices_transaction_match_started_at timestamp with time zone DEFAULT now() NOT NULL,
    sale_listing_prices_transaction_match_finished_at timestamp with time zone,
    CONSTRAINT sale_listing_prices_transaction_match_run_mode_check CHECK (sale_listing_prices_transaction_match_run_mode = ANY (ARRAY['dry_run'::text, 'auto_link_safe'::text])),
    CONSTRAINT sale_listing_prices_transaction_match_threshold_check CHECK (sale_listing_prices_transaction_match_score_threshold >= 0),
    CONSTRAINT sale_listing_prices_transaction_match_margin_check CHECK (sale_listing_prices_transaction_match_competitor_margin >= 0)
);
CREATE TABLE IF NOT EXISTS public.sale_listing_prices_transaction_match_candidates (
    sale_listing_prices_transaction_match_candidate_id uuid DEFAULT gen_random_uuid() NOT NULL PRIMARY KEY,
    sale_listing_prices_transaction_match_run_id uuid NOT NULL REFERENCES public.sale_listing_prices_transaction_match_runs(sale_listing_prices_transaction_match_run_id) ON DELETE CASCADE,
    sale_listing_id uuid NOT NULL REFERENCES public.sale_listings(sale_listing_id) ON DELETE CASCADE,
    prices_transaction_id uuid NOT NULL REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE CASCADE,
    sale_listing_prices_transaction_match_score integer NOT NULL,
    sale_listing_prices_transaction_match_confidence text NOT NULL,
    sale_listing_prices_transaction_match_status text DEFAULT 'candidate'::text NOT NULL,
    sale_listing_prices_transaction_match_reasons jsonb DEFAULT '{}'::jsonb NOT NULL,
    sale_listing_prices_transaction_match_price_delta_percent double precision,
    sale_listing_prices_transaction_match_created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT sale_listing_prices_transaction_match_candidate_unique UNIQUE (sale_listing_prices_transaction_match_run_id, sale_listing_id, prices_transaction_id),
    CONSTRAINT sale_listing_prices_transaction_match_confidence_check CHECK (sale_listing_prices_transaction_match_confidence = ANY (ARRAY['high'::text, 'medium'::text, 'low'::text])),
    CONSTRAINT sale_listing_prices_transaction_match_status_check CHECK (sale_listing_prices_transaction_match_status = ANY (ARRAY['candidate'::text, 'auto_linked'::text, 'ambiguous'::text, 'rejected'::text]))
);
CREATE INDEX IF NOT EXISTS idx_sale_listing_prices_transaction_match_candidates_run_status ON public.sale_listing_prices_transaction_match_candidates (sale_listing_prices_transaction_match_run_id, sale_listing_prices_transaction_match_status);
CREATE INDEX IF NOT EXISTS idx_sale_listing_prices_transaction_match_candidates_listing_score ON public.sale_listing_prices_transaction_match_candidates (sale_listing_id, sale_listing_prices_transaction_match_score DESC);
CREATE INDEX IF NOT EXISTS idx_sale_listing_prices_transaction_match_candidates_transaction_score ON public.sale_listing_prices_transaction_match_candidates (prices_transaction_id, sale_listing_prices_transaction_match_score DESC);
CREATE OR REPLACE FUNCTION public.fnc__refresh_sale_listing_prices_transaction_matches(auto_link_safe boolean DEFAULT false, score_threshold integer DEFAULT 90, competitor_margin integer DEFAULT 15)
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
            sl.sale_listing_public_id,
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
