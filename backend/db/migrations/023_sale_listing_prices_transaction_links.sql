DROP TRIGGER IF EXISTS trg__sync_sale_listing_from_prices_transaction ON public.prices_transactions;
DROP TRIGGER IF EXISTS trg__refresh_sale_listings_from_prices_neighborhood ON public.prices_neighborhoods;
DROP FUNCTION IF EXISTS public.fnc__sync_sale_listing_from_prices_transaction();
DROP FUNCTION IF EXISTS public.fnc__refresh_sale_listings_from_prices_neighborhood();
DROP FUNCTION IF EXISTS public.fnc__prices_transaction_rooms_count(text, text);
DROP FUNCTION IF EXISTS public.fnc__prices_transaction_floor_level(text);
DROP FUNCTION IF EXISTS public.fnc__prices_transaction_total_floors(text);
DELETE FROM public.sale_listings
WHERE sale_listing_source_provider = 'prices'
  AND sale_listing_source_kind = 'transaction';
ALTER TABLE public.sale_listings DROP CONSTRAINT IF EXISTS sale_listings_source_provider_check;
ALTER TABLE public.sale_listings
ADD CONSTRAINT sale_listings_source_provider_check
CHECK (sale_listing_source_provider = ANY (ARRAY['shortcut'::text, 'frontdoor'::text]));
ALTER TABLE public.sale_listings DROP CONSTRAINT IF EXISTS sale_listings_source_kind_check;
ALTER TABLE public.sale_listings
ADD CONSTRAINT sale_listings_source_kind_check
CHECK (sale_listing_source_kind = ANY (ARRAY['ad'::text, 'announcement'::text]));
ALTER TABLE public.sale_listings DROP CONSTRAINT IF EXISTS sale_listings_has_source_check;
ALTER TABLE public.sale_listings
ADD CONSTRAINT sale_listings_has_source_check
CHECK (
    shortcut_ad_id IS NOT NULL OR
    frontdoor_ad_id IS NOT NULL OR
    frontdoor_building_announcement_id IS NOT NULL
);
CREATE OR REPLACE FUNCTION public.fnc__link_sale_listing_prices_transaction(listing_public_id text, transaction_id uuid)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    listing_id uuid;
    existing_listing_id uuid;
BEGIN
    IF transaction_id IS NULL THEN
        RAISE EXCEPTION 'transaction_id is required';
    END IF;
    SELECT sale_listing_id INTO listing_id
    FROM public.sale_listings
    WHERE sale_listing_public_id = listing_public_id
    FOR UPDATE;
    IF listing_id IS NULL THEN
        RAISE EXCEPTION 'sale listing % not found', listing_public_id;
    END IF;
    PERFORM 1
    FROM public.prices_transactions
    WHERE prices_transaction_id = transaction_id;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'prices transaction % not found', transaction_id;
    END IF;
    SELECT sale_listing_id INTO existing_listing_id
    FROM public.sale_listings
    WHERE prices_transaction_id = transaction_id
      AND sale_listing_id <> listing_id;
    IF existing_listing_id IS NOT NULL THEN
        RAISE EXCEPTION 'prices transaction % is already linked to sale listing %', transaction_id, existing_listing_id;
    END IF;
    UPDATE public.sale_listings
    SET prices_transaction_id = transaction_id,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = listing_id;
    RETURN listing_id;
END;
$$;
CREATE OR REPLACE FUNCTION public.fnc__unlink_sale_listing_prices_transaction(listing_public_id text)
RETURNS uuid
LANGUAGE plpgsql
AS $$
DECLARE
    listing_id uuid;
BEGIN
    SELECT sale_listing_id INTO listing_id
    FROM public.sale_listings
    WHERE sale_listing_public_id = listing_public_id
    FOR UPDATE;
    IF listing_id IS NULL THEN
        RAISE EXCEPTION 'sale listing % not found', listing_public_id;
    END IF;
    UPDATE public.sale_listings
    SET prices_transaction_id = NULL,
        sale_listing_updated_at = now()
    WHERE sale_listing_id = listing_id;
    RETURN listing_id;
END;
$$;
