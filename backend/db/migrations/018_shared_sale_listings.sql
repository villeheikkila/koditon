CREATE TABLE public.sale_listings (
    sale_listing_id uuid DEFAULT gen_random_uuid() NOT NULL CONSTRAINT sale_listings_pkey PRIMARY KEY,
    sale_listing_public_id text NOT NULL CONSTRAINT sale_listings_public_id_key UNIQUE,
    shortcut_ad_id bigint CONSTRAINT sale_listings_shortcut_ad_id_fkey REFERENCES public.shortcut_ads(shortcut_ad_id) ON DELETE SET NULL,
    frontdoor_ad_id uuid CONSTRAINT sale_listings_frontdoor_ad_id_fkey REFERENCES public.frontdoor_ads(frontdoor_ad_id) ON DELETE SET NULL,
    frontdoor_building_announcement_id uuid CONSTRAINT sale_listings_frontdoor_building_announcement_id_fkey REFERENCES public.frontdoor_building_announcements(frontdoor_building_announcement_id) ON DELETE SET NULL,
    prices_transaction_id uuid CONSTRAINT sale_listings_prices_transaction_id_fkey REFERENCES public.prices_transactions(prices_transaction_id) ON DELETE SET NULL,
    sale_listing_source_provider text NOT NULL,
    sale_listing_source_kind text NOT NULL,
    sale_listing_native_id text NOT NULL,
    sale_listing_canonical_id text NOT NULL CONSTRAINT sale_listings_canonical_id_key UNIQUE,
    sale_listing_url text,
    sale_listing_headline text NOT NULL,
    sale_listing_street_address text,
    sale_listing_city text,
    sale_listing_postal text,
    sale_listing_asking_price bigint,
    sale_listing_area_value double precision,
    sale_listing_room_layout text,
    sale_listing_last_seen_at timestamp with time zone,
    sale_listing_published_at timestamp with time zone,
    sale_listing_search_text text,
    sale_listing_created_at timestamp with time zone DEFAULT now() NOT NULL,
    sale_listing_updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT sale_listings_source_provider_check CHECK (sale_listing_source_provider = ANY (ARRAY['shortcut'::text, 'frontdoor'::text])),
    CONSTRAINT sale_listings_source_kind_check CHECK (sale_listing_source_kind = ANY (ARRAY['ad'::text, 'announcement'::text])),
    CONSTRAINT sale_listings_has_source_check CHECK (
        shortcut_ad_id IS NOT NULL OR
        frontdoor_ad_id IS NOT NULL OR
        frontdoor_building_announcement_id IS NOT NULL
    )
);
CREATE UNIQUE INDEX sale_listings_shortcut_ad_id_key ON public.sale_listings (shortcut_ad_id) WHERE shortcut_ad_id IS NOT NULL;
CREATE UNIQUE INDEX sale_listings_frontdoor_ad_id_key ON public.sale_listings (frontdoor_ad_id) WHERE frontdoor_ad_id IS NOT NULL;
CREATE UNIQUE INDEX sale_listings_frontdoor_building_announcement_id_key ON public.sale_listings (frontdoor_building_announcement_id) WHERE frontdoor_building_announcement_id IS NOT NULL;
CREATE UNIQUE INDEX sale_listings_prices_transaction_id_key ON public.sale_listings (prices_transaction_id) WHERE prices_transaction_id IS NOT NULL;
CREATE INDEX idx_sale_listings_source ON public.sale_listings (sale_listing_source_provider, sale_listing_source_kind);
CREATE INDEX idx_sale_listings_city ON public.sale_listings (sale_listing_city);
CREATE INDEX idx_sale_listings_postal ON public.sale_listings (sale_listing_postal);
CREATE INDEX idx_sale_listings_price ON public.sale_listings (sale_listing_asking_price);
CREATE INDEX idx_sale_listings_area ON public.sale_listings (sale_listing_area_value);
CREATE INDEX idx_sale_listings_last_seen ON public.sale_listings (sale_listing_last_seen_at DESC);
CREATE INDEX idx_sale_listings_search_trgm ON public.sale_listings USING gin (lower(sale_listing_search_text) gin_trgm_ops);
INSERT INTO public.sale_listings (
    sale_listing_public_id,
    shortcut_ad_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    sale_listing_url,
    sale_listing_headline,
    sale_listing_street_address,
    sale_listing_city,
    sale_listing_postal,
    sale_listing_asking_price,
    sale_listing_area_value,
    sale_listing_room_layout,
    sale_listing_last_seen_at,
    sale_listing_published_at,
    sale_listing_search_text
)
SELECT
    'l_' || substr(md5('shortcut:ad:' || sa.shortcut_ad_id::text), 1, 16),
    sa.shortcut_ad_id,
    'shortcut',
    'ad',
    sa.shortcut_ad_id::text,
    'shortcut:ad:' || sa.shortcut_ad_id::text,
    sa.shortcut_ad_url,
    COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address, sa.shortcut_ad_id::text),
    COALESCE(sa.shortcut_ad_street_address, sb.shortcut_building_address),
    sa.shortcut_ad_city,
    sa.shortcut_ad_postal,
    sa.shortcut_ad_price,
    sa.shortcut_ad_area_value,
    sa.shortcut_ad_data #>> '{adData,roomConfiguration}',
    sa.shortcut_ad_last_seen_at,
    (sa.shortcut_ad_data #>> '{adData,published}')::timestamptz,
    concat_ws(' ', sa.shortcut_ad_search_text, sb.shortcut_building_address, sb.shortcut_building_housing_company)
FROM public.shortcut_ads sa
LEFT JOIN public.shortcut_buildings sb ON sb.shortcut_building_id = sa.shortcut_building_id
WHERE sa.shortcut_ad_type = 'listing'
ON CONFLICT (sale_listing_canonical_id) DO NOTHING;
INSERT INTO public.sale_listings (
    sale_listing_public_id,
    frontdoor_ad_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    sale_listing_url,
    sale_listing_headline,
    sale_listing_street_address,
    sale_listing_city,
    sale_listing_postal,
    sale_listing_asking_price,
    sale_listing_area_value,
    sale_listing_room_layout,
    sale_listing_last_seen_at,
    sale_listing_published_at,
    sale_listing_search_text
)
SELECT
    'l_' || substr(md5('frontdoor:ad:' || fa.frontdoor_ad_external_id), 1, 16),
    fa.frontdoor_ad_id,
    'frontdoor',
    'ad',
    fa.frontdoor_ad_external_id,
    'frontdoor:ad:' || fa.frontdoor_ad_external_id,
    fa.frontdoor_ad_url,
    COALESCE(fa.frontdoor_ad_street_address, fa.frontdoor_ad_external_id),
    fa.frontdoor_ad_street_address,
    fa.frontdoor_ad_city,
    fa.frontdoor_ad_postal,
    fa.frontdoor_ad_price,
    fa.frontdoor_ad_area_value,
    fa.frontdoor_ad_data #>> '{residenceDetailsDTO,roomStructure}',
    fa.frontdoor_ad_last_seen_at,
    fa.frontdoor_ad_publishing_time,
    fa.frontdoor_ad_search_text
FROM public.frontdoor_ads fa
ON CONFLICT (sale_listing_canonical_id) DO NOTHING;
INSERT INTO public.sale_listings (
    sale_listing_public_id,
    frontdoor_building_announcement_id,
    sale_listing_source_provider,
    sale_listing_source_kind,
    sale_listing_native_id,
    sale_listing_canonical_id,
    sale_listing_url,
    sale_listing_headline,
    sale_listing_street_address,
    sale_listing_city,
    sale_listing_postal,
    sale_listing_asking_price,
    sale_listing_area_value,
    sale_listing_room_layout,
    sale_listing_last_seen_at,
    sale_listing_published_at,
    sale_listing_search_text
)
SELECT
    'l_' || substr(md5('frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text), 1, 16),
    fba.frontdoor_building_announcement_id,
    'frontdoor',
    'announcement',
    fba.frontdoor_building_announcement_id::text,
    'frontdoor:announcement:' || fba.frontdoor_building_announcement_id::text,
    fb.frontdoor_building_url,
    COALESCE(fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_id::text),
    concat_ws(' ', fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2),
    COALESCE(fba.frontdoor_building_announcement_location, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area),
    fb.frontdoor_building_postcode,
    CASE WHEN fba.frontdoor_building_announcement_search_price IS NULL THEN NULL ELSE fba.frontdoor_building_announcement_search_price::bigint END,
    fba.frontdoor_building_announcement_area,
    fba.frontdoor_building_announcement_room_structure,
    fba.frontdoor_building_announcement_last_seen_at,
    NULL::timestamptz,
    concat_ws(' ', fba.frontdoor_building_announcement_id::text, fba.frontdoor_building_announcement_external_id::text, fba.frontdoor_building_announcement_friendly_id, fba.frontdoor_building_announcement_address_line1, fba.frontdoor_building_announcement_address_line2, fba.frontdoor_building_announcement_location, fb.frontdoor_building_postcode, fb.frontdoor_building_municipality, fb.frontdoor_building_post_area, fb.frontdoor_building_url, fba.frontdoor_building_announcement_room_structure)
FROM public.frontdoor_building_announcements fba
JOIN public.frontdoor_buildings fb ON fb.frontdoor_building_id = fba.frontdoor_building_id
WHERE fba.frontdoor_building_announcement_rent_period IS NULL
  AND fba.frontdoor_building_announcement_rental_unique_no IS NULL
ON CONFLICT (sale_listing_canonical_id) DO NOTHING;
