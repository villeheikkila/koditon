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
    doc.primary_source_listing_id AS sale_listing_id,
    pt.prices_transaction_id,
    doc.room_layout AS listing_layout,
    pt.prices_transaction_description AS transaction_layout,
    doc.area_m2 AS sale_listing_area_value,
    pt.prices_transaction_area,
    doc.property_type_code AS listing_type,
    pt.prices_transaction_type AS transaction_type,
    doc.build_year AS sale_listing_build_year,
    pt.prices_transaction_build_year,
    doc.floor_level AS sale_listing_floor_level,
    doc.total_floors AS sale_listing_total_floors,
    pt.prices_transaction_floor AS transaction_floor,
    doc.elevator AS sale_listing_elevator,
    pt.prices_transaction_elevator,
    doc.condition AS listing_condition,
    pt.prices_transaction_condition AS transaction_condition,
    doc.plot_owned AS sale_listing_plot_owned,
    pt.prices_transaction_plot_owned,
    COALESCE(doc.energy_efficiency_label, doc.energy_class) AS listing_energy,
    pt.prices_transaction_energy_class AS transaction_energy,
    doc.asking_price AS sale_listing_asking_price,
    pt.prices_transaction_price,
    doc.first_seen_at AS sale_listing_first_seen_at,
    doc.last_seen_at AS sale_listing_last_seen_at,
    COALESCE(doc.first_seen_at, doc.refreshed_at) AS sale_listing_created_at,
    doc.refreshed_at AS sale_listing_updated_at,
    pt.prices_transaction_created_at
FROM public.listing_search_documents doc
JOIN origin.prices_transactions pt ON true
JOIN origin.prices_neighborhoods pn ON pn.prices_neighborhood_id = pt.prices_neighborhood_id
LEFT JOIN origin.prices_postal_codes ppc ON ppc.prices_postal_code_id = pn.prices_postal_code_id
LEFT JOIN origin.postal_postal_codes postal ON postal.postal_postal_code_id = pn.prices_neighborhood_postal_postal_code_id
WHERE doc.kind = 'ad'
    AND doc.listing_status = 'active'
    AND doc.primary_source_listing_id IS NOT NULL
    AND (sqlc.narg(target_listing_id)::uuid IS NULL OR doc.primary_source_listing_id = sqlc.narg(target_listing_id)::uuid)
    AND NULLIF(regexp_replace(trim(COALESCE(doc.postal, '')), '[^0-9]+', '', 'g'), '') = NULLIF(regexp_replace(trim(COALESCE(COALESCE(ppc.prices_postal_code_code, postal.postal_postal_code_code), '')), '[^0-9]+', '', 'g'), '')
    AND doc.area_m2 IS NOT NULL
    AND doc.area_m2 = pt.prices_transaction_area
    AND NOT EXISTS (
        SELECT 1
        FROM public.price_links source_link
        WHERE source_link.link_status <> 'rejected'
            AND (
                (source_link.target_type = 'source_listing' AND source_link.target_id = doc.primary_source_listing_id)
                OR (source_link.target_type = 'listing' AND source_link.target_id = doc.property_offering_id)
            )
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

-- name: ApplyTransactionMatchLink :one
WITH source_doc AS (
    SELECT doc.primary_source_listing_id, doc.listing_id, doc.first_seen_at, doc.refreshed_at
    FROM public.listing_search_documents doc
    WHERE doc.primary_source_listing_id = sqlc.arg(listing_id)::uuid
        AND doc.listing_status = 'active'
        AND doc.kind = 'ad'
    ORDER BY doc.last_seen_at DESC NULLS LAST, doc.refreshed_at DESC
    LIMIT 1
),
inserted AS (
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
        source_doc.primary_source_listing_id,
        sqlc.arg(transaction_id)::uuid,
        'confirmed',
        'sync_auto',
        sqlc.arg(score)::int4,
        sqlc.arg(reasons)::jsonb,
        COALESCE(source_doc.first_seen_at, now()),
        now()
    FROM source_doc
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
        updated_at = now()
    RETURNING target_id
),
updated_state AS (
    INSERT INTO public.listing_price_match_states (
        source_listing_id,
        listing_id,
        match_status,
        last_attempted_at,
        run_id,
        updated_at
    )
    SELECT
        source_doc.primary_source_listing_id,
        source_doc.listing_id,
        'auto_linked',
        now(),
        sqlc.arg(run_id)::uuid,
        now()
    FROM source_doc
    WHERE EXISTS (SELECT 1 FROM inserted)
    ON CONFLICT (source_listing_id) DO UPDATE SET
        listing_id = EXCLUDED.listing_id,
        match_status = EXCLUDED.match_status,
        last_attempted_at = EXCLUDED.last_attempted_at,
        run_id = EXCLUDED.run_id,
        updated_at = now()
    RETURNING 1
)
SELECT count(*)::int4 AS rows_affected FROM inserted;

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
