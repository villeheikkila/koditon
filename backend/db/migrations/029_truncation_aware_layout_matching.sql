CREATE OR REPLACE FUNCTION public.fnc__layout_match_key(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT NULLIF(regexp_replace(lower(trim(COALESCE(value, ''))), '[^[:alnum:]åäö]+', '', 'g'), '')
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_layout_is_truncated(value text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT COALESCE(value, '') ~ '(\.\.\.|…)'
$$;
CREATE OR REPLACE FUNCTION public.fnc__prices_transaction_layout_match_key(value text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT public.fnc__layout_match_key(regexp_replace(COALESCE(value, ''), '(\.\.\.|…).*$', '', 'g'))
$$;
CREATE OR REPLACE FUNCTION public.fnc__layout_prefix_match(listing_layout text, transaction_description text)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    WITH values AS (
        SELECT
            public.fnc__layout_match_key(listing_layout) AS listing_key,
            public.fnc__prices_transaction_layout_match_key(transaction_description) AS transaction_key,
            public.fnc__prices_transaction_layout_is_truncated(transaction_description) AS is_truncated
    )
    SELECT listing_key IS NOT NULL
        AND transaction_key IS NOT NULL
        AND length(transaction_key) >= 2
        AND CASE
            WHEN is_truncated THEN left(listing_key, length(transaction_key)) = transaction_key
            ELSE listing_key = transaction_key
        END
    FROM values
$$;
