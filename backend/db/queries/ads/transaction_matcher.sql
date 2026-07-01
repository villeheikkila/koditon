-- name: CreateTransactionMatchRun :one
INSERT INTO public.sale_listing_prices_transaction_match_runs (
    sale_listing_prices_transaction_match_run_mode,
    sale_listing_prices_transaction_match_score_threshold,
    sale_listing_prices_transaction_match_competitor_margin
)
VALUES (sqlc.arg(mode)::text, sqlc.arg(score_threshold)::int4, sqlc.arg(competitor_margin)::int4)
RETURNING sale_listing_prices_transaction_match_run_id;

-- name: LoadTransactionMatchCandidateRows :many
SELECT
    sl.sale_listing_id,
    pt.prices_transaction_id,
    COALESCE(sl.sale_listing_room_layout, '')::text AS listing_layout,
    COALESCE(pt.prices_transaction_description, '')::text AS transaction_layout,
    sl.sale_listing_area_value,
    pt.prices_transaction_area,
    COALESCE(sl.sale_listing_property_type_code, '')::text AS listing_type,
    COALESCE(pt.prices_transaction_type, '')::text AS transaction_type,
    sl.sale_listing_build_year,
    pt.prices_transaction_build_year,
    sl.sale_listing_floor_level,
    sl.sale_listing_total_floors,
    COALESCE(pt.prices_transaction_floor, '')::text AS transaction_floor,
    sl.sale_listing_elevator,
    pt.prices_transaction_elevator,
    COALESCE(sl.sale_listing_condition, '')::text AS listing_condition,
    COALESCE(pt.prices_transaction_condition, '')::text AS transaction_condition,
    sl.sale_listing_plot_owned,
    pt.prices_transaction_plot_owned,
    COALESCE(sl.sale_listing_energy_efficiency_match_code, '')::text AS listing_energy,
    COALESCE(pt.prices_transaction_energy_class, '')::text AS transaction_energy,
    sl.sale_listing_asking_price,
    pt.prices_transaction_price,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_created_at,
    sl.sale_listing_updated_at,
    pt.prices_transaction_created_at
FROM public.property_source_offerings sl
JOIN public.prices_transactions pt ON true
JOIN public.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN public.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN public.postal_postal_codes postal ON postal.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
WHERE sl.sale_listing_source_kind = 'ad'
    AND (sqlc.narg(target_listing_id)::uuid IS NULL OR sl.sale_listing_id = sqlc.narg(target_listing_id)::uuid)
    AND sl.sale_listing_postal_norm = public.fnc__normalize_postal(COALESCE(ppc.prices_postal_code_code, postal.postal_postal_code_code))
    AND sl.sale_listing_area_value IS NOT NULL
    AND sl.sale_listing_area_value = pt.prices_transaction_area
    AND NOT EXISTS (
        SELECT 1
        FROM public.price_links source_link
        WHERE source_link.target_type = 'source_listing'
            AND source_link.target_id = sl.sale_listing_id
            AND source_link.link_status <> 'rejected'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM public.price_links linked
        WHERE linked.prices_transaction_id = pt.prices_transaction_id
            AND linked.link_status <> 'rejected'
    );

-- name: InsertTransactionMatchCandidate :exec
INSERT INTO public.sale_listing_prices_transaction_match_candidates (
    sale_listing_prices_transaction_match_run_id,
    sale_listing_id,
    prices_transaction_id,
    sale_listing_prices_transaction_match_score,
    sale_listing_prices_transaction_match_confidence,
    sale_listing_prices_transaction_match_reasons,
    sale_listing_prices_transaction_match_price_delta_percent
)
VALUES (
    sqlc.arg(run_id)::uuid,
    sqlc.arg(listing_id)::uuid,
    sqlc.arg(transaction_id)::uuid,
    sqlc.arg(score)::int4,
    sqlc.arg(confidence)::text,
    sqlc.arg(reasons)::jsonb,
    sqlc.narg(price_delta_percent)::double precision
);

-- name: ApplyTransactionMatchLink :execrows
WITH updated_source AS (
    UPDATE public.property_source_offerings
    SET prices_transaction_id = sqlc.arg(transaction_id)::uuid,
        sale_listing_prices_match_status = 'auto_linked',
        sale_listing_prices_match_run_id = sqlc.arg(run_id)::uuid,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = sqlc.arg(listing_id)::uuid
        AND (prices_transaction_id IS NULL OR prices_transaction_id = sqlc.arg(transaction_id)::uuid)
        AND NOT EXISTS (
            SELECT 1
            FROM public.price_links existing
            WHERE existing.prices_transaction_id = sqlc.arg(transaction_id)::uuid
                AND existing.link_status <> 'rejected'
        )
    RETURNING sale_listing_id, sale_listing_created_at, sale_listing_updated_at
)
INSERT INTO public.price_links (
    target_type,
    target_id,
    prices_transaction_id,
    link_status,
    link_method,
    link_score,
    link_reasons,
    created_at,
    updated_at
)
SELECT
    'source_listing',
    sale_listing_id,
    sqlc.arg(transaction_id)::uuid,
    'confirmed',
    'sync_auto',
    sqlc.arg(score)::int4,
    sqlc.arg(reasons)::jsonb,
    sale_listing_created_at,
    sale_listing_updated_at
FROM updated_source
WHERE NOT EXISTS (
        SELECT 1
        FROM public.price_links existing
        WHERE existing.prices_transaction_id = sqlc.arg(transaction_id)::uuid
            AND existing.link_status <> 'rejected'
    )
ON CONFLICT (target_type, target_id, prices_transaction_id) DO UPDATE SET
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = EXCLUDED.link_reasons,
    updated_at = now();

-- name: SyncSourceListingTransactionMatchState :exec
UPDATE public.source_listings src
SET normalized_at = sl.sale_listing_updated_at,
    updated_at = sl.sale_listing_updated_at
FROM public.property_source_offerings sl
WHERE sl.sale_listing_id = sqlc.arg(listing_id)::uuid
    AND src.source_listing_id = sl.sale_listing_id;

-- name: MarkTransactionMatchLinked :exec
UPDATE public.sale_listing_prices_transaction_match_candidates
SET sale_listing_prices_transaction_match_status = 'auto_linked'
WHERE sale_listing_prices_transaction_match_run_id = sqlc.arg(run_id)::uuid
    AND sale_listing_id = sqlc.arg(listing_id)::uuid
    AND prices_transaction_id = sqlc.arg(transaction_id)::uuid;

-- name: MarkAmbiguousTransactionMatches :execrows
UPDATE public.sale_listing_prices_transaction_match_candidates
SET sale_listing_prices_transaction_match_status = 'ambiguous'
WHERE sale_listing_prices_transaction_match_run_id = sqlc.arg(run_id)::uuid
    AND sale_listing_prices_transaction_match_status = 'candidate'
    AND sale_listing_prices_transaction_match_score >= sqlc.arg(threshold)::int4;

-- name: FinishTransactionMatchRun :exec
UPDATE public.sale_listing_prices_transaction_match_runs
SET sale_listing_prices_transaction_match_candidates_count = sqlc.arg(candidates)::int4,
    sale_listing_prices_transaction_match_auto_linked_count = sqlc.arg(auto_linked)::int4,
    sale_listing_prices_transaction_match_ambiguous_count = sqlc.arg(ambiguous)::int4,
    sale_listing_prices_transaction_match_finished_at = now()
WHERE sale_listing_prices_transaction_match_run_id = sqlc.arg(run_id)::uuid;
