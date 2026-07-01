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
