DROP INDEX IF EXISTS public.target_sources_active_source_listing;
CREATE UNIQUE INDEX target_sources_active_source_listing ON public.target_sources USING btree (source_id)
WHERE target_type = 'listing'
    AND source_type = 'source_listing'
    AND link_status = 'confirmed';

DROP INDEX IF EXISTS public.idx_source_listing_match_candidates_active_pair_method;
CREATE UNIQUE INDEX idx_source_listing_match_candidates_active_pair_method ON public.source_listing_match_candidates USING btree (source_listing_id_a, source_listing_id_b, match_method)
WHERE match_status IN ('proposed', 'accepted');

ALTER TABLE public.source_listing_match_candidates
    DROP CONSTRAINT IF EXISTS source_listing_match_candidates_method_check;
ALTER TABLE public.source_listing_match_candidates
    ADD CONSTRAINT source_listing_match_candidates_method_check CHECK (match_method IN ('exact_provider_neutral_unit_v1', 'address_missing_stair_one_to_one_v1', 'frontdoor_removed_ad_announcement_v1'));
ALTER TABLE public.source_listing_match_candidates
    ADD COLUMN IF NOT EXISTS evaluation_version text DEFAULT 'source_listing_match_v2' NOT NULL;
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE connamespace = 'public'::regnamespace
            AND conrelid = 'public.source_listing_match_candidates'::regclass
            AND conname = 'source_listing_match_candidates_score_check'
    ) THEN
        ALTER TABLE public.source_listing_match_candidates
            ADD CONSTRAINT source_listing_match_candidates_score_check CHECK (match_score BETWEEN 0 AND 100);
    END IF;
END $$;

WITH evidence_links AS (
    SELECT
        entity_evidence.listing_id AS target_id,
        source_listing.source_listing_id AS source_id,
        entity_evidence.link_method,
        round(entity_evidence.confidence * 100)::integer AS link_score,
        entity_evidence.reasons AS link_reasons,
        source_listing.first_seen_at,
        source_listing.last_seen_at,
        entity_evidence.created_at
    FROM public.entity_evidence entity_evidence
    JOIN public.evidence_sources evidence ON evidence.evidence_source_id = entity_evidence.evidence_source_id
    JOIN origin.source_listings source_listing ON source_listing.frontdoor_ad_id = evidence.frontdoor_ad_id
        OR source_listing.shortcut_ad_id = evidence.shortcut_ad_id
        OR source_listing.frontdoor_building_announcement_id = evidence.frontdoor_building_announcement_id
    JOIN public.listings listing ON listing.listing_id = entity_evidence.listing_id
    WHERE entity_evidence.listing_id IS NOT NULL
        AND entity_evidence.link_status = 'confirmed'
),
primary_links AS (
    SELECT
        listing.listing_id AS target_id,
        source_listing.source_listing_id AS source_id,
        'backfill_auto'::text AS link_method,
        100 AS link_score,
        jsonb_build_object('method', 'primary_source_listing_backfill') AS link_reasons,
        source_listing.first_seen_at,
        source_listing.last_seen_at,
        listing.created_at
    FROM public.listings listing
    JOIN origin.source_listings source_listing ON source_listing.source_listing_id = listing.primary_source_listing_id
),
ranked_links AS (
    SELECT
        links.*,
        row_number() OVER (
            PARTITION BY links.source_id
            ORDER BY
                (links.link_method = 'manual') DESC,
                links.link_score DESC,
                links.created_at,
                links.target_id
        ) AS source_rank
    FROM (
        SELECT * FROM evidence_links
        UNION ALL
        SELECT * FROM primary_links
    ) links
)
INSERT INTO public.target_sources (
    target_type,
    target_id,
    source_type,
    source_id,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at
)
SELECT
    'listing',
    ranked_links.target_id,
    'source_listing',
    ranked_links.source_id,
    'confirmed',
    CASE WHEN ranked_links.link_method = 'manual' THEN 'manual' ELSE 'backfill_auto' END,
    ranked_links.link_score,
    ranked_links.link_reasons || jsonb_build_object('original_link_method', ranked_links.link_method),
    ranked_links.first_seen_at,
    ranked_links.last_seen_at
FROM ranked_links
WHERE ranked_links.source_rank = 1
    AND NOT EXISTS (
        SELECT 1
        FROM public.target_sources existing
        WHERE existing.target_type = 'listing'
            AND existing.source_type = 'source_listing'
            AND existing.source_id = ranked_links.source_id
            AND existing.link_status = 'confirmed'
    )
ON CONFLICT (target_type, target_id, source_type, source_id) DO NOTHING;
