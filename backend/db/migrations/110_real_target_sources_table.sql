DROP VIEW IF EXISTS public.target_sources;
CREATE TABLE IF NOT EXISTS public.target_sources (
    target_source_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    link_status text NOT NULL,
    link_method text NOT NULL,
    link_score integer NOT NULL DEFAULT 0,
    link_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['listing','unit','building','housing_company','house'])),
    CHECK (source_type = ANY (ARRAY['source_listing','source_housing_company','document','price_transaction','manual'])),
    CHECK (link_status = ANY (ARRAY['confirmed','candidate','rejected','superseded'])),
    CHECK (link_method = ANY (ARRAY['sync_auto','source_match_auto','document_match_auto','manual','backfill_auto']))
);
CREATE UNIQUE INDEX IF NOT EXISTS target_sources_unique_target_source
ON public.target_sources (target_type, target_id, source_type, source_id);
CREATE INDEX IF NOT EXISTS target_sources_target
ON public.target_sources (target_type, target_id, link_status);
CREATE INDEX IF NOT EXISTS target_sources_source
ON public.target_sources (source_type, source_id, link_status);
WITH mapped AS (
    SELECT
        CASE WHEN pts.target_type = 'offering' THEN 'listing' ELSE pts.target_type END AS target_type,
        pts.target_id,
        CASE
            WHEN pts.source_table = 'property_source_offerings' THEN 'source_listing'
            WHEN pts.source_table = 'housing_company_sources' THEN 'source_housing_company'
            WHEN pts.source_table = 'property_documents' THEN 'document'
            WHEN pts.source_table = 'prices_transactions' THEN 'price_transaction'
            WHEN pts.source_provider = 'manual' THEN 'manual'
            ELSE 'source_listing'
        END AS source_type,
        pts.source_id,
        pts.link_status,
        CASE
            WHEN pts.link_method = ANY (ARRAY['sync_auto','source_match_auto','document_match_auto','manual','backfill_auto']) THEN pts.link_method
            ELSE 'backfill_auto'
        END AS link_method,
        pts.link_score,
        pts.link_reasons,
        pts.first_seen_at,
        pts.last_seen_at,
        pts.created_at,
        pts.updated_at
    FROM public.property_target_sources pts
    WHERE pts.source_id IS NOT NULL
),
ranked AS (
    SELECT
        mapped.*,
        row_number() OVER (
            PARTITION BY source_id
            ORDER BY link_score DESC, last_seen_at DESC NULLS LAST, updated_at DESC NULLS LAST
        ) AS active_source_rank
    FROM mapped
    WHERE target_type = 'listing'
        AND source_type = 'source_listing'
        AND link_status <> 'rejected'
),
backfill_rows AS (
    SELECT mapped.*
    FROM mapped
    LEFT JOIN ranked ON ranked.target_type = mapped.target_type
        AND ranked.target_id = mapped.target_id
        AND ranked.source_type = mapped.source_type
        AND ranked.source_id = mapped.source_id
        AND ranked.link_status = mapped.link_status
    WHERE mapped.target_type = ANY (ARRAY['listing','unit','building','housing_company','house'])
        AND mapped.source_type = ANY (ARRAY['source_listing','source_housing_company','document','price_transaction','manual'])
        AND mapped.link_status = ANY (ARRAY['confirmed','candidate','rejected','superseded'])
        AND (
            mapped.target_type <> 'listing'
            OR mapped.source_type <> 'source_listing'
            OR mapped.link_status = 'rejected'
            OR ranked.active_source_rank = 1
        )
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
    last_seen_at,
    created_at,
    updated_at
)
SELECT DISTINCT ON (target_type, target_id, source_type, source_id)
    target_type,
    target_id,
    source_type,
    source_id,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
FROM backfill_rows
ORDER BY target_type, target_id, source_type, source_id, link_status, last_seen_at DESC NULLS LAST
ON CONFLICT (target_type, target_id, source_type, source_id) DO UPDATE SET
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = public.target_sources.link_reasons || EXCLUDED.link_reasons,
    first_seen_at = LEAST(COALESCE(public.target_sources.first_seen_at, EXCLUDED.first_seen_at), COALESCE(EXCLUDED.first_seen_at, public.target_sources.first_seen_at)),
    last_seen_at = GREATEST(COALESCE(public.target_sources.last_seen_at, EXCLUDED.last_seen_at), COALESCE(EXCLUDED.last_seen_at, public.target_sources.last_seen_at)),
    updated_at = now();
CREATE UNIQUE INDEX IF NOT EXISTS target_sources_active_source_listing
ON public.target_sources (source_id)
WHERE target_type = 'listing'
  AND source_type = 'source_listing'
  AND link_status <> 'rejected';
