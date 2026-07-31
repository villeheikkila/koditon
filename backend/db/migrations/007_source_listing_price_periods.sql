CREATE TABLE origin.source_listing_price_periods (
    source_listing_price_period_id uuid DEFAULT gen_random_uuid() NOT NULL,
    source_listing_id uuid NOT NULL,
    asking_price bigint,
    debt_free_price bigint,
    debt_share_amount bigint,
    price_per_m2 double precision,
    currency text DEFAULT 'EUR' NOT NULL,
    price_state_hash text NOT NULL,
    first_observed_at timestamp with time zone DEFAULT now() NOT NULL,
    last_observed_at timestamp with time zone DEFAULT now() NOT NULL,
    superseded_at timestamp with time zone,
    source_payload_hash text,
    parser_version integer DEFAULT 1 NOT NULL,
    observation_method text DEFAULT 'sync' NOT NULL,
    CONSTRAINT source_listing_price_periods_pkey PRIMARY KEY (source_listing_price_period_id),
    CONSTRAINT source_listing_price_periods_source_listing_id_fkey FOREIGN KEY (source_listing_id) REFERENCES origin.source_listings(source_listing_id) ON DELETE CASCADE,
    CONSTRAINT source_listing_price_periods_currency_check CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT source_listing_price_periods_amounts_check CHECK (
        (asking_price IS NULL OR asking_price >= 0)
        AND (debt_free_price IS NULL OR debt_free_price >= 0)
        AND (debt_share_amount IS NULL OR debt_share_amount >= 0)
        AND (price_per_m2 IS NULL OR price_per_m2 >= 0)
    ),
    CONSTRAINT source_listing_price_periods_observation_method_check CHECK (observation_method IN ('sync', 'backfill')),
    CONSTRAINT source_listing_price_periods_observed_range_check CHECK (last_observed_at >= first_observed_at),
    CONSTRAINT source_listing_price_periods_superseded_range_check CHECK (superseded_at IS NULL OR superseded_at >= last_observed_at)
);

CREATE UNIQUE INDEX source_listing_price_periods_current_source ON origin.source_listing_price_periods (source_listing_id)
WHERE superseded_at IS NULL;
CREATE INDEX source_listing_price_periods_source_history ON origin.source_listing_price_periods (source_listing_id, first_observed_at DESC);

ALTER TABLE origin.frontdoor_building_announcements
    ADD COLUMN frontdoor_building_announcement_identity_key text;
UPDATE origin.frontdoor_building_announcements
SET frontdoor_building_announcement_identity_key = 'legacy:' || frontdoor_building_announcement_id::text;
WITH stable_candidates AS (
    SELECT
        frontdoor_building_announcement_id,
        CASE
            WHEN frontdoor_building_announcement_external_id IS NOT NULL
                THEN concat_ws(':', 'external', frontdoor_building_announcement_external_id::text)
            WHEN NULLIF(trim(frontdoor_building_announcement_friendly_id), '') IS NOT NULL
                THEN 'friendly:' || trim(frontdoor_building_announcement_friendly_id)
            ELSE concat_ws(
                ':',
                'fallback',
                frontdoor_building_id::text,
                COALESCE(frontdoor_building_announcement_rental_unique_no::text, 'none'),
                lower(trim(COALESCE(frontdoor_building_announcement_address_line1, ''))),
                lower(trim(COALESCE(frontdoor_building_announcement_address_line2, '')))
            )
        END AS stable_key,
        row_number() OVER (
            PARTITION BY CASE
                WHEN frontdoor_building_announcement_external_id IS NOT NULL
                    THEN concat_ws(':', 'external', frontdoor_building_announcement_external_id::text)
                WHEN NULLIF(trim(frontdoor_building_announcement_friendly_id), '') IS NOT NULL
                    THEN 'friendly:' || trim(frontdoor_building_announcement_friendly_id)
                ELSE concat_ws(
                    ':',
                    'fallback',
                    frontdoor_building_id::text,
                    COALESCE(frontdoor_building_announcement_rental_unique_no::text, 'none'),
                    lower(trim(COALESCE(frontdoor_building_announcement_address_line1, ''))),
                    lower(trim(COALESCE(frontdoor_building_announcement_address_line2, '')))
                )
            END
            ORDER BY frontdoor_building_announcement_last_seen_at DESC, frontdoor_building_announcement_id
        ) AS stable_rank
    FROM origin.frontdoor_building_announcements
)
UPDATE origin.frontdoor_building_announcements announcement
SET frontdoor_building_announcement_identity_key = stable_candidates.stable_key
FROM stable_candidates
WHERE stable_candidates.frontdoor_building_announcement_id = announcement.frontdoor_building_announcement_id
    AND stable_candidates.stable_rank = 1;
ALTER TABLE origin.frontdoor_building_announcements
    ALTER COLUMN frontdoor_building_announcement_identity_key SET NOT NULL;
