DROP VIEW IF EXISTS public.source_listings;
CREATE TABLE public.source_listings (
    source_listing_id uuid PRIMARY KEY,
    provider text NOT NULL,
    source_kind text NOT NULL,
    native_id text NOT NULL,
    canonical_source_id text NOT NULL,
    raw_table text NOT NULL,
    raw_id text NOT NULL,
    url text,
    payload_hash text,
    normalized_version integer NOT NULL DEFAULT 0,
    normalized_at timestamptz,
    first_seen_at timestamptz,
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (provider = ANY (ARRAY['shortcut','frontdoor'])),
    CHECK (source_kind = ANY (ARRAY['ad','announcement']))
);
CREATE UNIQUE INDEX source_listings_provider_kind_native_key
ON public.source_listings (provider, source_kind, native_id);
CREATE UNIQUE INDEX source_listings_canonical_source_id_key
ON public.source_listings (canonical_source_id);
CREATE INDEX idx_source_listings_raw
ON public.source_listings (raw_table, raw_id);
CREATE INDEX idx_source_listings_last_seen
ON public.source_listings (last_seen_at DESC);
CREATE OR REPLACE FUNCTION public.fnc__sync_source_listing_from_property_source_offering()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        DELETE FROM public.source_listings WHERE source_listing_id = OLD.sale_listing_id;
        RETURN OLD;
    END IF;
    INSERT INTO public.source_listings (
        source_listing_id,
        provider,
        source_kind,
        native_id,
        canonical_source_id,
        raw_table,
        raw_id,
        url,
        payload_hash,
        normalized_version,
        normalized_at,
        first_seen_at,
        last_seen_at,
        created_at,
        updated_at
    )
    SELECT
        NEW.sale_listing_id,
        NEW.sale_listing_source_provider,
        NEW.sale_listing_source_kind,
        NEW.sale_listing_native_id,
        NEW.sale_listing_canonical_id,
        CASE
            WHEN NEW.shortcut_ad_id IS NOT NULL THEN 'shortcut_ads'
            WHEN NEW.frontdoor_ad_id IS NOT NULL THEN 'frontdoor_ads'
            WHEN NEW.frontdoor_building_announcement_id IS NOT NULL THEN 'frontdoor_building_announcements'
            WHEN NEW.prices_transaction_id IS NOT NULL THEN 'prices_transactions'
            ELSE 'property_source_offerings'
        END,
        COALESCE(NEW.shortcut_ad_id::text, NEW.frontdoor_ad_id::text, NEW.frontdoor_building_announcement_id::text, NEW.prices_transaction_id::text, NEW.sale_listing_id::text),
        NEW.sale_listing_url,
        COALESCE(sa.shortcut_ad_data_hash, fa.frontdoor_ad_data_hash),
        GREATEST(COALESCE(sa.shortcut_ad_data_normalized_version, 0), COALESCE(fa.frontdoor_ad_data_normalized_version, 0)),
        NEW.sale_listing_updated_at,
        NEW.sale_listing_first_seen_at,
        NEW.sale_listing_last_seen_at,
        NEW.sale_listing_created_at,
        NEW.sale_listing_updated_at
    FROM (SELECT NEW.shortcut_ad_id, NEW.frontdoor_ad_id) src
    LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = src.shortcut_ad_id
    LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = src.frontdoor_ad_id
    ON CONFLICT (source_listing_id) DO UPDATE SET
        provider = EXCLUDED.provider,
        source_kind = EXCLUDED.source_kind,
        native_id = EXCLUDED.native_id,
        canonical_source_id = EXCLUDED.canonical_source_id,
        raw_table = EXCLUDED.raw_table,
        raw_id = EXCLUDED.raw_id,
        url = EXCLUDED.url,
        payload_hash = EXCLUDED.payload_hash,
        normalized_version = EXCLUDED.normalized_version,
        normalized_at = EXCLUDED.normalized_at,
        first_seen_at = EXCLUDED.first_seen_at,
        last_seen_at = EXCLUDED.last_seen_at,
        updated_at = EXCLUDED.updated_at;
    RETURN NEW;
END;
$$;
INSERT INTO public.source_listings (
    source_listing_id,
    provider,
    source_kind,
    native_id,
    canonical_source_id,
    raw_table,
    raw_id,
    url,
    payload_hash,
    normalized_version,
    normalized_at,
    first_seen_at,
    last_seen_at,
    created_at,
    updated_at
)
SELECT
    sl.sale_listing_id,
    sl.sale_listing_source_provider,
    sl.sale_listing_source_kind,
    sl.sale_listing_native_id,
    sl.sale_listing_canonical_id,
    CASE
        WHEN sl.shortcut_ad_id IS NOT NULL THEN 'shortcut_ads'
        WHEN sl.frontdoor_ad_id IS NOT NULL THEN 'frontdoor_ads'
        WHEN sl.frontdoor_building_announcement_id IS NOT NULL THEN 'frontdoor_building_announcements'
        WHEN sl.prices_transaction_id IS NOT NULL THEN 'prices_transactions'
        ELSE 'property_source_offerings'
    END,
    COALESCE(sl.shortcut_ad_id::text, sl.frontdoor_ad_id::text, sl.frontdoor_building_announcement_id::text, sl.prices_transaction_id::text, sl.sale_listing_id::text),
    sl.sale_listing_url,
    COALESCE(sa.shortcut_ad_data_hash, fa.frontdoor_ad_data_hash),
    GREATEST(COALESCE(sa.shortcut_ad_data_normalized_version, 0), COALESCE(fa.frontdoor_ad_data_normalized_version, 0)),
    sl.sale_listing_updated_at,
    sl.sale_listing_first_seen_at,
    sl.sale_listing_last_seen_at,
    sl.sale_listing_created_at,
    sl.sale_listing_updated_at
FROM public.property_source_offerings sl
LEFT JOIN public.shortcut_ads sa ON sa.shortcut_ad_id = sl.shortcut_ad_id
LEFT JOIN public.frontdoor_ads fa ON fa.frontdoor_ad_id = sl.frontdoor_ad_id
ON CONFLICT (source_listing_id) DO UPDATE SET
    provider = EXCLUDED.provider,
    source_kind = EXCLUDED.source_kind,
    native_id = EXCLUDED.native_id,
    canonical_source_id = EXCLUDED.canonical_source_id,
    raw_table = EXCLUDED.raw_table,
    raw_id = EXCLUDED.raw_id,
    url = EXCLUDED.url,
    payload_hash = EXCLUDED.payload_hash,
    normalized_version = EXCLUDED.normalized_version,
    normalized_at = EXCLUDED.normalized_at,
    first_seen_at = EXCLUDED.first_seen_at,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = EXCLUDED.updated_at;
DROP TRIGGER IF EXISTS trg__sync_source_listing_from_property_source_offering ON public.property_source_offerings;
CREATE TRIGGER trg__sync_source_listing_from_property_source_offering
AFTER INSERT OR UPDATE OR DELETE ON public.property_source_offerings
FOR EACH ROW
EXECUTE FUNCTION public.fnc__sync_source_listing_from_property_source_offering();
