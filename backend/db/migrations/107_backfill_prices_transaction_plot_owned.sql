WITH normalized AS (
    SELECT
        prices_transaction_id,
        CASE NULLIF(trim(BOTH '_' FROM regexp_replace(lower(trim(COALESCE(prices_transaction_plot, ''))), '[^[:alnum:]åäö]+', '_', 'g')), '')
            WHEN '1' THEN true
            WHEN 'oma' THEN true
            WHEN 'own' THEN true
            WHEN 'owned' THEN true
            WHEN 'omistus' THEN true
            WHEN 'omistettu' THEN true
            WHEN '2' THEN false
            WHEN '3' THEN false
            WHEN 'vuokra' THEN false
            WHEN 'rent' THEN false
            WHEN 'rented' THEN false
            WHEN 'rental' THEN false
            WHEN 'lease' THEN false
            WHEN 'leased' THEN false
            WHEN 'vuokralla' THEN false
            WHEN 'vuokratontti' THEN false
            WHEN 'optional_rental' THEN false
            WHEN 'valinnainen_vuokratontti' THEN false
            ELSE NULL
        END AS plot_owned
    FROM public.prices_transactions
)
UPDATE public.prices_transactions pt
SET prices_transaction_plot_owned = normalized.plot_owned
FROM normalized
WHERE pt.prices_transaction_id = normalized.prices_transaction_id
    AND pt.prices_transaction_plot_owned IS DISTINCT FROM normalized.plot_owned;