DROP INDEX IF EXISTS origin.frontdoor_building_announcements_ext_id_unpub_time_price_key;
CREATE UNIQUE INDEX frontdoor_building_announcements_identity_key ON origin.frontdoor_building_announcements (frontdoor_building_announcement_identity_key);
CREATE INDEX frontdoor_building_announcements_external_unpublishing ON origin.frontdoor_building_announcements (frontdoor_building_announcement_external_id, frontdoor_building_announcement_unpublishing_time);

WITH normalized_versions AS (
    SELECT
        keeper_source.source_listing_id,
        version.frontdoor_building_announcement_id,
        version.frontdoor_building_announcement_search_price::bigint AS asking_price,
        version.frontdoor_building_announcement_price_per_square AS price_per_m2,
        md5(jsonb_build_array(version.frontdoor_building_announcement_search_price::bigint, NULL, NULL, version.frontdoor_building_announcement_price_per_square, 'EUR')::text) AS price_state_hash,
        version.frontdoor_building_announcement_first_seen_at AS first_observed_at,
        GREATEST(version.frontdoor_building_announcement_first_seen_at, version.frontdoor_building_announcement_last_seen_at) AS last_observed_at
    FROM origin.frontdoor_building_announcements keeper
    JOIN origin.source_listings keeper_source ON keeper_source.frontdoor_building_announcement_id = keeper.frontdoor_building_announcement_id
    JOIN origin.frontdoor_building_announcements version
        ON CASE
            WHEN version.frontdoor_building_announcement_external_id IS NOT NULL
                THEN concat_ws(':', 'external', version.frontdoor_building_announcement_external_id::text)
            WHEN NULLIF(trim(version.frontdoor_building_announcement_friendly_id), '') IS NOT NULL
                THEN 'friendly:' || trim(version.frontdoor_building_announcement_friendly_id)
            ELSE concat_ws(
                ':',
                'fallback',
                version.frontdoor_building_id::text,
                COALESCE(version.frontdoor_building_announcement_rental_unique_no::text, 'none'),
                lower(trim(COALESCE(version.frontdoor_building_announcement_address_line1, ''))),
                lower(trim(COALESCE(version.frontdoor_building_announcement_address_line2, '')))
            )
        END = keeper.frontdoor_building_announcement_identity_key
    WHERE keeper.frontdoor_building_announcement_identity_key NOT LIKE 'legacy:%'
), state_boundaries AS (
    SELECT
        normalized_versions.*,
        CASE WHEN lag(price_state_hash) OVER (
            PARTITION BY source_listing_id
            ORDER BY first_observed_at, last_observed_at, frontdoor_building_announcement_id
        ) IS DISTINCT FROM price_state_hash THEN 1 ELSE 0 END AS starts_period
    FROM normalized_versions
), grouped_versions AS (
    SELECT
        state_boundaries.*,
        sum(starts_period) OVER (
            PARTITION BY source_listing_id
            ORDER BY first_observed_at, last_observed_at, frontdoor_building_announcement_id
        ) AS period_number
    FROM state_boundaries
), grouped_periods AS (
    SELECT
        source_listing_id,
        period_number,
        (array_agg(asking_price ORDER BY first_observed_at DESC, last_observed_at DESC, frontdoor_building_announcement_id DESC))[1] AS asking_price,
        (array_agg(price_per_m2 ORDER BY first_observed_at DESC, last_observed_at DESC, frontdoor_building_announcement_id DESC))[1] AS price_per_m2,
        (array_agg(price_state_hash ORDER BY first_observed_at DESC, last_observed_at DESC, frontdoor_building_announcement_id DESC))[1] AS price_state_hash,
        min(first_observed_at) AS first_observed_at,
        max(last_observed_at) AS last_observed_at
    FROM grouped_versions
    GROUP BY source_listing_id, period_number
), bounded_periods AS (
    SELECT
        grouped_periods.*,
        lead(first_observed_at) OVER (PARTITION BY source_listing_id ORDER BY period_number) AS next_period_at
    FROM grouped_periods
)
INSERT INTO origin.source_listing_price_periods (
    source_listing_id,
    asking_price,
    price_per_m2,
    currency,
    price_state_hash,
    first_observed_at,
    last_observed_at,
    superseded_at,
    parser_version,
    observation_method
)
SELECT
    source_listing_id,
    asking_price,
    price_per_m2,
    'EUR',
    price_state_hash,
    first_observed_at,
    last_observed_at,
    CASE WHEN next_period_at IS NULL THEN NULL ELSE GREATEST(last_observed_at, next_period_at) END,
    1,
    'backfill'
FROM bounded_periods;

INSERT INTO origin.source_listing_price_periods (
    source_listing_id,
    asking_price,
    debt_free_price,
    currency,
    price_state_hash,
    first_observed_at,
    last_observed_at,
    source_payload_hash,
    parser_version,
    observation_method
)
SELECT
    facts.source_listing_id,
    CASE WHEN facts.asking_price >= 0 THEN facts.asking_price END,
    CASE WHEN facts.debt_free_price >= 0 THEN facts.debt_free_price END,
    'EUR',
    md5(jsonb_build_array(CASE WHEN facts.asking_price >= 0 THEN facts.asking_price END, CASE WHEN facts.debt_free_price >= 0 THEN facts.debt_free_price END, NULL, NULL, 'EUR')::text),
    facts.refreshed_at,
    facts.refreshed_at,
    source_listing.payload_hash,
    1,
    'backfill'
