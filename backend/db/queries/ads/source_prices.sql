-- name: ObserveSourceListingPrice :one
WITH locked_source AS MATERIALIZED (
    SELECT source_listing_id
    FROM origin.source_listings
    WHERE source_listing_id = @source_listing_id
    FOR UPDATE
), current_period AS MATERIALIZED (
    SELECT
        source_listing_price_period_id,
        price_state_hash
    FROM origin.source_listing_price_periods
    WHERE source_listing_id = (SELECT source_listing_id FROM locked_source)
        AND superseded_at IS NULL
    FOR UPDATE
), incoming AS MATERIALIZED (
    SELECT
        md5(jsonb_build_array(
            sqlc.narg('asking_price')::bigint,
            sqlc.narg('debt_free_price')::bigint,
            sqlc.narg('debt_share_amount')::bigint,
            sqlc.narg('price_per_m2')::double precision,
            @currency::text
        )::text) AS price_state_hash,
        clock_timestamp() AS observed_at
), closed_period AS (
    UPDATE origin.source_listing_price_periods period
    SET superseded_at = GREATEST(period.last_observed_at, incoming.observed_at)
    FROM current_period, incoming
    WHERE period.source_listing_price_period_id = current_period.source_listing_price_period_id
        AND current_period.price_state_hash IS DISTINCT FROM incoming.price_state_hash
    RETURNING period.source_listing_price_period_id
), refreshed_period AS (
    UPDATE origin.source_listing_price_periods period
    SET last_observed_at = GREATEST(period.last_observed_at, incoming.observed_at),
        source_payload_hash = COALESCE(sqlc.narg('source_payload_hash')::text, period.source_payload_hash),
        parser_version = @parser_version::integer
    FROM current_period, incoming
    WHERE period.source_listing_price_period_id = current_period.source_listing_price_period_id
        AND current_period.price_state_hash = incoming.price_state_hash
    RETURNING period.source_listing_price_period_id
), inserted_period AS (
    INSERT INTO origin.source_listing_price_periods (
        source_listing_id,
        asking_price,
        debt_free_price,
        debt_share_amount,
        price_per_m2,
        currency,
        price_state_hash,
        first_observed_at,
        last_observed_at,
        source_payload_hash,
        parser_version,
        observation_method
    )
    SELECT
        @source_listing_id,
        sqlc.narg('asking_price')::bigint,
        sqlc.narg('debt_free_price')::bigint,
        sqlc.narg('debt_share_amount')::bigint,
        sqlc.narg('price_per_m2')::double precision,
        @currency::text,
        incoming.price_state_hash,
        incoming.observed_at,
        incoming.observed_at,
        sqlc.narg('source_payload_hash')::text,
        @parser_version::integer,
        @observation_method::text
    FROM incoming
    WHERE NOT EXISTS (SELECT 1 FROM current_period)
        OR EXISTS (SELECT 1 FROM closed_period)
    RETURNING source_listing_price_period_id
)
SELECT
    COALESCE(
        (SELECT source_listing_price_period_id FROM inserted_period),
        (SELECT source_listing_price_period_id FROM refreshed_period)
    )::uuid AS source_listing_price_period_id,
    EXISTS (SELECT 1 FROM inserted_period) AS price_changed;

-- name: ListSourceListingPriceHistory :many
SELECT
    source_listing_price_period_id,
    source_listing_id,
    asking_price,
    debt_free_price,
    debt_share_amount,
    price_per_m2,
    currency,
    price_state_hash,
    first_observed_at,
    last_observed_at,
    superseded_at,
    source_payload_hash,
    parser_version,
    observation_method
FROM origin.source_listing_price_periods
WHERE source_listing_id = @source_listing_id
ORDER BY first_observed_at DESC, source_listing_price_period_id DESC;
