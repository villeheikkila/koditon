CREATE TABLE IF NOT EXISTS public.price_links (
    price_link_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    prices_transaction_id uuid NOT NULL REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE CASCADE,
    link_status text NOT NULL,
    link_method text NOT NULL,
    link_score integer NOT NULL,
    link_reasons jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (target_type = ANY (ARRAY['listing','source_listing','source_building_announcement','building','housing_company'])),
    CHECK (link_status = ANY (ARRAY['confirmed','candidate','rejected','superseded'])),
    CHECK (link_method = ANY (ARRAY['sync_auto','source_match_auto','document_match_auto','manual','backfill_auto']))
);
CREATE UNIQUE INDEX IF NOT EXISTS price_links_unique_target_transaction
ON public.price_links (target_type, target_id, prices_transaction_id);
CREATE UNIQUE INDEX IF NOT EXISTS price_links_one_confirmed_listing_per_transaction
ON public.price_links (prices_transaction_id)
WHERE target_type = 'listing'
  AND link_status = 'confirmed';
CREATE INDEX IF NOT EXISTS idx_price_links_target
ON public.price_links (target_type, target_id, link_status);
CREATE INDEX IF NOT EXISTS idx_price_links_transaction
ON public.price_links (prices_transaction_id, link_status);
WITH ranked_links AS (
    SELECT
        pot.property_offering_transaction_id,
        pot.property_offering_id,
        pot.prices_transaction_id,
        pot.property_offering_transaction_link_status,
        pot.property_offering_transaction_link_method,
        pot.property_offering_transaction_link_score,
        pot.property_offering_transaction_link_reasons,
        pot.property_offering_transaction_created_at,
        pot.property_offering_transaction_updated_at,
        row_number() OVER (
            PARTITION BY pot.prices_transaction_id
            ORDER BY pot.property_offering_transaction_link_score DESC, pot.property_offering_transaction_updated_at DESC, pot.property_offering_transaction_id
        ) AS confirmed_rank
    FROM public.property_offering_transactions pot
    WHERE pot.property_offering_transaction_link_status = ANY (ARRAY['confirmed','auto_linked','manual_linked'])
)
INSERT INTO public.price_links (
    price_link_id,
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
    property_offering_transaction_id,
    'listing',
    property_offering_id,
    prices_transaction_id,
    CASE WHEN confirmed_rank = 1 THEN 'confirmed' ELSE 'candidate' END,
    property_offering_transaction_link_method,
    property_offering_transaction_link_score,
    property_offering_transaction_link_reasons,
    property_offering_transaction_created_at,
    property_offering_transaction_updated_at
FROM ranked_links
ON CONFLICT (target_type, target_id, prices_transaction_id) DO UPDATE SET
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = EXCLUDED.link_reasons,
    updated_at = now();
INSERT INTO public.price_links (
    price_link_id,
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
    pot.property_offering_transaction_id,
    'listing',
    pot.property_offering_id,
    pot.prices_transaction_id,
    CASE
        WHEN pot.property_offering_transaction_link_status = 'rejected' THEN 'rejected'
        WHEN pot.property_offering_transaction_link_status = 'superseded' THEN 'superseded'
        ELSE 'candidate'
    END,
    pot.property_offering_transaction_link_method,
    pot.property_offering_transaction_link_score,
    pot.property_offering_transaction_link_reasons,
    pot.property_offering_transaction_created_at,
    pot.property_offering_transaction_updated_at
FROM public.property_offering_transactions pot
WHERE pot.property_offering_transaction_link_status <> ALL (ARRAY['confirmed','auto_linked','manual_linked'])
ON CONFLICT (target_type, target_id, prices_transaction_id) DO UPDATE SET
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = EXCLUDED.link_reasons,
    updated_at = now();
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
    sl.sale_listing_id,
    sl.prices_transaction_id,
    CASE
        WHEN sl.sale_listing_prices_match_status = 'rejected' THEN 'rejected'
        WHEN sl.sale_listing_prices_match_status = ANY (ARRAY['needs_review','pending','deferred','expired','noop']) THEN 'candidate'
        ELSE 'confirmed'
    END,
    'sync_auto',
    100,
    jsonb_build_object('source', 'property_source_offerings.prices_transaction_id', 'old_status', sl.sale_listing_prices_match_status),
    sl.sale_listing_created_at,
    sl.sale_listing_updated_at
FROM public.property_source_offerings sl
WHERE sl.prices_transaction_id IS NOT NULL
ON CONFLICT (target_type, target_id, prices_transaction_id) DO UPDATE SET
    link_status = EXCLUDED.link_status,
    link_method = EXCLUDED.link_method,
    link_score = EXCLUDED.link_score,
    link_reasons = EXCLUDED.link_reasons,
    updated_at = now();
CREATE TABLE IF NOT EXISTS public.target_observations (
    target_observation_id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    target_type text NOT NULL,
    target_id uuid NOT NULL,
    observation_key text NOT NULL,
    observation_kind text NOT NULL,
    severity text NOT NULL,
    direction text NOT NULL,
    value jsonb,
    text text,
    confidence double precision NOT NULL,
    source_type text NOT NULL,
    source_id uuid NOT NULL,
    evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    superseded_at timestamptz,
    CHECK (target_type = ANY (ARRAY['listing','unit','building','housing_company','house'])),
    CHECK (observation_kind = ANY (ARRAY['risk','opportunity','inconsistency','summary','valuation_note'])),
    CHECK (source_type = ANY (ARRAY['source_listing','source_housing_company','document','price_transaction','dimension_claim','manual'])),
    CHECK (confidence >= 0 AND confidence <= 1)
);
CREATE UNIQUE INDEX IF NOT EXISTS target_observations_active_unique
ON public.target_observations (target_type, target_id, observation_key, source_type, source_id)
WHERE superseded_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_observations_target
ON public.target_observations (target_type, target_id, observation_kind, severity)
WHERE superseded_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_target_observations_source
ON public.target_observations (source_type, source_id)
WHERE superseded_at IS NULL;
INSERT INTO public.target_observations (
    target_type,
    target_id,
    observation_key,
    observation_kind,
    severity,
    direction,
    value,
    text,
    confidence,
    source_type,
    source_id,
    evidence,
    created_at
)
SELECT
    'listing',
    pos.property_offering_id,
    insight.property_source_offering_insight_key,
    'valuation_note',
    insight.property_source_offering_insight_severity,
    insight.property_source_offering_insight_direction,
    to_jsonb(insight.property_source_offering_insight_value),
    insight.property_source_offering_insight_text,
    LEAST(GREATEST(insight.property_source_offering_insight_confidence::double precision / 100.0, 0), 1),
    'source_listing',
    insight.sale_listing_id,
    jsonb_build_object('source_field', insight.property_source_offering_insight_source_field),
    insight.property_source_offering_insight_created_at
FROM public.property_source_offering_insights insight
JOIN public.property_offering_sources pos ON pos.sale_listing_id = insight.sale_listing_id
    AND pos.property_offering_source_link_status <> 'rejected'
ON CONFLICT (target_type, target_id, observation_key, source_type, source_id) WHERE superseded_at IS NULL DO UPDATE SET
    severity = EXCLUDED.severity,
    direction = EXCLUDED.direction,
    value = EXCLUDED.value,
    text = EXCLUDED.text,
    confidence = EXCLUDED.confidence,
    evidence = EXCLUDED.evidence;
CREATE OR REPLACE VIEW public.source_listings AS
SELECT
    sl.sale_listing_id AS source_listing_id,
    sl.sale_listing_source_provider AS provider,
    sl.sale_listing_source_kind AS source_kind,
    sl.sale_listing_native_id AS native_id,
    sl.sale_listing_canonical_id AS canonical_source_id,
    CASE
        WHEN sl.shortcut_ad_id IS NOT NULL THEN 'shortcut_ads'
        WHEN sl.frontdoor_ad_id IS NOT NULL THEN 'frontdoor_ads'
        WHEN sl.frontdoor_building_announcement_id IS NOT NULL THEN 'frontdoor_building_announcements'
        WHEN sl.prices_transaction_id IS NOT NULL THEN 'prices_transactions'
        ELSE 'property_source_offerings'
    END AS raw_table,
    COALESCE(sl.shortcut_ad_id::text, sl.frontdoor_ad_id::text, sl.frontdoor_building_announcement_id::text, sl.prices_transaction_id::text, sl.sale_listing_id::text) AS raw_id,
    sl.sale_listing_url AS url,
    COALESCE(sa.shortcut_ad_data_hash, fa.frontdoor_ad_data_hash) AS payload_hash,
    GREATEST(COALESCE(sa.shortcut_ad_data_normalized_version, 0), COALESCE(fa.frontdoor_ad_data_normalized_version, 0)) AS normalized_version,
    sl.sale_listing_updated_at AS normalized_at,
    sl.sale_listing_first_seen_at AS first_seen_at,
    sl.sale_listing_last_seen_at AS last_seen_at,
    sl.sale_listing_created_at AS created_at,
    sl.sale_listing_updated_at AS updated_at
FROM public.property_source_offerings sl
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id;
CREATE OR REPLACE VIEW public.listings AS
SELECT
    property_offering_id AS listing_id,
    property_offering_type AS listing_type,
    property_offering_status AS listing_status,
    primary_sale_listing_id AS primary_source_listing_id,
    property_unit_id AS unit_id,
    property_house_id AS house_id,
    property_offering_first_seen_at AS first_seen_at,
    property_offering_last_seen_at AS last_seen_at,
    property_offering_created_at AS created_at,
    property_offering_updated_at AS updated_at
FROM public.property_offerings;
CREATE OR REPLACE VIEW public.units AS
SELECT
    property_unit_id AS unit_id,
    housing_company_id,
    physical_building_id,
    property_unit_identity_key AS identity_key,
    property_unit_address_norm AS address_norm,
    NULL::text AS apartment,
    property_unit_floor_level AS floor_level,
    property_unit_area_value AS area_m2,
    property_unit_room_layout AS room_layout,
    property_unit_created_at AS created_at,
    property_unit_updated_at AS updated_at
FROM public.property_units;
CREATE OR REPLACE VIEW public.houses AS
SELECT
    property_house_id AS house_id,
    property_house_identity_key AS identity_key,
    property_house_address_norm AS address_norm,
    property_house_postal_norm AS postal_norm,
    property_house_city_norm AS city_norm,
    property_house_latitude AS latitude,
    property_house_longitude AS longitude,
    property_house_created_at AS created_at,
    property_house_updated_at AS updated_at
FROM public.property_houses;
CREATE OR REPLACE VIEW public.source_housing_companies AS
SELECT
    housing_company_source_id AS source_housing_company_id,
    housing_company_source_provider AS provider,
    housing_company_source_kind AS source_kind,
    NULLIF(housing_company_source_external_id, '') AS native_id,
    housing_company_source_table AS raw_table,
    housing_company_source_id_value AS raw_id,
    housing_company_source_url AS url,
    housing_company_source_first_seen_at AS first_seen_at,
    housing_company_source_last_seen_at AS last_seen_at,
    housing_company_source_created_at AS created_at,
    housing_company_source_updated_at AS updated_at
FROM public.housing_company_sources;
CREATE OR REPLACE VIEW public.target_sources AS
SELECT
    property_target_source_id AS target_source_id,
    CASE WHEN target_type = 'offering' THEN 'listing' ELSE target_type END AS target_type,
    target_id,
    CASE
        WHEN source_table = 'property_source_offerings' THEN 'source_listing'
        WHEN source_table = 'housing_company_sources' THEN 'source_housing_company'
        WHEN source_table = 'property_documents' THEN 'document'
        WHEN source_table = 'prices_transactions' THEN 'price_transaction'
        WHEN source_provider = 'manual' THEN 'manual'
        ELSE 'source_listing'
    END AS source_type,
    source_id,
    link_status,
    link_method,
    link_score,
    link_reasons,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
FROM public.property_target_sources
WHERE source_id IS NOT NULL;
CREATE OR REPLACE VIEW public.dimension_claims AS
SELECT * FROM public.property_dimension_claims;
CREATE OR REPLACE VIEW public.dimension_values AS
SELECT * FROM public.property_dimension_values;
CREATE OR REPLACE VIEW public.dimension_profiles AS
SELECT * FROM public.property_dimension_profiles;