FROM public.source_listing_match_facts facts
JOIN origin.source_listings source_listing ON source_listing.source_listing_id = facts.source_listing_id
WHERE (facts.asking_price >= 0 OR facts.debt_free_price >= 0)
    AND source_listing.frontdoor_building_announcement_id IS NULL
ON CONFLICT (source_listing_id) WHERE superseded_at IS NULL DO NOTHING;

WITH legacy_sources AS (
    SELECT source_listing.source_listing_id
    FROM origin.source_listings source_listing
    JOIN origin.frontdoor_building_announcements announcement
        ON announcement.frontdoor_building_announcement_id = source_listing.frontdoor_building_announcement_id
    WHERE announcement.frontdoor_building_announcement_identity_key LIKE 'legacy:%'
), legacy_targets AS (
    SELECT DISTINCT target_source.target_id
    FROM public.target_sources target_source
    JOIN legacy_sources ON legacy_sources.source_listing_id = target_source.source_id
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.link_status = 'confirmed'
        AND NOT EXISTS (
            SELECT 1
            FROM public.target_sources other_link
            WHERE other_link.target_type = 'listing'
                AND other_link.target_id = target_source.target_id
                AND other_link.source_type = 'source_listing'
                AND other_link.link_status = 'confirmed'
                AND other_link.source_id NOT IN (SELECT source_listing_id FROM legacy_sources)
        )
)
UPDATE public.listings listing
SET listing_status = 'removed', updated_at = now()
WHERE listing.listing_id IN (SELECT target_id FROM legacy_targets);

WITH legacy_sources AS (
    SELECT source_listing.source_listing_id
    FROM origin.source_listings source_listing
    JOIN origin.frontdoor_building_announcements announcement
        ON announcement.frontdoor_building_announcement_id = source_listing.frontdoor_building_announcement_id
    WHERE announcement.frontdoor_building_announcement_identity_key LIKE 'legacy:%'
), legacy_targets AS (
    SELECT DISTINCT target_source.target_id
    FROM public.target_sources target_source
    JOIN legacy_sources ON legacy_sources.source_listing_id = target_source.source_id
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.link_status = 'confirmed'
        AND NOT EXISTS (
            SELECT 1
            FROM public.target_sources other_link
            WHERE other_link.target_type = 'listing'
                AND other_link.target_id = target_source.target_id
                AND other_link.source_type = 'source_listing'
                AND other_link.link_status = 'confirmed'
                AND other_link.source_id NOT IN (SELECT source_listing_id FROM legacy_sources)
        )
)
UPDATE public.property_offerings offering
SET property_offering_status = 'removed', property_offering_updated_at = now()
WHERE offering.property_offering_id IN (SELECT target_id FROM legacy_targets);

WITH legacy_sources AS (
    SELECT source_listing.source_listing_id
    FROM origin.source_listings source_listing
    JOIN origin.frontdoor_building_announcements announcement
        ON announcement.frontdoor_building_announcement_id = source_listing.frontdoor_building_announcement_id
    WHERE announcement.frontdoor_building_announcement_identity_key LIKE 'legacy:%'
), legacy_targets AS (
    SELECT DISTINCT target_source.target_id
    FROM public.target_sources target_source
    JOIN legacy_sources ON legacy_sources.source_listing_id = target_source.source_id
    WHERE target_source.target_type = 'listing'
        AND target_source.source_type = 'source_listing'
        AND target_source.link_status = 'confirmed'
        AND NOT EXISTS (
            SELECT 1
            FROM public.target_sources other_link
            WHERE other_link.target_type = 'listing'
                AND other_link.target_id = target_source.target_id
                AND other_link.source_type = 'source_listing'
                AND other_link.link_status = 'confirmed'
                AND other_link.source_id NOT IN (SELECT source_listing_id FROM legacy_sources)
        )
)
UPDATE public.listing_search_documents document
SET listing_status = 'removed', refreshed_at = now()
WHERE document.listing_id IN (SELECT target_id FROM legacy_targets);

DELETE FROM public.target_sources target_source
USING origin.source_listings source_listing, origin.frontdoor_building_announcements announcement
WHERE target_source.source_type = 'source_listing'
    AND target_source.source_id = source_listing.source_listing_id
    AND source_listing.frontdoor_building_announcement_id = announcement.frontdoor_building_announcement_id
    AND announcement.frontdoor_building_announcement_identity_key LIKE 'legacy:%';

DELETE FROM origin.frontdoor_building_announcements
WHERE frontdoor_building_announcement_identity_key LIKE 'legacy:%';
